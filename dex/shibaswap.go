package dex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

type ShibaSwap struct {
	Dex
}

func (d *ShibaSwap) PriceCallCount() int {
	return 2
}

func (u *ShibaSwap) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	poolContract := multicall.Contract{ABI: u.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(reserves), "getReserves").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "totalFee").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (u *ShibaSwap) CalcPrice(calls []*multicall.Call, blockNumber *big.Int, pool *dt.Pool) {
	if len(calls) == 0 || calls[0].Failed || calls[1].Failed {
		return
	}
	u.Fee = tools.PreservePrecision(float64(calls[1].Outputs.(*dt.ResBigInt).Uint64())*1e-3, 6)
	logger := u.monitor.Logger()
	res := calls[0].Outputs.(*reserves)
	price := CalcPriceV2(res.Reserve0, res.Reserve1, pool.Token0.Decimals, pool.Token1.Decimals)
	u.SavePair(pool, price, res.Reserve0, res.Reserve1, blockNumber, u.Fee)
	logger.Debug(pool.Token0.Symbol, "/", pool.Token1.Symbol, " price: ", price, " Pool: ", pool.Address, " blockNumber: ", blockNumber, " reserves: ", res.Reserve0, res.Reserve1, u.Name)
}
