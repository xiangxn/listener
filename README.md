# listener — EVM 链上套利监控框架

基于 Golang 实现的 EVM 链上套利系统。通过 WebSocket 订阅多个 DEX 的 Swap 事件，实时抓取事件涉及交易对的链上价格，运行套利策略（默认「搬砖 MovingBrick」策略：同一交易对在不同池之间的价差套利），调用链上 `Trader` 合约完成交易（资金不足时自动走闪电贷），并支持 **Anvil 链上分叉模拟**、**Flashbots 私有交易**、Telegram 通知与 MongoDB 持久化。

## 一、整体架构

```
┌────────────────────────────────────────────────────────────────────┐
│                            listener (Go)                          │
│                                                                    │
│  main.go (cobra CLI: arb / stats / crypto)                        │
│    │                                                              │
│    ▼                                                              │
│  monitor.EventMonitor ──────────────┬──────────────┬───────────── │
│  │  WS 订阅 Swap 事件(多WS故障转移)   │              │             │
│  │  事件缓存/去重/批处理              │              │             │
│  │  预处理(新池/新Token入库)          │              │             │
│  │  价格批量刷新(Multicall)           │              │             │
│  │  套利判定 → 执行                  │              │             │
│  └────────┬──────────────┬──────────┼──────────────┼───────────── │
│           ▼              ▼          ▼              ▼              │
│      strategies      dex/*       simulation     database/         │
│      (MovingBrick)   (DEX适配器)   (Anvil分叉)     (MongoDB)      │
│           │                         │              │              │
│           └──────────┬──────────────┘              │              │
│                      ▼                             │              │
│                trader/Trader.sol ◄────链上执行──────┘              │
│                (Foundry 合约: 双腿Swap+闪电贷)                     │
│                      │                                             │
│                      ▼                                             │
│           直接发送 / Alchemy / Flashbots 私有交易                   │
└────────────────────────────────────────────────────────────────────┘
```

### 核心数据流

```mermaid
flowchart LR
    A[WS 订阅 Swap 事件] -->|缓存去重 map[poolAddr]Log| B{定时器 EventWaitingTime}
    B --> C[预处理<br/>新池/新Token 走 Multicall 拉取入库]
    C --> D[过滤黑名单<br/>pool/token 黑名单]
    D --> E[按事件 Token 查库获取相关池]
    E --> F[Multicall 批量刷新价格<br/>+ 区块号 + basefee + 合约余额]
    F -->|价格与事件同区块| G[策略 MovingBrick<br/>计算套利收益]
    G -->|profit > MinProfitUSD| H[TG 通知]
    H --> I{simulation.enable?}
    I -->|是| J[Anvil 分叉模拟交易<br/>部署Trader→验证收益]
    I -->|否| K[真实交易<br/>直接发送/Flashbots]
    J & K --> L[(MongoDB<br/>tokens/pools/prices/transactions)]
    L --> M[后台协程每20s确认交易<br/>回写 gas/income, 失败→黑名单]
```

## 二、目录结构

