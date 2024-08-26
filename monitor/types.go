package monitor

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/go-multicall"
	"github.com/xiangxn/listener/config"
	dt "github.com/xiangxn/listener/types"
)

type IDex interface {
	GetName() string
	GetTopic() common.Hash
	GetAbi() *abi.ABI
	// 存储价格
	SavePair(pool *dt.Pool, price *big.Float, reserve0, reserve1 *big.Int, blockNumber uint64, fee float64) dt.Pair
	CreatePair(pool *dt.Pool, price *big.Float, reserve0, reserve1 *big.Int, blockNumber uint64, fee float64) dt.Pair

	//创建查询链上价格数据的Call
	CreatePriceCall(pool *dt.Pool) []*multicall.Call
	// 根据链上数据计算价格,如何已经完成计算不需要获取链上数据时返回true
	// 统一以token1除以token0表示价格
	CalcPrice(calls []*multicall.Call, blockNumber uint64, pool *dt.Pool) dt.Pair

	PriceCallCount() int
	// 获取传给合约的交易池类型,1是UniswapV3,2是UniswapV2
	GetType() uint8
}

type monitor struct {
	ctx                context.Context
	cancel             context.CancelFunc
	cfg                config.Configuration
	cli                *ethclient.Client
	httpClient         *ethclient.Client
	handler            dt.EventHandler
	logger             logrus.FieldLogger
	currentBlockNumber uint64
	cacheEvents        map[common.Address]types.Log
	dexs               map[string]IDex
	database           dt.IActions
	multicall          *multicall.Caller
	factorys           []string
	poolBlacklist      []string
	tokenBlacklist     []string
	tokenErc20a        []string
	privateKey         string
	signKey            string
	chainId            *big.Int
	baseFee            *big.Int
	gasPrice           float64
	baseBalance        map[string]float64
	cipher             [32]byte
	sync.RWMutex
}
