package dex

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

type SwapV3 struct {
	Sender       common.Address
	Recipient    common.Address
	Amount0      *big.Int
	Amount1      *big.Int
	SqrtPriceX96 *big.Int
	Liquidity    *big.Int
	Tick         *big.Int
}

type Slot0 struct {
	SqrtPriceX96               *big.Int
	Tick                       *big.Int
	ObservationIndex           uint16
	ObservationCardinality     uint16
	ObservationCardinalityNext uint16
	FeeProtocol                uint8
	Unlocked                   bool
}

type UniswapV3 struct {
	Dex
}

const MIN_TICK = -887272
const MAX_TICK = 887272

// Create a big.Float with the value 2^96
var Q96 = new(big.Int).Lsh(big.NewInt(1), 96)

func (d *UniswapV3) GetType() uint8      { return 2 }
func (d *UniswapV3) PriceCallCount() int { return 4 }

func (u *UniswapV3) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	poolContract := multicall.Contract{ABI: u.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(dt.ResBigInt), "fee").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "tickSpacing").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(Slot0), "slot0").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "liquidity").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (u *UniswapV3) CalcPrice(calls []*multicall.Call, blockNumber uint64, pool *dt.Pool) (pair dt.Pair) {
	if len(calls) == 0 || calls[0].Failed || calls[1].Failed || calls[2].Failed || calls[3].Failed {
		return
	}

	slot0 := calls[2].Outputs.(*Slot0)
	liquidity := calls[3].Outputs.(*dt.ResBigInt).Int
	u.Fee = tools.PreservePrecision(float64(calls[0].Outputs.(*dt.ResBigInt).Uint64())*1e-6, 6)
	tickSpacing := int32(calls[1].Outputs.(*dt.ResBigInt).Int64())
	price := CalcPriceV3(slot0.SqrtPriceX96, pool.Token0.Decimals, pool.Token1.Decimals)

	// Calculate token0 and token1 reserves
	token0Reserve, token1Reserve := CalcReserveV3(slot0.Tick, tickSpacing, liquidity, slot0.SqrtPriceX96)

	// u.SavePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	pair = u.CreatePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	u.monitor.Logger().Debug(pool.Token0.Symbol, "/", pool.Token1.Symbol, " price: ", price, " Pool: ", pool.Address,
		" blockNumber: ", blockNumber, " reserves: ", token0Reserve, token1Reserve, u.Name)
	return
}

func CalcReserveV3(tick *big.Int, tickSpacing int32, liquidity, sqrtPriceX96 *big.Int) (token0Reserve, token1Reserve *big.Int) {
	tickLower, tickUpper := GetLowerUpperTick(int32(tick.Int64()), tickSpacing)
	priceLower := TickToSqrtPriceQ96(tickLower.Int64())
	priceUpper := TickToSqrtPriceQ96(tickUpper.Int64())
	token0Reserve = CalcAmount0(liquidity, sqrtPriceX96, priceUpper)
	token1Reserve = CalcAmount1(liquidity, priceLower, sqrtPriceX96)
	return
}

func CalcPriceV3(sqrtPriceX96 *big.Int, decimals0, decimals1 uint64) (price *big.Float) {
	sqrtPriceX96Float := new(big.Float).SetInt(sqrtPriceX96)
	// Divide sqrtPriceX96 by 2^96
	sqrtPrice := new(big.Float).Quo(sqrtPriceX96Float, new(big.Float).SetInt(Q96))
	// Square the result to get the price
	price = new(big.Float).Mul(sqrtPrice, sqrtPrice)
	decimalsDiff := int64(decimals0) - int64(decimals1)
	adjustmentFactor := new(big.Float)
	if decimalsDiff >= 0 {
		adjustmentFactor.SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalsDiff)), nil))
		price.Mul(price, adjustmentFactor)
	} else {
		adjustmentFactor.SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-decimalsDiff)), nil))
		price.Quo(price, adjustmentFactor)
	}
	return
}