```
listener/
├── main.go                 # 入口: cobra 命令 arb(套利主程序) / stats(统计) / crypto(加解密)
├── config/
│   └── config.go           # 配置结构体与 YAML/JSON 加载
├── monitor/
│   ├── eventmonitor.go     # 核心事件监控器: 订阅/批处理/价格刷新/发单/交易确认
│   └── types.go            # monitor 内部结构定义
├── types/
│   ├── data.go             # 数据模型: Pool/Token/Pair/Transaction/Arbitrage/SwapParams
│   └── interfaces.go       # 核心接口: IMonitor(监控器) / IActions(数据库) / EventHandler(策略)
├── dex/                    # DEX 适配层(每种协议一个文件)
│   ├── base.go             # 通用: FactoryABI/TokenABI、池与Token批量拉取、事件预处理
│   ├── uniswapv2.go / uniswapv3.go
│   ├── pancakev2.go / pancakev3.go
│   ├── sushiswap.go / sushiswapv3.go
│   ├── solidlyv3.go / thena.go / aerodrome.go
│   ├── apeswap.go / biswap.go / mdex.go / defiswap.go / shibaswap.go
├── strategies/
│   └── base.go             # 默认策略 MovingBrick(价差搬砖), 实现 EventHandler 接口
├── simulation/
│   └── simulation.go       # Anvil 分叉模拟: 起节点/冒充账户/部署合约/查回执/revert原因
├── flashbots/
│   └── flashbots.go        # Flashbots 私有交易 JSON-RPC 客户端
├── database/
│   ├── mongo.go            # Mongo 客户端(全局单例)
│   └── actions.go          # 数据访问层: 集合 tokens/pools/prices/transactions
├── stats/
│   └── stats.go            # 交易统计(按日/区间/模拟或真实)
├── tools/                  # 工具: bigint转换 / 并发Multicall / ABI读取 / 加解密 / JSON保存
├── trader/                 # Foundry 合约工程
│   ├── src/Trader.sol      # 通用 Trader 合约(双腿Swap + UniswapV3风格闪电贷)
│   ├── src/BSCTrader.sol   # BSC 专用版本
│   └── Trader.go           # abigen 生成的 Go 绑定(由 BuildTraderToGo.sh 生成)
├── abis/                   # 各 DEX / ERC20 的 ABI JSON(启动时按名字加载)
├── doc/swap.go             # 旧版 Swap 合约绑定(保留)
├── test/                   # 单元测试与调试用 test
└── 脚本:
    ├── BuildTraderToGo.sh  # forge build + abigen 生成 trader/Trader.go
    ├── build-linux.sh      # 交叉编译 Linux 版本(先执行上面的脚本)
    ├── startdb.sh          # 启动本地 MongoDB
    ├── publish.sh          # rsync 部署到远程服务器
    ├── download_config.sh  # 从服务器拉取配置与黑名单
    ├── download_db.sh      # 从服务器拉取 mongodump 备份并恢复
    └── clean.sh            # 清理编译产物
```

## 三、核心工作流程

### 1. 事件订阅与批处理（monitor）

