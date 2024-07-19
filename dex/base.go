package dex

import (
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/tools"

	"github.com/elliotchance/pie/v2"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/go-multicall"
	"go.mongodb.org/mongo-driver/bson/primitive"

	dt "github.com/xiangxn/listener/types"
)

const BlockNumberABI = `[{"inputs":[],"name":"getBlockNumber","outputs":[{"internalType":"uint256","name":"blockNumber","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getBasefee","outputs":[{"internalType":"uint256","name":"basefee","type":"uint256"}],"stateMutability":"view","type":"function"}]`

const FactoryABI = `[
	{
        "inputs": [],
        "name": "factory",
        "outputs": [
            {
                "internalType": "address",
                "name": "",
                "type": "address"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
	{
        "inputs": [],
        "name": "token0",
        "outputs": [
            {
                "internalType": "address",
                "name": "",
                "type": "address"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "token1",
        "outputs": [
            {
                "internalType": "address",
                "name": "",
                "type": "address"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
	{
        "inputs": [],
        "name": "fee",
        "outputs": [
            {
                "internalType": "uint24",
                "name": "",
                "type": "uint24"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    }
]`

var TokenABI string
var TokenAABI string

type Dex struct {
	Name    string
	Topic   common.Hash
	Abi     *abi.ABI
	monitor dt.IMonitor
	Fee     float64
}

func (d *Dex) GetName() string       { return d.Name }
func (d *Dex) GetTopic() common.Hash { return d.Topic }
func (d *Dex) GetAbi() *abi.ABI      { return d.Abi }
func (d *Dex) GetType() uint8        { return 2 }
func (d *Dex) SavePair(pool *dt.Pool, price *big.Float, reserve0, reserve1, blockNumber *big.Int, fee float64) dt.Pair {
	db := d.monitor.DB()
	return db.SavePair(pool, price, reserve0, reserve1, blockNumber, fee, d.GetName())
}

func (d *Dex) PriceCallCount() int { return 1 }

func (d *Dex) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	poolContract := multicall.Contract{ABI: d.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(reserves), "getReserves").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (d *Dex) CalcPrice(calls []*multicall.Call, blockNumber *big.Int, pool *dt.Pool) {
	if len(calls) == 0 || calls[0].Failed {
		return
	}
	call := calls[0]
	logger := d.monitor.Logger()
	if call.Failed {
		logger.WithFields(logrus.Fields{"Dex": d.Name, "Pool": pool.Address, "Method": call.Method}).Error("CalcPrice: Failed to call the contract")
		return
	}
	res := call.Outputs.(*reserves)
	price := CalcPriceV2(res.Reserve0, res.Reserve1, pool.Token0.Decimals, pool.Token1.Decimals)
	d.SavePair(pool, price, res.Reserve0, res.Reserve1, blockNumber, d.Fee)
	logger.Debug(pool.Token0.Symbol, "/", pool.Token1.Symbol, " price: ", price, " Pool: ", pool.Address, " blockNumber: ", blockNumber, " reserves: ", res.Reserve0, res.Reserve1, d.Name)
}

// 处理非标准token发生的异常
func FixTokenError(m dt.IMonitor, calls []*multicall.Call, msg string) {
	var index int = -1
	re := regexp.MustCompile(`failed to unpack call outputs at index \[(\d+)\]:`)
	parts := re.FindStringSubmatch(msg)
	if len(parts) >= 2 {
		d, _ := strconv.Atoi(parts[1])
		index = d
	}
	if index > -1 && index < len(calls) { //记录到旧erc20名单,会在下一次自动处理
		m.AddERC20A(calls[index].Contract.Address)
	}
}

// 交易所类型约束,添加新交易所时需要在这里添加类
type CDex interface {
	UniswapV2 | UniswapV3 | SushiSwap | PancakeV3 | PancakeV2 | SolidlyV3 | DefiSwap | ShibaSwap | Thena | ApeSwap | Biswap | MDEX | Aerodrome
}

