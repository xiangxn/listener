# Robinhood Chain 移植计划

> 分支：`feat/robinhood-port` ｜ 撰写日期：2026-09-01
> 所有合约地址均通过 Robinhood Chain 主网 RPC 链上验证（eth_getCode / eth_call / eth_getLogs）。

## 1. 结论

**可行，且比预期顺利。** 主要利好：

- Robinhood Chain 是 EVM 完全兼容的 Arbitrum Orbit L2（chainId 4663，ETH 为 gas），Solidity/Go 代码无需重写。
- **Uniswap V3 / V2 / PancakeSwap V3 / SushiSwap V2/V3 均已部署**，Swap 事件 topic 与 BSC 完全一致（签名哈希）。
- **PancakeSwap V3 工厂地址与 BSC 完全相同**（`0x0BFbCF9f...`，CREATE2 确定性部署）→ 现有 PancakeV3 配置零改动复用。
- Multicall3 已部署在标准地址；Trader.sol（通用版）的 Uniswap V3 flash 借贷路径可直接用。

主要障碍（均有解决方案）：

| 障碍 | 说明 | 方案 |
|---|---|---|
| WS 订阅 | 公开 RPC 无 JSON-RPC WebSocket（HTTP 400）；sequencer feed 是自定义二进制协议 | Alchemy / QuickNode 提供 wss（官方推荐 Alchemy）；bot 本就支持多 WS failover |
| 区块时间 ~100ms | BSC 是 3s，事件密度高、竞争激烈、无 Flashbots 保护 | 调小 event_waiting_time / 确认轮询间隔；普通发送 + coinbase bribe |
| V2 流动性极薄 | UniswapV2/SushiV2 基本无量（SushiV2 WETH/USDG 2 万块 0 条 Swap） | 策略主战场放在 V3 / PancakeV3 池之间 |
| USDG 6 位小数 | 与常规 18 位不同 | bot 从合约读 decimals，自动适配，仅配置/换算注意 |

## 2. 网络参数（已验证）

| 参数 | 主网 | 测试网 |
|---|---|---|
| Chain ID | 4663 (0x1237) | 46630 |
| HTTP RPC | `https://rpc.mainnet.chain.robinhood.com` | `https://rpc.testnet.chain.robinhood.com` |
| WS (JSON-RPC) | `wss://robinhood-mainnet.g.alchemy.com/v2/{KEY}`（需申请） | `wss://robinhood-testnet.g.alchemy.com/v2/{KEY}` |
| 浏览器 | `https://robinhoodchain.blockscout.com` | `https://explorer.testnet.chain.robinhood.com` |
| Faucet | — | `faucet.testnet.chain.robinhood.com` |
| Gas | baseFee ~0.53 gwei，约 100ms/块 | — |
| 其他提供方 | QuickNode / Blockdaemon / dRPC / Validation Cloud | — |

## 3. 已验证合约地址（主网）

| 组件 | 地址 | 验证方式 |
|---|---|---|
| **Uniswap V3 Factory** | `0x1f7d7550b1b028f7571e69a784071f0205fd2efa` | `getPool(WETH,USDG,100)` 返回真实池 `0x52e65b17...`；注意：**非标准地址**（Uniswap 官方 CREATE2 地址 `0x1F98431c8a...` 在链上无对应工厂，仅有 2KB 占位码） |
| **Uniswap V2 Factory** | `0x8bceaa40b9acdfaedf85adf4ff01f5ad6517937f` | V2 池 `Rabbit/WETH` 调 `factory()` |
| **PancakeSwap V3 Factory** | `0x0BFbCF9fa4f9C56B0F40a671Ad40E0805A091865` | V3 池 `USDG/WETH 0.05%` 调 `factory()`；**与 BSC 同地址** |
| SushiSwap V2 Factory | `0xe52abd50ad151ecdf56427effd715e703696a6b1` | SushiV2 池 `WETH/USDG` 调 `factory()` |
| SushiSwap V3 Factory | `0xe51960f1b45f1c9fb6d166e6a884f866fc70433b` | SushiV3 池调 `factory()` |
| WETH | `0x0Bd7D308f8E1639FAb988df18A8011f41EAcAD73` | eth_getCode 有码 |
| USDG（稳定币，**6 decimals**） | `0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168` | `decimals()` = 6 |
| Multicall3 | `0xcA11bde05977b3631167028862bE2a173976CA11` | eth_getCode 有码（bot 依赖） |
| V4 PoolManager（二期） | `0x8366a39cc670b4001a1121b8f6a443a643e40951` | eth_getCode 有码 |