- 启动时读取 `config.dexs`，把各 DEX 的 Swap Topic 合并成一个 `FilterQuery` 通过 WebSocket 订阅（[eventmonitor.go:44](monitor/eventmonitor.go#L44)）。
- 多个 WS 地址自动故障转移：断线后轮询下一个，5 秒后重连（[eventmonitor.go:391](monitor/eventmonitor.go#L391)）。
- 事件先缓存进 `map[poolAddr]Log`（按池去重，同一池只保留最后一次事件），定时器 `event_waiting_time`（毫秒）到期后整批处理——把同一段时间内多个池的 Swap 合并计算（[eventmonitor.go:176](monitor/eventmonitor.go#L176)）。

### 2. 预处理与黑名单（dex/base.go）

- **新池入库**：事件地址先查 `pools` 表，缺失的池通过 Multicall 批量调用 `factory()/token0()/token1()` 拉取信息（[dex/base.go:259](dex/base.go#L259)）；工厂不在配置列表中的池直接加入池黑名单。
- **新 Token 入库**：批量调用 `name/symbol/totalSupply/decimals`，兼容 name/symbol 为 bytes32 的旧式 ERC20（`token_erc20a.json` 名单，[dex/base.go:455](dex/base.go#L455)）。
- 黑名单（`<net_name>_pool_blacklist.json`、`<net_name>_token_blacklist.json`）启动时加载、运行时自动追加并落盘。

### 3. 价格刷新（multicall 单批读取）

事件涉及的所有 token 先从 `prices` 关联 `pools`/`tokens` 查出全部相关池，再打成一个 Multicall 批（[eventmonitor.go:569](monitor/eventmonitor.go#L569)）：

```
getBlockNumber  +  getBasefee  +  Trader合约各baseToken余额  +  每个池的价格调用
```

- 一次调用即取到当前区块号与全部价格，保证价格一致性；结果按 `chunk_length` 分片、`max_concurrent` 并发执行。
- **若拿到的区块号 > 事件所在区块号，说明价格已过期，直接丢弃这批事件**（否则价格不是事件发生时点的快照）。

### 4. 套利判定（strategies/MovingBrick）

「搬砖」策略：同一交易对（如 ETH/USDT）在不同池之间存在价差时，在**低价池买入、高价池卖出**（[strategies/base.go:92](strategies/base.go#L92)）：

1. 事件池必须包含配置的 base token（如 ETH、USDT、USDC、WBNB）。
2. 取出该交易对所有池的最新价格，对齐为 base/quote 方向，过滤掉 reserve 不足 `base_min_reserve` 的池。
3. 按价格排序：最低价池 = 买入池，最高价池 = 卖出池。
4. 计算目标价 = 两池均价，卖出量 `deltaSell` 取较小储备池的一半再打折（`delta_coefficient` 手动修正），利润 = 卖出所得 − 买入成本（均含手续费）。
5. 收益换算成 USD（gas 价格 × 历史 gas 用量，`GetGas` 取历史 min/max），扣除 gas 后仍大于 `min_profit_usd` 才执行。
6. 借贷池：从配置 `base_tokens` 里选一个不等于买卖池的池，资金不足时由 Trader 合约从这里闪电贷。

### 5. 交易执行

- **模拟模式**（`simulation.enable: true`）：用 Anvil 在**事件所在区块**分叉链（[simulation.go:53](simulation/simulation.go#L53)）→ 冒充大额账户给测试地址转 ETH → 部署 Trader 合约并转入 0.1 base token → 发送交易 → 轮询回执、计算收益、失败取 revert 原因。`Deadline` 自动 +3 块补偿 Anvil 自动挖矿。
- **真实模式**：按配置走 普通发送 / Alchemy 私有交易 / Flashbots 私有交易（`rpcs.flashbots`，[eventmonitor.go:785](monitor/eventmonitor.go#L785)），交易记录先落库，由后台协程确认。

### 6. 交易确认与失败处理

- 后台协程每 20 秒查一次未确认交易，拿到回执后回写 `use_gas/gas_price/income/confirm`（[eventmonitor.go:362](monitor/eventmonitor.go#L362)）。
- 失败交易按 Trader 合约错误码处理（[eventmonitor.go:819](monitor/eventmonitor.go#L819)）：
  - `D`（过期）不处理；`E`（套利失败）需累计 2 次；
  - 其余错误（或 revert）1 次即把交易对中**非 base token** 加入 token 黑名单 → 该池被联动拉黑，不再参与套利。

## 四、Trader 合约（trader/src/Trader.sol）

执行双腿交易的链上合约（Foundry 工程，Solidity 0.8.13）：

```
swap() 传入: buyPool, sellPool, baseToken, borrowPool, amount, deadline,
            borrow(借哪个token), buyPoolType, sellPoolType, buyFee(1e4), sellFee(1e4)
```

- 池类型：`1`=UniswapV2 类、`2`=UniswapV3/Aerodrome、`3`=PancakeV3、`4`=Thena(Algebra)、`5`=SolidlyV3。
- 先在 **sellPool 卖出 baseToken**，再在 **buyPool 买入 baseToken**，最后校验余额不减少（`require(balanceAfter >= balanceBefore, "E")`）。
- **余额不足时自动闪电贷**：调用 borrowPool（UniswapV3 风格 flash）借出差额，交易后连本带息归还。
- `sendfee`：利润的 40%（`rates=40`）打给 `block.coinbase`（给区块构建者的优先级贿赂）。
- 错误码：`D`=交易过期、`E`=套利失败（没赚到钱）、`S`=池无流动性、`IIA`=输入金额不足、`IL`=流动性不足、`EP`=非借贷池回调、`EB`=获取余额失败、`W`=withdraw 零地址。

编译部署流程：`BuildTraderToGo.sh` 用 `forge build` 编译并用 `abigen` 生成 `trader/Trader.go`（Go 端 ABI 绑定）；合约本身用 Foundry 脚本部署（`trader/script/Trader.s.sol`），部署地址填入配置 `trader_contract`。

## 五、数据库（MongoDB）

库名：`<net_name>listener`（如 `bsclistener`、`ethlistener`），共 4 张集合：

| 集合 | 说明 | 唯一索引 |
|---|---|---|
| `tokens` | Token 信息（address/name/symbol/decimals/totalSupply） | address |
| `pools` | 交易池（address/factory/token0/token1，token 为地址字符串） | address |
| `prices` | 每池最新价格（price/reserve0/reserve1/blockNumber/fee/dexName，每次更新 `updateTimes`+1） | pool |
| `transactions` | 每次套利尝试（tx/ok/simulation/cost/income/buy_pool/sell_pool/use_gas/error…） | — |

## 六、配置文件（config.example.yaml → config.yaml）

```yaml
net_name: bsc                        # 网络名: 用于库名 <net_name>listener 和黑名单文件名
dburl: mongodb://localhost:27017     # MongoDB 连接
rpcs:
    flashbots: ""                    # 私有交易通道: ""=普通发送, "alchemy"=Alchemy私有交易, "flashbot"=Flashbots
    http: https://bsc-dataseed.defibit.io   # HTTP RPC(查询/发交易)
    ws: wss://bsc.blockpi.network/v1/ws/<key>  # WS RPC(订阅事件, 可配多个自动故障转移)
simulation:
    enable: true                     # 是否用 Anvil 分叉模拟交易
    funds: 0x98cF...                 # 模拟时冒充转账的大额账户(链上实际有钱的地址)
strategies:
    gas_token: {base: ..., quote: ...}   # gas 计价 token 对, 用于把 gas 成本换算成 USD
    base_tokens:                     # base token → 可借贷池列表(每个 base token 至少配 2 个池)
        0xbb4CdB...: [池1, 池2]
event_waiting_time: 100              # 事件批处理等待时间(毫秒), 合并同一窗口内的事件
gas_price: 1e-09                     # gas 单价兜底值(ETH 单位), 实际优先用节点建议值
gas_times: 2                         # gas 上限倍数: GasLimit * gas_times
gas_limit: 300000                    # 单笔交易 gas 上限
eip1559: false                       # 是否使用 EIP-1559 动态费用交易
trader_contract: ""                  # 已部署的 Trader 合约地址(留空则只计算不发单)
base_min_reserve: 5                  # 池中 base token 最小储备量, 过滤流动性不足的池
chunk_length: 100                    # 批量 Multicall 分片大小
pool_chunk_length:                   # 批量拉池时分片大小(不配置则与 chunk_length 逻辑不同, 建议按需设置)
max_concurrent: 10                   # 最大并发数(拉取/处理/确认共用)
min_profit_usd: 0.01                 # 最小收益阈值(USD), 低于不执行
delta_coefficient: 0                 # 卖出量手动系数(默认0, 策略注释建议0.4, 修正理论计算偏差)
tg: {chat_id: ..., token: ...}       # Telegram 通知
debug: false                         # 调试日志
dexs:                                # 监听/支持的 DEX 列表(按需增删)
    - name: PancakeV2                # 适配器名(对应 dex/*.go 与 abis/*.json)
      event: Swap
      topic: 0xd78ad95f...           # Swap 事件签名 Topic0
      factory: 0xcA143Ce...          # 工厂地址(多个工厂可同 name, 配不同 fee)
      fee: 0.0025                    # 手续费率(用于收益计算)
```

## 七、密钥与环境变量（example.env → .env）

启动 `arb` 时需要输入密码，该密码与 `crypto` 命令配合完成密钥加解密：

| 变量 | 说明 |
|---|---|
| `PRIVATE_WIF` | 交易私钥（用 `listener crypto -E <hex私钥>` 加密后填入） |
| `SIGN_WIF` | Flashbots 签名私钥（同上） |
| `RPC_MAINNET` / `RPC_TESTNET` / `RPC_BASE` / `RPC_BSC` | RPC 地址（参考） |

```bash
# 加密: 输入密码后输出 base32 密文, 填入 .env
./listener crypto -E <要加密的内容>
# 解密
./listener crypto -D <base32密文>
```

## 八、编译与运行

#### 1. 编译 Trader 合约的 Go 绑定文件（需安装 foundry、abigen）
```
./BuildTraderToGo.sh          # 默认编译 Trader 合约; 也可带合约名: ./BuildTraderToGo.sh BSCTrader
```

#### 2. 编译项目
Mac 下直接 `go build`；编译 Linux 版本：
```
./build-linux.sh
```

#### 3. 启动数据库（本机 MongoDB，数据目录 ~/work/mongodb）
```
./startdb.sh
```

#### 4. 部署到服务器（可选，也可以本地运行）
```
./publish.sh [eth|bsc|base]   # rsync 二进制/配置/abis/黑名单 到远程
./download_config.sh [eth|bsc|base]   # 拉取服务器配置与黑名单
./download_db.sh [ethlistener|bsclistener]  # 拉取 mongodump 备份并 mongorestore
```

#### 5. 准备配置
```
cp config.example.yaml config.yaml   # 修改 RPC/密钥/池配置, 或按网络命名为 eth.config.yaml 等
cp example.env .env                  # 填入加密后的私钥
```

#### 6. 运行项目
```
# 未编译: go run main.go arb -c config.yaml
# 已编译:
./listener arb -c config.yaml        # 套利主程序(启动时输入密码)
```

## 九、命令一览

| 命令 | 说明 |
|---|---|
| `listener arb -c config.yaml` | 套利主程序：订阅事件 → 计算 → 执行交易 |
| `listener stats -D 7` | 统计最近 7 天真实交易 |
| `listener stats -S 2024-01-01 -E 2024-01-31` | 统计指定区间 |
| `listener stats -M` | 只统计模拟交易（不带 -M 则只统计真实交易） |
| `listener crypto -E <内容>` / `-D <密文>` | 用密码加密/解密密钥 |

## 十、一些数据库查询语句

```javascript
// 查重复的 token / pool
db.tokens.aggregate([{$group:{_id:"$address",count:{$sum:1}}},{$match:{count:{$gt:1}}}])
db.pools.aggregate([{$group:{_id:"$address",count:{$sum:1}}},{$match:{count:{$gt:1}}}])
db.prices.aggregate([{$group:{_id:"$pool",count:{$sum:1}}},{$match:{count:{$gt:1}}}])

// 失败的交易 gas 消耗范围
db.transactions.aggregate([{$match:{buy_pool:"",sell_pool:""}},{$group:{_id:null,maxValue: { $max: "$use_gas" },minValue: { $min: "$use_gas" }}}])

// 最近交易
db.transactions.find().sort({created_at:-1}).limit(2)

// 两个 token 间的池及 token 详情(关联查询)
db.pools.aggregate([
    {$match:{token0:{$in:["0x2Bf83D080d8Bc4715984e75E5b3D149805d11751","0x55d398326f99059fF775485246999027B3197955"]},token1:{$in:["0x2Bf83D080d8Bc4715984e75E5b3D149805d11751","0x55d398326f99059fF775485246999027B3197955"]}}},
    {$lookup:{from: "tokens", localField: "token0", foreignField: "address", as: "token0_d"}},
    {$unwind: "$token0_d"},
    {$lookup:{from: "tokens", localField: "token1", foreignField: "address", as: "token1_d"}},
    {$unwind: "$token1_d"}
])
```

## 十一、一些池地址（供配置 base_tokens 参考）

```solidity
// base
address public immutable baseToken = 0x4200000000000000000000000000000000000006;
address public immutable borrowPool1 = 0xd0b53D9277642d899DF5C87A3966A349A798F224;
address public immutable borrowPool2 = 0x48413707B70355597404018e7c603B261fcADf3f;

// eth
address public immutable baseToken = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;
address public immutable borrowPool1 = 0x11b815efB8f581194ae79006d24E0d814B7697F6; // ETH/USDT
address public immutable borrowPool2 = 0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640; // USDC/ETH

// USDC/USDT
address public immutable borrowUSD = 0x3416cF6C708Da44DB2624D63ea0AAef7113527C6;

// bsc
address public immutable baseToken = 0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c;
address public immutable borrowPool1 = 0x172fcD41E0913e95784454622d1c3724f546f849;
address public immutable borrowPool2 = 0xf2688Fb5B81049DFB7703aDa5e770543770612C4;
```

## 十二、扩展指南

- **新增 DEX**：在 [abis/](abis/) 放 ABI → 在 [dex/](dex/) 实现 `IDex` 接口（`GetType/GetName/GetTopic/CreatePriceCall/CalcPrice/PriceCallCount`，V2 类可复用 [dex/base.go](dex/base.go) 默认实现）→ 在 [monitor/eventmonitor.go:127](monitor/eventmonitor.go#L127) 的 switch 注册 → 配置 `dexs` 列表 → 若池类型非 V2，在 `Trader.sol` 中实现对应回调并在 `_swap` 分发。
- **新增策略**：实现 `types.EventHandler` 接口（[types/interfaces.go:75](types/interfaces.go#L75)），在 [main.go:157](main.go#L157) 替换 `Handler`。
- **新增借贷池类型**：Trader 合约只支持 UniswapV3 风格 flash 回调，其他借贷协议需在合约内扩展。
