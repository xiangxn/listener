package dex

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xiangxn/go-multicall"

	"github.com/xiangxn/listener/tools"
	dt "github.com/xiangxn/listener/types"
)

type ThenaEvent struct {
	Sender    common.Address
	Recipient common.Address
	Amount0   *big.Int
	Amount1   *big.Int
	Price     *big.Int
	Liquidity *big.Int
	Tick      *big.Int
}

type GlobalState struct {
	Price              *big.Int
	Tick               *big.Int
	Fee                uint16
	TimepointIndex     uint16
	CommunityFeeToken0 uint16
	CommunityFeeToken1 uint16
	Unlocked           bool
}

// algebra
type Thena struct {
	Dex
}

func (d *Thena) GetType() uint8      { return 4 }
func (d *Thena) PriceCallCount() int { return 3 }

func (u *Thena) CreatePriceCall(pool *dt.Pool) (calls []*multicall.Call) {
	poolContract := multicall.Contract{ABI: u.Abi, Address: common.HexToAddress(pool.Address)}
	call := poolContract.NewCall(new(GlobalState), "globalState").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "tickSpacing").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	call = poolContract.NewCall(new(dt.ResBigInt), "liquidity").Name(pool.Address).AllowFailure()
	calls = append(calls, call)
	return
}

func (u *Thena) CalcPrice(calls []*multicall.Call, blockNumber uint64, pool *dt.Pool) (pair dt.Pair) {
	if len(calls) == 0 {
		return
	}
	if calls[0].Failed || calls[1].Failed || calls[2].Failed {
		return
	}
	slot0 := calls[0].Outputs.(*GlobalState)
	u.Fee = tools.PreservePrecision(float64(slot0.Fee)*1e-6, 6)
	tickSpacing := int32(calls[1].Outputs.(*dt.ResBigInt).Int64())
	liquidity := calls[2].Outputs.(*dt.ResBigInt).Int

	price := CalcPriceV3(slot0.Price, pool.Token0.Decimals, pool.Token1.Decimals)
	// Calculate token0 and token1 reserves
	token0Reserve, token1Reserve := CalcReserveV3(slot0.Tick, tickSpacing, liquidity, slot0.Price)

	// u.SavePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	pair = u.CreatePair(pool, price, token0Reserve, token1Reserve, blockNumber, u.Fee)
	u.monitor.Logger().Debug(pool.Token0.Symbol, "/", pool.Token1.Symbol, " price: ", price, " Pool: ", pool.Address,
		" blockNumber: ", blockNumber, " reserves: ", token0Reserve, token1Reserve, u.Name)
	return
}
