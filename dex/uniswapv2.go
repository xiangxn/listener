package dex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	dt "github.com/xiangxn/listener/types"
)

type SwapV2 struct {
	Sender     common.Address
	Amount0In  *big.Int
	Amount1In  *big.Int
	Amount0Out *big.Int
	Amount1Out *big.Int
	To         common.Address
}

type UniswapV2 struct {
	Dex
}

type reserves struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

// 根据剩余储备量计算整体平均价格
// 统一以y除以x表示价格
func CalcPriceV2(x, y *big.Int, xDecimals, yDecimals uint64) (price *big.Float) {
	xDec := new(big.Int).SetUint64(xDecimals)
	yDec := new(big.Int).SetUint64(yDecimals)
	// 转换为相同的小数位数
	xFloat := new(big.Float).Quo(new(big.Float).SetInt(x), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), xDec, nil)))
	yFloat := new(big.Float).Quo(new(big.Float).SetInt(y), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), yDec, nil)))
	price = new(big.Float).Quo(yFloat, xFloat)
	return
}

// 根据事件数据计算最后一笔交易的平均价格
func (u *UniswapV2) calcSwapPrice(pool dt.Pool, vLog types.Log) (price *big.Float) {
	swapEvent := UnpackSwapEvent[SwapV2](vLog, *u.Abi, DEFAULT_SWAP_NAME)
	xDec := new(big.Int).SetUint64(pool.Token0.Decimals)
	yDec := new(big.Int).SetUint64(pool.Token1.Decimals)
	if swapEvent.Amount0In.Cmp(big.NewInt(0)) == 1 && swapEvent.Amount1Out.Cmp(big.NewInt(0)) == 1 {
		xFloat := new(big.Float).Quo(new(big.Float).SetInt(swapEvent.Amount0In), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), xDec, nil)))
		yFloat := new(big.Float).Quo(new(big.Float).SetInt(swapEvent.Amount1Out), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), yDec, nil)))
		price = new(big.Float).Quo(yFloat, xFloat)
	} else if swapEvent.Amount1In.Cmp(big.NewInt(0)) == 1 && swapEvent.Amount0Out.Cmp(big.NewInt(0)) == 1 {
		xFloat := new(big.Float).Quo(new(big.Float).SetInt(swapEvent.Amount0Out), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), xDec, nil)))
		yFloat := new(big.Float).Quo(new(big.Float).SetInt(swapEvent.Amount1In), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), yDec, nil)))
		price = new(big.Float).Quo(yFloat, xFloat)
	}
	return
}

// 计算带价格与滑点
// 其中x是token1,y是token0
func (u *UniswapV2) calculateSlippage(x, y, dx *big.Int, xDecimals, yDecimals uint64) (priceBefore, priceAfter, slippage *big.Float) {
	xDec := new(big.Int).SetUint64(xDecimals)
	yDec := new(big.Int).SetUint64(yDecimals)
	// 转换为相同的小数位数
	xFloat := new(big.Float).Quo(new(big.Float).SetInt(x), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), xDec, nil)))
	yFloat := new(big.Float).Quo(new(big.Float).SetInt(y), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), yDec, nil)))
	dxFloat := new(big.Float).Quo(new(big.Float).SetInt(dx), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), xDec, nil)))

	// 交易前价格
	priceBefore = new(big.Float).Quo(xFloat, yFloat)

	// 模拟交易后的新储备量
	xPrime := new(big.Float).Add(xFloat, dxFloat)
	yPrime := new(big.Float).Quo(new(big.Float).Mul(xFloat, yFloat), xPrime)

	// 交易后价格
	priceAfter = new(big.Float).Quo(xPrime, yPrime)

	// 滑点计算
	// slippage = new(big.Float).Sub(
	// 	new(big.Float).Quo(priceBefore, priceAfter),
	// 	big.NewFloat(1),
	// )
	slippage = new(big.Float).Quo(new(big.Float).Sub(priceAfter, priceBefore), priceBefore)
	return
}