func TickToSqrtPriceQ96(tick int64) *big.Int {
	ratio := new(big.Int)
	absTick := tick
	if tick < 0 {
		absTick = -tick
	}
	if absTick > MAX_TICK {
		panic(fmt.Sprintf("TickToSqrtPriceQ96 tick invalid, tick: %d", absTick))
	}
	if absTick&0x1 != 0 {
		ratio.SetString("fffcb933bd6fad37aa2d162d1a594001", 16)
	} else {
		ratio.SetString("100000000000000000000000000000000", 16)
	}
	if absTick&0x2 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("fff97272373d413259a46990580e213a", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x4 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("fff2e50f5f656932ef12357cf3c7fdcc", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x8 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("ffe5caca7e10e4e61c3624eaa0941cd0", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x10 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("ffcb9843d60f6159c9db58835c926644", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x20 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("ff973b41fa98c081472e6896dfb254c0", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x40 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("ff2ea16466c96a3843ec78b326b52861", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x80 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("fe5dee046a99a2a811c461f1969c3053", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x100 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("fcbe86c7900a88aedcffc83b479aa3a4", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x200 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("f987a7253ac413176f2b074cf7815e54", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x400 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("f3392b0822b70005940c7a398e4b70f3", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x800 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("e7159475a2c29b7443b29c7fa6e889d9", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x1000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("d097f3bdfd2022b8845ad8f792aa5825", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x2000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("a9f746462d870fdf8a65dc1f90e061e5", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x4000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("70d869a156d2a1b890bb3df62baf32f7", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x8000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("31be135f97d08fd981231505542fcfa6", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x10000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("9aa508b5b7a84e1c677de54f3e99bc9", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x20000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("5d6af8dedb81196699c329225ee604", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x40000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("2216e584f5fa1ea926041bedfe98", 16))
		ratio = ratio.Rsh(ratio, 128)
	}
	if absTick&0x80000 != 0 {
		ratio = ratio.Mul(ratio, tools.ParseBigInt("48a170391f7dc42444e8fa2", 16))
		ratio = ratio.Rsh(ratio, 128)
	}

	if tick > 0 {
		ratio = new(big.Int).Div(tools.ParseBigInt("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16), ratio)
	}

	b := new(big.Int).Lsh(big.NewInt(1), 32)
	if new(big.Int).Mod(ratio, b).Cmp(big.NewInt(0)) == 0 {
		return ratio.Rsh(ratio, 32)
	} else {
		ratio.Rsh(ratio, 32)
		return ratio.Add(ratio, big.NewInt(1))
	}
}

func GetLowerUpperTick(tick int32, tickSpacing int32) (*big.Int, *big.Int) {
	var tickLower, tickUpper int32
	tickLower = int32(math.Floor(float64(tick)/float64(tickSpacing))) * tickSpacing
	tickUpper = tickLower + tickSpacing
	if tickLower < MIN_TICK {
		tickLower = MIN_TICK
		tickUpper = tickLower + tickSpacing
	}
	if tickUpper > MAX_TICK {
		tickUpper = MAX_TICK
		tickLower = tickUpper - tickSpacing
	}
	return big.NewInt(int64(tickLower)), big.NewInt(int64(tickUpper))
}

func CalcAmount0Delta(liquidity, pa, pb *big.Int, roundUp bool) *big.Int {
	var a, b *big.Int
	if pa.Cmp(pb) == 1 {
		a = new(big.Int).Set(pb)
		b = new(big.Int).Set(pa)
	} else {
		a = new(big.Int).Set(pa)
		b = new(big.Int).Set(pb)
	}
	numerator1 := new(big.Int).Mul(liquidity, Q96)
	numerator2 := new(big.Int).Sub(b, a)
	result := new(big.Int).Mul(numerator1, numerator2)
	check := result
	result = result.Quo(result, a)
	result = result.Quo(result, b)
	if roundUp {
		check = check.Mod(check, a)
		if check.Cmp(big.NewInt(0)) == 1 {
			result = result.Add(result, big.NewInt(1))
		}
	}
	return result
}

func CalcAmount1Delta(liquidity, pa, pb *big.Int, roundUp bool) *big.Int {
	var a, b *big.Int
	if pa.Cmp(pb) == 1 {
		a = pb
		b = pa
	} else {
		a = pa
		b = pb
	}
	p := new(big.Int).Sub(b, a)
	result := new(big.Int).Mul(liquidity, p)
	result = result.Quo(result, Q96)
	if roundUp {
		check := new(big.Int).Mul(liquidity, p)
		check = check.Mod(check, Q96)
		if check.Cmp(big.NewInt(0)) == 1 {
			result = result.Add(result, big.NewInt(1))
		}
	}
	return result
}

func CalcAmount0(liquidity, pa, pb *big.Int) *big.Int {
	if liquidity.Cmp(big.NewInt(0)) == -1 {
		result := CalcAmount0Delta(liquidity.Neg(liquidity), pa, pb, false)
		return result.Neg(result)
	}
	return CalcAmount0Delta(liquidity, pa, pb, true)
}

func CalcAmount1(liquidity, pa, pb *big.Int) *big.Int {
	if liquidity.Cmp(big.NewInt(0)) == -1 {
		result := CalcAmount1Delta(liquidity.Neg(liquidity), pa, pb, false)
		return result.Neg(result)
	}
	return CalcAmount1Delta(liquidity, pa, pb, true)
}