### Swap 事件 topic（链上验证）

| 协议 | topic | 验证 |
|---|---|---|
| Uniswap V3 / SushiSwap V3 | `0xc42079f9...67` | V3 池近 5000 块 9570 条日志 ✓ |
| PancakeSwap V3 | `0x19b47279...dc83` | PancakeV3 池近 2 万块 116 条日志 ✓（与 BSC 配置相同） |
| Uniswap V2 / SushiSwap V2 | `0xd78ad95f...d822` | 标准签名哈希（V2 池过于冷清，靠签名一致性确认） |

### 参考池（借 flash 款 / 模拟资金）

- **WETH/USDG 0.01% V3 池** `0x52e65b17fb6e5ba00ed806f37afcd2daa50271ca`：**主力池**（近 1000 块 2954 笔 swap，全网最高），fee=100，t0=WETH t1=USDG，WETH 余额 ≈ 2550 —— 当 `simulation.funds`（Anvil impersonate 池地址即可）和 borrow pool。
- **WETH/USDG 0.05% V3 池** `0x69bfaf19c9f377bb306a89aed9f6b07e2c1a8d9a`：同一工厂（0x1f7d7550），fee=500，WETH 余额 ≈ 532 —— 第二个 borrow pool / 套利对手池。
- **WETH/USDG 0.3% V3 池** `0xa9188730fe85be88ad499d7d52b099e800fb0334`：同一工厂，fee=3000，当前区间流动性 ≈ 1.5e17（较薄，原始 WETH 余额 ≈ 0，套利前被大额卖出填充后可用）—— 套利对手池 / 测试砸盘对象。
- ~~`0x16679e2a...`~~：⚠️ 早期 GeckoTerminal 数据误判为 Uni 0.05% 池；实际 `factory()` 返回 `0x1ac9db4a...`（**不是** Uni 工厂），是 45 字节 EIP-1167 代理池（实现 `0x11725976...`），fee 动态（60→53 变过）—— **不能作为 borrow pool，已从配置移除**。

> ⚠️ **LOK 重入约束**：V3 `flash()` 期间池子持锁，**borrow 池不能与 buy/sell 池相同**（回调中再调同一池子 swap 会 revert "LOK"）。配置 borrow pools 时必须保证与套利路径池两两不同。

> ⚠️ **实测价格修正**：当前链上价格 tick ≈ **-198306**（raw 价格 2.44e-9 = 十进制 2444 USDG/WETH）。早期记录"tick=-1"有误（来自其他池/读数），以本次实测为准。

### 池子接口验证（2026-09-01 补充）

- Robinhood 的 V3 池是**标准 UniswapV3Pool 接口**（`slot0()=0x3850c7bd`、`liquidity`、`swap(address,bool,int256,uint160,bytes)`、`flash(address,uint256,uint256,bytes)` 全部存在），但有 **fee-tweaker 扩展**：`setFee` 可用，**fee 是动态存储值**（uni 0.01% 池当前 fee=100，代理池曾为 60/53）→ bot 的 `CreatePriceCall` 每次刷新都调 `fee()` ✓ 无需改代码；**切勿硬编码费率**。
- 反编译确认：池子 dispatcher 26 个函数 = 标准 V3 全函数 + `setFee` + `collectProtocol`；用 `cast disassemble`/selector 校验过。
- **当前链上状态：各 WETH/USDG 池价格完全一致**（实测 tick ≈ -198306，1 WETH ≈ 2444 USDG），价差已被套利者抹平 → 实盘无稳定套利空间，策略主要靠事件驱动抓瞬时价差；fork 测试**不用 `vm.store` 改池子 slot0**（会破坏 tick/流动性区间一致性，导致输出爆炸），改为**真实砸盘**：prank 链上大户（如主力池自身 2550 WETH）转账 WETH → 大额真实卖出砸低 0.3% 池价格 → Trader flash 套利（详见 Phase 3 fork 测试）。

