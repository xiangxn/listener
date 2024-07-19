package dex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

type DefiSwap struct {
	Dex
}

const DEFISWAP_TOTALFEEBASISPOINT_ABI = `[{"constant":true,"inputs":[],"name":"totalFeeBasisPoint","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`

func (d *DefiSwap) PriceCallCount() int {
	return 2
}

func (u *DefiSwap) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	poolContract := multicall.Contract{ABI: u.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(reserves), "getReserves").Name(pool.Address).AllowFailure()
	calls = append(calls, call)

	fABI, _ := multicall.ParseABI(DEFISWAP_TOTALFEEBASISPOINT_ABI)
	factoryContract := multicall.Contract{ABI: fABI, Address: common.HexToAddress(pool.Factory)}
	call = factoryContract.NewCall(new(dt.ResBigInt), "totalFeeBasisPoint").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (u *DefiSwap) CalcPrice(calls []*multicall.Call, blockNumber *big.Int, pool *dt.Pool) (pair dt.Pair) {
	if len(calls) == 0 || calls[0].Failed || calls[1].Failed {
		return
	}
	logger := u.monitor.Logger()
	u.Fee = tools.PreservePrecision(float64(calls[1].Outputs.(*dt.ResBigInt).Uint64())*1e-4, 6)
	res := calls[0].Outputs.(*reserves)
	price := CalcPriceV2(res.Reserve0, res.Reserve1, pool.Token0.Decimals, pool.Token1.Decimals)
	// u.SavePair(pool, price, res.Reserve0, res.Reserve1, blockNumber, u.Fee)
	pair = u.CreatePair(pool, price, res.Reserve0, res.Reserve1, blockNumber, u.Fee)
	logger.Debug(pool.Token0.Symbol, "/", pool.Token1.Symbol, " price: ", price, " Pool: ", pool.Address, " blockNumber: ", blockNumber, " reserves: ", res.Reserve0, res.Reserve1, u.Name)
	return
}