func GetDex[T CDex](dexConfig config.DexConfig, monitor dt.IMonitor) *T {
	if TokenABI == "" {
		TokenABI = tools.ReadABIString("ERC20")
	}
	abiPtr := tools.ReadABI(dexConfig.Name)
	if abiPtr == nil {
		panic(fmt.Sprintf("读取abi文件[%s]失败!", dexConfig.Name))
	}
	return &T{
		Dex: Dex{
			Name:    dexConfig.Name,
			Topic:   common.HexToHash(dexConfig.Topic),
			Abi:     abiPtr,
			monitor: monitor,
			Fee:     dexConfig.Fee,
		},
	}
}

const DEFAULT_SWAP_NAME = "Swap"

func UnpackSwapEvent[T any](vLog types.Log, swapAbi abi.ABI, eventName string) T {
	event := new(T)
	err := swapAbi.UnpackIntoInterface(event, eventName, vLog.Data)
	if err != nil {
		log.Fatal(err)
	}

	var indexed abi.Arguments
	for _, arg := range swapAbi.Events[eventName].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	err = abi.ParseTopics(event, indexed, vLog.Topics[1:])
	if err != nil {
		log.Fatal(err)
	}
	return *event
}

// 获取交易对token信息
func GetTokens(m dt.IMonitor, pool *dt.Pool) bool {
	return m.DB().GetTokens(pool)
}

// 把池数据(包括token数据)存储到数据库中,同时过滤掉不支持的交易所的池子的事件
func PreprocessEvent(m dt.IMonitor, factorys []string, logs []types.Log) (result []types.Log) {
	var poolAddrs []string
	if len(logs) < 1 {
		return
	}
	for _, v := range logs {
		poolAddrs = append(poolAddrs, v.Address.Hex())
	}

	existingPool := m.DB().GetPools(poolAddrs)

	// 处理数据库中还不存在的池
	missingPool := pie.FilterNot(poolAddrs, func(value string) bool {
		return pie.Contains(existingPool, value)
	})
	m.Logger().WithFields(logrus.Fields{"PoolCount": len(poolAddrs), "MissingCount": len(missingPool)}).Debug("池处理情况")
	if len(missingPool) == 0 { // 如果没有新池, 就直接返回
		result = logs
		return
	}

	failPool := BatchPool(m, missingPool, factorys)
	for _, v := range logs {
		if !pie.Contains(failPool, v.Address.Hex()) {
			result = append(result, v)
		}
	}
	return
}

// 批量从链上获取池信息(包括token信息)
func BatchPool(m dt.IMonitor, pools []string, factorys []string) (failPool []string) {
	chunk := pie.Chunk(pools, m.Config().ChunkLength)
	var wg sync.WaitGroup
	taskChan := make(chan []string)
	concurrent := make(chan struct{}, m.Config().MaxConcurrent)
	for _, arr := range chunk {
		concurrent <- struct{}{}
		wg.Add(1)
		go func(poolAddrs []string) {
			defer wg.Done()
			result := FetchPool(m, poolAddrs, factorys)
			taskChan <- result
			<-concurrent
		}(arr)
	}
	go func() {
		wg.Wait()
		close(taskChan)
	}()
	for result := range taskChan {
		failPool = append(failPool, result...)
	}
	return
}