## 4. 移植工作清单（分阶段）

### Phase 0 — 前置准备（需你操作）

- [ ] 注册 Alchemy（免费层即可），创建 Robinhood Chain 应用，拿 HTTP + **WS** key（主网/测试网各一）
- [ ] 准备交易账户（多签/冷钱包自行决定），主网注入少量 ETH（gas）；测试网从 faucet 领
- [ ] 确认现有 BSC bot 运行状态 —— 两链可并行（配置文件、DB 名、黑名单文件均按 `net_name` 隔离）

### Phase 1 — 配置移植（纯配置，不动代码）

新建 `robinhood.config.yaml`（参照 config.example.yaml）：

```yaml
net_name: robinhood          # → DB: robinhoodlistener；黑名单: robinhood_*.json
http: https://rpc.mainnet.chain.robinhood.com   # 或 Alchemy http
ws:
  - wss://robinhood-mainnet.g.alchemy.com/v2/xxx
  - wss://<endpoint>.robinhood-mainnet.quiknode.pro/xxx   # 可选第二路
dexs:                        # 只保留链上真实存在的
  - name: UniswapV3          # topic 0xc42079f9..., factory 0x1f7d7550..., fee: 0
  - name: PancakeV3          # topic 0x19b47279..., factory 0x0BFbCF9f...（BSC 同款）, fee: 0
  - name: UniswapV2          # topic 0xd78ad95f..., factory 0x8bceaa40..., fee: 0.003
  - name: SushiSwap          # topic 0xd78ad95f..., factory 0xe52abd50..., fee: 0.003
  - name: SushiSwapV3        # topic 0xc42079f9..., factory 0xe51960f1..., fee: 0
  # Thena/ApeSwap/Biswap/MDEX 等 BSC-only 的不配置
strategies:
  base_tokens:
    - WETH (0x0Bd7D...)      # decimals 18
    - USDG (0x5fc5360D...)   # decimals 6 —— 注意金额换算
  # borrow pools（支持 flash 的 V3 池）：
  #   WETH → 0x52e65b17..., 0x69bfaf19...
  #   USDG → 0x52e65b17..., 0x69bfaf19...
simulation:
  funds: 0x52e65b17fb6e5ba00ed806f37afcd2daa50271ca
gas_price: 1e-09             # 当前 baseFee ~0.53 gwei，1 gwei 可行
eip1559: false
```

- [x] 创建 `robinhood.config.yaml` + `data/robinhood_pool_blacklist.json` / `data/robinhood_token_blacklist.json` / `data/robinhood_token_erc20a.json`（空数组，黑名单统一放 `data/`）
- [x] `foundry.toml` 加 `robinhood = ${RPC_ROBINHOOD}`；`example.env` 同步 RPC_ROBINHOOD(_TESTNET)
- [x] `event_waiting_time` 调小（100ms 区块下 100ms 起步，实盘观察后再调）

### Phase 2 — 代码微调（小改动）

- [x] `monitor/eventmonitor.go`：硬编码 `etherscan.io` 链接 → config 新增 `explorer` 字段（robinhood → `robinhoodchain.blockscout.com`）
- [x] **删除 Flashbots 整套代码**（无 mempool/relay）：`flashbots/` 包删除、`rpcs.flashbots` 配置移除、`SendPrivateTransaction`/`GetSignKey` 移除，交易直接 `SendTransaction`（sequencer 排序）
- [x] tx 确认轮询间隔 20s → 3s（100ms 出块，原 20s 太慢）
- [ ] 可选二期：eth_getLogs 轮询 fallback（注意：该链 getLogs 有范围限制，50 万块范围即超时，轮询窗口需 ≤1 万块）；V4 adapter（PoolManager 已部署，memecoin 成交量主要在 V4，但工作量大，建议二期）

