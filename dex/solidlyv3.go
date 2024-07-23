package dex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

type SolidlyV3 struct {
	Dex
}

type SolidlySlot0 struct {
	SqrtPriceX96 *big.Int
	Tick         *big.Int
	Fee          *big.Int
	Unlocked     bool
}

func (d *SolidlyV3) GetType() uint8      { return 5 }
func (d *SolidlyV3) PriceCallCount() int { return 3 }

func (u *SolidlyV3) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	poolContract := multicall.Contract{ABI: u.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(SolidlySlot0), "slot0").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "tickSpacing").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "liquidity").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (u *SolidlyV3) CalcPrice(calls []*multicall.Call, blockNumber *big.Int, pool *dt.Pool) (pair dt.Pair) {
	if len(calls) == 0 || calls[0].Failed || calls[1].Failed || calls[2].Failed {
		return
	}

	slot0 := calls[0].Outputs.(*SolidlySlot0)
	u.Fee = tools.PreservePrecision(float64(slot0.Fee.Uint64())*1e-6, 6)
	tickSpacing := int32(calls[1].Outputs.(*dt.ResBigInt).Int64())
	liquidity := calls[2].Outputs.(*dt.ResBigInt).Int
	price := CalcPriceV3(slot0.SqrtPriceX96, pool.Token0.Decimals, pool.Token1.Decimals)

	// Calculate token0 and token1 reserves
	token0Reserve, token1Reserve := CalcReserveV3(slot0.Tick, tickSpacing, liquidity, slot0.SqrtPriceX96)

	// u.SavePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	pair = u.CreatePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	u.monitor.Logger().Debug(pool.Token0.Symbol, "/", pool.Token1.Symbol, " price: ", price, " Pool: ", pool.Address,
		" blockNumber: ", blockNumber, " reserves: ", token0Reserve, token1Reserve, u.Name)
	return
}