// 批量获取池的链上信息，并存储进数据库
func FetchPool(m dt.IMonitor, pools []string, factorys []string) (failPool []string) {
	if len(pools) < 1 {
		return
	}
	mc := m.Multicall()
	var calls []*multicall.Call
	for _, pool := range pools {
		contract, err := multicall.NewContract(FactoryABI, pool)
		if err != nil {
			m.Logger().Error("FetchFactory error: ", err)
			failPool = append(failPool, pool)
			continue
		}
		calls = append(calls, contract.NewCall(new(dt.ResAddress), "factory").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResAddress), "token0").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResAddress), "token1").AllowFailure())
	}
	t := time.Now()
	results, err := mc.Call(nil, calls...)
	if err != nil { //调用multicall失败全部返回
		m.Logger().Error("FetchFactory error: ", err)
		failPool = append(failPool, pools...)
		failPool = pie.Unique(failPool)
		return
	}
	m.Logger().WithFields(logrus.Fields{"T": time.Since(t), "CallCount": len(calls)}).Debug("获取新池")
	var tokens []string
	var docs []interface{}
	resChunk := pie.Chunk(results, 3)
	for _, res := range resChunk {
		address := res[0].Contract.Address.Hex()
		if res[0].Failed || res[1].Failed || res[2].Failed { // 是否有调用链上合约失败
			failPool = append(failPool, address)
			m.Logger().WithField("pool", address).Info("获取池信息失败")
			m.AddPoolBlacklist(address)
			continue
		}
		doc := dt.SimplePool{
			Address: address,
			Token0:  res[1].Outputs.(*dt.ResAddress).Address.Hex(),
			Token1:  res[2].Outputs.(*dt.ResAddress).Address.Hex(),
		}
		if pie.Contains(m.GetTokenBlacklist(), doc.Token0) { // token0在黑名单中
			failPool = append(failPool, doc.Address)
			m.AddPoolBlacklist(doc.Address)
			continue
		} else if pie.Contains(m.GetTokenBlacklist(), doc.Token1) { // token1在黑名单中
			failPool = append(failPool, doc.Address)
			m.AddPoolBlacklist(doc.Address)
			continue
		}
		doc.Factory = res[0].Outputs.(*dt.ResAddress).Hex()
		if !pie.Contains(factorys, doc.Factory) { // 如果工厂地址不在给定的数组中
			failPool = append(failPool, doc.Address)
			m.Logger().WithFields(logrus.Fields{"pool": doc.Address, "factory": doc.Factory}).Info("还未支持的交易市场")
			continue
		}
		tokens = append(tokens, doc.Token0, doc.Token1)
		docs = append(docs, doc)
	}
	tokens = pie.Unique(tokens) // 对tokens去重,用于后面获取token信息
	failTokens := BatchToken(m, tokens)
	// 处理获取token信息失败
	var failPoolIndex []int
	for i, p := range docs {
		for _, token := range failTokens {
			if p.(dt.SimplePool).Token0 == token || p.(dt.SimplePool).Token1 == token {
				failPoolIndex = append(failPoolIndex, i)
				failPool = append(failPool, p.(dt.SimplePool).Address)
			}
		}
	}
	failPoolIndex = pie.Unique(failPoolIndex)
	// 删除获取token失败的pool信息
	if len(failPoolIndex) > 0 {
		docs = pie.Delete(docs, failPoolIndex...)
	}

	if len(docs) > 0 {
		// 把池信息存储到数据库
		err = m.DB().SavePools(docs)
		if err != nil {
			m.Logger().Error("FetchFactory error: ", err)
			failPool = append(failPool, pools...)
		}
	}
	failPool = pie.Unique(failPool)
	return
}

// 批量从链上获取token信息,返回获取失败的token地址
func BatchToken(m dt.IMonitor, tokens []string) (result []string) {
	ts := CheckTokens(m, tokens)
	m.Logger().WithFields(logrus.Fields{"TokenCount": len(tokens), "MissingCount": len(ts)}).Debug("Token处理情况")
	chunk := pie.Chunk(ts, m.Config().ChunkLength)
	var wg sync.WaitGroup
	taskChan := make(chan []string)
	concurrent := make(chan struct{}, m.Config().MaxConcurrent)
	for _, arr := range chunk {
		concurrent <- struct{}{}
		wg.Add(1)
		go func(tokenAddrs []string) {
			defer wg.Done()
			res := FetchToken(m, tokenAddrs)
			taskChan <- res
			<-concurrent
		}(arr)
	}
	go func() {
		wg.Wait()
		close(taskChan)
	}()
	for res := range taskChan {
		result = append(result, res...)
	}
	result = pie.Unique(result)
	return
}

// 解码ERC20属性
func DecodeTokenString(call *multicall.Call) string {
	switch call.Outputs.(type) {
	case *dt.ResHash:
		return strings.ReplaceAll(string(call.Outputs.(*dt.ResHash).Bytes()), "\u0000", "")
	case *dt.ResString:
		return call.Outputs.(*dt.ResString).Result
	default:
		return ""
	}
}
func DecodeTokenBigint(m dt.IMonitor, call *multicall.Call) primitive.Decimal128 {
	ts, ok := primitive.ParseDecimal128FromBigInt(call.Outputs.(*dt.ResBigInt).Int, 0)
	if !ok {
		m.Logger().Error("FetchToken: Converting Decimal128 failed, token=", call.Contract.Address.Hex())
		return primitive.NewDecimal128(0, 0)
	}
	return ts
}

