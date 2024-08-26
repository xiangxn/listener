package dex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

type AerodromeSlot0 struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	Unlocked                   bool
}

type Aerodrome struct {
	Dex
}

func (d *Aerodrome) GetType() uint8      { return 2 }
func (d *Aerodrome) PriceCallCount() int { return 4 }

func (u *Aerodrome) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	poolContract := multicall.Contract{ABI: u.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(dt.ResBigInt), "fee").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "tickSpacing").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(AerodromeSlot0), "slot0").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "liquidity").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (u *Aerodrome) CalcPrice(calls []*multicall.Call, blockNumber uint64, pool *dt.Pool) (pair dt.Pair) {
	if len(calls) == 0 || calls[0].Failed || calls[1].Failed || calls[2].Failed || calls[3].Failed {
		return
	}

	u.Fee = tools.PreservePrecision(float64(calls[0].Outputs.(*dt.ResBigInt).Uint64())*1e-6, 6)
	tickSpacing := int32(calls[1].Outputs.(*dt.ResBigInt).Int64())
	slot0 := calls[2].Outputs.(*AerodromeSlot0)
	liquidity := calls[3].Outputs.(*dt.ResBigInt).Int

	price := CalcPriceV3(slot0.SqrtPriceX96, pool.Token0.Decimals, pool.Token1.Decimals)
	// Calculate token0 and token1 reserves
	token0Reserve, token1Reserve := CalcReserveV3(slot0.Tick, tickSpacing, liquidity, slot0.SqrtPriceX96)

	// u.SavePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	pair = u.CreatePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	u.monitor.Logger().Debug(pool.Token0.Symbol, "/", pool.Token1.Symbol, " price: ", price, " Pool: ", pool.Address,
		" blockNumber: ", blockNumber, " reserves: ", token0Reserve, token1Reserve, u.Name)
	return
}
