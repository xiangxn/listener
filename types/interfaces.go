package types

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/go-multicall"
	"github.com/xiangxn/listener/config"
)

type IActions interface {
	InitDataBase()
	GetSimplePools(addrs []string) (pools []SimplePool)
	GetSimplePool(addr string) SimplePool
	SaveTransaction(tx Transaction)
	SavePair(pool *Pool, price *big.Float, reserve0, reserve1, blockNumber *big.Int, fee float64, dexName string) Pair
	GetTokens(pool *Pool) bool
	GetPools(poolAddrs []string) (existingPool []string)
	GetPoolsByTokens(tokens []string) (pools []Pool)
	SavePools(pools []interface{}) error
	SaveTokens(docs []interface{}) error
	GetExistingTokens(tokens []string) (existingToken []string)
	GetPairsByTokens(tokens []string) (pairs Pairs)
	GetTransactions(ok bool, confirm bool) (txs []Transaction)
	UpdateTransaction(hash string, confirm bool, gasUsed, gasPrice uint64, income float64, ok bool)
	GetToken(addr string) Token
	GetGas(buyPool, sellPool string) (min, max int64)
	GetFailTransacttionCount(buyPool, sellPool string) int

	//查询Transacttion
	SearchTransacttion(simulation bool, start time.Time, end time.Time) (txs []Transaction)
}

type IMonitor interface {
	Run()
	Cancel()

	/**********公开给处理器调用的*********/
	GetPrivateKey() string
	GetContext() context.Context
	GetHttpClient() *ethclient.Client
	GetChainID() *big.Int
	Logger() logrus.FieldLogger
	DB() IActions
	Multicall() *multicall.Caller
	Config() *config.Configuration
	//添加新的token黑名单,并保存到json文件
	AddTokenBlacklist(addr string)
	GetTokenBlacklist() []string
	// 添加地址到pool黑名单(里面也包括不支持池,过滤掉为了提高处理效率)
	AddPoolBlacklist(addr string)
	GetPoolBlacklist() []string
	//添加旧的ERC20 token (name和symbol都是byte32类型),并保存到json文件
	AddERC20A(addr common.Address)
	GetERC20A() []string
	SendToTG(msg string)
	//更新指定池价格,并返回带价格的池信息
	UpdatePrice(pools []Pool)
	// 调用合约发起套利,并保存交易hash后续验证结果
	DoSwap(params SwapParams)
	GetUseGas(buyPool, sellPool *Pair, amount float64) int64

	TestEvent(event types.Log)
}

// EventHandler 事件业务句柄
type EventHandler interface {
	CreateBalanceCalls(tokenABI string, account common.Address) []*multicall.Call
	GetBaseTokens() ([]string, []uint64)
	GetBaseToken(token0, token1 string) string
	GetBaseDecimals(baseToken string) uint64
	// 计算套利,如果能套利则返回真
	// tokenPair中第一个为买入token地址，第二个为卖出token地址
	CalcArbitrage(monitor IMonitor, event types.Log, pool *Pool, gasPrice float64) (arbitrage *Arbitrage, ok bool)
	// Do 处理命中的事件
	Do(monitor IMonitor, arbitrage *Arbitrage)
}