// 创建ERC20调用
func CreateTokenCall(m dt.IMonitor, token string) (calls []*multicall.Call) {
	if TokenABI == "" {
		TokenABI = tools.ReadABIString("ERC20")
	}
	if TokenAABI == "" {
		TokenAABI = tools.ReadABIString("ERC20A")
	}
	if pie.Contains(m.GetERC20A(), token) {
		contract, err := multicall.NewContract(TokenAABI, token)
		if err != nil {
			panic(err)
		}
		calls = append(calls, contract.NewCall(new(dt.ResHash), "name").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResHash), "symbol").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResBigInt), "totalSupply").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResBigInt), "decimals").AllowFailure())
	} else {
		contract, err := multicall.NewContract(TokenABI, token)
		if err != nil {
			panic(err)
		}
		calls = append(calls, contract.NewCall(new(dt.ResString), "name").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResString), "symbol").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResBigInt), "totalSupply").AllowFailure())
		calls = append(calls, contract.NewCall(new(dt.ResBigInt), "decimals").AllowFailure())
	}
	return
}

// 批量获取token的链上信息，并存储进数据库
func FetchToken(m dt.IMonitor, tokens []string) (failTokens []string) {
	mc := m.Multicall()
	var calls []*multicall.Call
	for _, token := range tokens {
		cs := CreateTokenCall(m, token)
		calls = append(calls, cs...)
	}
	t := time.Now()
	results, err := mc.Call(nil, calls...)
	if err != nil { //调用multicall失败全部返回
		m.Logger().Warn("FetchToken error: ", err)
		FixTokenError(m, calls, err.Error())
		failTokens = append(failTokens, tokens...)
		failTokens = pie.Unique(failTokens)
		return
	}
	m.Logger().WithFields(logrus.Fields{"T": time.Since(t), "CallCount": len(calls)}).Debug("获取新tokens")
	var docs []interface{}
	resChunk := pie.Chunk(results, 4)
	for _, res := range resChunk {
		doc := dt.Token{
			Address: res[0].Contract.Address.Hex(),
		}
		if res[0].Failed || res[1].Failed || res[2].Failed || res[3].Failed { // 是否有调用链上合约失败
			failTokens = append(failTokens, doc.Address)
			m.Logger().Info("获取token信息失败: token=", doc.Address)
			continue
		}
		dec := res[3].Outputs.(*dt.ResBigInt).Uint64()
		doc.Name = DecodeTokenString(res[0])
		doc.Symbol = DecodeTokenString(res[1])
		doc.TotalSupply = tools.BigIntToFloat64(res[2].Outputs.(*dt.ResBigInt).Int, dec)
		doc.Decimals = dec
		docs = append(docs, doc)
	}
	// 把池信息存储到数据库
	err = m.DB().SaveTokens(docs)
	if err != nil {
		m.Logger().Error("FetchToken error: ", err)
		failTokens = append(failTokens, tokens...)
	}
	failTokens = pie.Unique(failTokens)
	return
}

// 检查token地址列表,返回数据库中不存在的
func CheckTokens(m dt.IMonitor, tokens []string) (result []string) {
	if len(tokens) < 1 {
		return
	}
	existingToken := m.DB().GetExistingTokens(tokens)
	result = pie.FilterNot(tokens, func(value string) bool {
		return pie.Contains(existingToken, value)
	})

	return
}

// 获取池的工厂地址
func GetFactory(m dt.IMonitor, pool string) *dt.Pool {
	var result dt.Pool
	// 先检查数据库是否存在
	tp := m.DB().GetSimplePool(pool)
	if tp.Address == pool {
		result.Factory = tp.Factory
		result.Address = tp.Address
		result.Token0 = dt.Token{Address: tp.Token0}
		result.Token1 = dt.Token{Address: tp.Token1}
		if GetTokens(m, &result) {
			return &result
		}
	}
	return nil
}