### Phase 3 — Trader 合约部署（Foundry）

- [x] 使用通用 **Trader.sol**（V3 `flash()` + `IUniswapV3FlashCallback` 路径现成可用；池子接口已验证为标准 V3）
- [x] `foundry.toml` 加 `[rpc_endpoints] robinhood`；`Trader.s.sol` 部署注释补 robinhood 命令
- [x] 新增 fork 测试 `RobinhoodTrader.t.sol`：fork 主网 RPC，**真实砸盘制造价差**（prank 大户转账 WETH → 卖出 100 WETH 砸低 0.3% 池 → Trader flash 借 10 WETH → 主力池卖 → 砸盘池买回 → 还款 → 提款），2/2 测试通过。三池组合：borrow=0x69bfaf19 / sell=0x52e65b17 / buy=0xa9188730
- [x] **修复 Trader.sol flash 回调下溢 bug**：原 `sendfee(after-min, before-min)` 中 `sub(balanceBefore, amountMin)` 在借款池 fee>0 时必然 panic(0x11)（before=借款额 < amount+fee）；改为 `profit = sub(balanceAfter, balanceBefore)`（与还款无关，纯利润）。BSC 老代码同样有隐患，独立分支直接修复
- [x] `BuildTraderToGo.sh` 重新生成 `trader/Trader.go`（修复后字节码，abigen 1.17.5 生成，39KB，`go build ./...` 通过）
- [ ] 先部署测试网（46630），验证后部署主网；部署地址记录到 README/部署文档

### Phase 4 — 测试验证

- [ ] 测试网：部署 Trader → 模拟模式跑通 → 小额真实模式
- [ ] 主网模拟模式：Anvil fork 主网，impersonate `0x52e65b17...`，跑真实事件验证「发现套利 → 模拟成功」
- [ ] 小额实盘：Trader 注入 0.1~0.5 ETH + 少量 USDG，真实模式跑 1~2 天
- [ ] 调参：`event_waiting_time`、`min_profit_usd`、`delta_coefficient`、`gas_price`、gas_limit（300000 可能需上调）

### Phase 5 — 上线

- [ ] 资金到位、Telegram 告警验证、日志/DB 监控
- [ ] 稳定后评估二期：V4 池支持（memecoin 主力）、轮询 fallback、三角套利原型（test/triangles2.py 可顺手接入）

## 5. 风险与注意事项

1. **排序器中心化 + 合规筛查**：Robinhood 运营 sequencer，有制裁地址筛查，机器人行为理论上在筛查范围内，但正常套利无影响。
2. **无 MEV 保护**：没有 Flashbots relay，交易走公开 mempool → 100ms 出块 + coinbase bribe 是目前唯一的竞争壁垒；实盘后需评估被抢跑率。
3. **V2 流动性薄**：不要依赖 V2 池作为主战场；借 flash 款只用 V3 大池。
4. **USDG 6 decimals**：配置里金额换算和利润阈值注意，bot 从合约读 decimals 会自动处理，但 `min_profit_usd` 等仍是 USD 口径，无碍。
5. **公开 RPC 限流**：生产建议 HTTP 也走 Alchemy（或自建节点），仅用公共 RPC 会被限流。
6. **gas 波动**：Arbitrum 的 baseFee 会随 L1 数据价格波动（链启动 90 天免 gas 补贴可能已影响），保持 1 gwei 的 gas_price 并观察。

## 6. 参考链接

- Robinhood 官方开发文档：https://docs.robinhood.com/chain/connecting/
- Chainlist 4663：https://chainlist.org/chain/4663
- 区块浏览器：https://robinhoodchain.blockscout.com
- Uniswap Robinhood Chain 部署页：https://developers.uniswap.org/docs/protocols/v3/deployments/v3-robinhood-chain-deployments
- Uniswap 上线公告：https://blog.uniswap.org/es-ES/robinhood-chain-is-live
- robinhood-toolkit（已验证网络常量/坑点）：https://github.com/nirholas/robinhood-toolkit
- RobinArb（同类套利 bot，V4+曲线）：https://github.com/FlipZ3ro/RobinArb
