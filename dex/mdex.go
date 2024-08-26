package dex

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

type MDEX struct {
	Dex
}

const MDEX_GETPAIRFEES_ABI = `[{"constant":true,"inputs":[{"internalType":"address","name":"pair","type":"address"}],"name":"getPairFees","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`

func (d *MDEX) PriceCallCount() int {
	return 2
}

func (u *MDEX) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	fABI, _ := multicall.ParseABI(MDEX_GETPAIRFEES_ABI)
	factoryContract := multicall.Contract{ABI: fABI, Address: common.HexToAddress(pool.Factory)}
	poolContract := multicall.Contract{ABI: u.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(reserves), "getReserves").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = factoryContract.NewCall(new(dt.ResBigInt), "getPairFees", common.HexToAddress(pool.Address)).Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (u *MDEX) CalcPrice(calls []*multicall.Call, blockNumber uint64, pool *dt.Pool) (pair dt.Pair) {
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
