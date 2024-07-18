package main

import (
	"math/big"
	"testing"

	"github.com/xiangxn/listener/dex"
	"github.com/xiangxn/listener/tools"
)

func TestTickToSqrtPriceQ96(t *testing.T) {
	tick := new(big.Int).SetInt64(-122818)
	expected := tools.ParseBigInt("170629429566153624289111676", 10)
	sqrtPriceQ96 := dex.TickToSqrtPriceQ96(tick.Int64())
	if sqrtPriceQ96.Cmp(expected) != 0 {
		t.Errorf("TickToSqrtPriceQ96(%s)=%s, want=%s", tick, sqrtPriceQ96, expected)
	}
}

func TestCalcReserves(t *testing.T) {
	liquidity := tools.ParseBigInt("66835806823279760363", 10)
	sqrtPriceX96 := tools.ParseBigInt("182711349204900817339797210", 10)
	tick := int32(-121450)
	tickSpacing := int32(200)
	tickLower, tickUpper := dex.GetLowerUpperTick(tick, tickSpacing)

	priceLower := dex.TickToSqrtPriceQ96(tickLower.Int64())
	priceUpper := dex.TickToSqrtPriceQ96(tickUpper.Int64())
	token0Reserve := dex.CalcAmount0(liquidity, sqrtPriceX96, priceUpper)
	token1Reserve := dex.CalcAmount1(liquidity, priceLower, sqrtPriceX96)

	want0Reserve := "71872219985808057658"
	want1Reserve := "1154196053808741"

	if token0Reserve.Cmp(tools.ParseBigInt(want0Reserve, 10)) != 0 {
		t.Errorf("token0Reserve=%s, want=%s", token0Reserve, want0Reserve)
	}
	if token1Reserve.Cmp(tools.ParseBigInt(want1Reserve, 10)) != 0 {
		t.Errorf("token1Reserve=%s, want=%s", token1Reserve, want1Reserve)
	}
}
