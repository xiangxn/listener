package strategies

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/elliotchance/pie/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/go-multicall"
	dt "github.com/xiangxn/listener/types"
)

type MovingBrick struct {
	baseTokens  map[string]dt.Token
	borrowPools map[string][]dt.SimplePool
}

var _ dt.EventHandler = &MovingBrick{}

func (m *MovingBrick) GetBaseDecimals(baseToken string) uint64 {
	return m.baseTokens[baseToken].Decimals
}

func (m *MovingBrick) CreateBalanceCalls(tokenABI string, account common.Address) (calls []*multicall.Call) {
	for _, bt := range pie.Keys(m.baseTokens) {
		b, err := multicall.NewContract(tokenABI, bt)
		if err != nil {
			log.Fatalf("CreateBalanceCalls:%s", err)
			return
		}
		calls = append(calls, b.NewCall(new(dt.ResBigInt), "balanceOf", account))
	}
	return
}

func (m *MovingBrick) InitBaseTokens(monitor dt.IMonitor) {
	if len(m.baseTokens) == 0 {
		m.baseTokens = make(map[string]dt.Token)
	}
	if len(m.borrowPools) == 0 {
		m.borrowPools = make(map[string][]dt.SimplePool)
	}
	conf := monitor.Config().Strategies
	baseTokens := pie.Keys(conf.BaseTokens)
	ts := monitor.DB().GetTokens(baseTokens)
	for _, t := range ts {
		m.baseTokens[t.Address] = t
	}
	for _, bt := range baseTokens {
		bps := conf.BaseTokens[bt]
		m.borrowPools[bt] = monitor.DB().GetSimplePools(bps)
	}
}

func (m *MovingBrick) GetBaseTokens() []dt.Token {
	return pie.Values(m.baseTokens)
}

func (m *MovingBrick) getBorrowPool(buyPool, sellPool *dt.Pair, baseToken string) (borrow string, position uint8) {
	borrowPools := m.borrowPools[baseToken]
	for _, bp := range borrowPools {
		if bp.Address != buyPool.Pool && bp.Address != sellPool.Pool {
			borrow = bp.Address
			if baseToken == bp.Token0 {
				position = 0
			} else {
				position = 1
			}
			return
		}
	}
	return borrowPools[0].Address, 0
}

func (m *MovingBrick) isUSD(token string) bool {
	return strings.Contains(m.baseTokens[token].Symbol, "USD")
}

func (m *MovingBrick) GetBaseToken(token0, token1 string) string {
	keys := pie.Keys(m.baseTokens)
	if pie.Contains(keys, token0) {
		return token0
	} else if pie.Contains(keys, token1) {
		return token1
	}
	return ""
}

func (m *MovingBrick) CalcArbitrage(monitor dt.IMonitor, event types.Log, pool *dt.Pool, gasPrice float64) (arbitrage *dt.Arbitrage, ok bool) {
	baseToken := m.GetBaseToken(pool.Token0.Address, pool.Token1.Address)
	if baseToken == "" {
		monitor.Logger().Debug(fmt.Sprintf(`There is no "basetoken" in the trading pair: %s %s/%s`, pool.Address, pool.Token0.Symbol, pool.Token1.Symbol))
		return nil, false
	}

	data := monitor.DB().GetPairsByTokens([]string{pool.Token0.Address, pool.Token1.Address})
	if len(data) == 0 {
		return nil, false
	}
	// 对齐交易对中的币种
	data = pie.Each(data, func(p *dt.Pair) {
		if p.Token0 != baseToken {
			tmpT := p.Token0
			p.Token0 = p.Token1
			p.Token1 = tmpT
			tmpR := p.Reserve0
			p.Reserve0 = p.Reserve1
			p.Reserve1 = tmpR
			p.Price = 1 / p.Price
		}
	})
	// 过滤掉流动性不足的池
	data = pie.Filter(data, func(d *dt.Pair) bool {
		return d.Reserve0 >= monitor.Config().BaseMinReserve
	})
	if len(data) < 2 {
		return nil, false
	}
	// tmps := pie.Map(data, func(d *dt.Pair) float64 { return d.Price })
	// fmt.Println("data0: ", tmps)
	sort.Sort(&data)
	// tmps = pie.Map(data, func(d *dt.Pair) float64 { return d.Price })
	// fmt.Println("data0: ", tmps)

	var buyPool, sellPool *dt.Pair
	buyPool = data[0]
	sellPool = data[len(data)-1]
	amount, profit, avgPrice := m.calcArbitrage(monitor, sellPool, buyPool)
	if profit > 0 {
		var profitUSD float64
		conf := monitor.Config().Strategies.GasToken
		gasUSDPrice := monitor.DB().GetBasePrice(conf.Base, conf.Quote)
		gas := monitor.GetUseGas(buyPool, sellPool, amount)
		gasUSD := float64(gas) * gasPrice / gasUSDPrice
		borrow, position := m.getBorrowPool(buyPool, sellPool, baseToken)

		if baseToken == conf.Base {
			profitUSD = profit / avgPrice // avgPrice表示: 1 quote = N base
		} else if m.isUSD(baseToken) {
			profitUSD = profit
		} else {
			// 报价basetoken相对于USD的价格
			basePrice := monitor.DB().GetBasePrice(baseToken, conf.Quote) // basePrice表示: 1 base = N quote
			profitUSD = profit * basePrice
		}
		profitUSD -= gasUSD
		if profitUSD < monitor.Config().MinProfitUSD {
			return nil, false
		}

		monitor.Logger().WithFields(logrus.Fields{
			"Profit":      profit,
			"Profit(USD)": profitUSD,
			"Amount":      amount,
			"Symbol":      buyPool.Symbol,
			"PriceBuy":    buyPool.Price,
			"PriceSell":   sellPool.Price,
		}).Info("发现可套利交易")

		arbitrage = new(dt.Arbitrage)
		arbitrage.BlockNumber = event.BlockNumber
		arbitrage.Amount = amount
		arbitrage.BuyPool = *buyPool
		arbitrage.SellPool = *sellPool
		arbitrage.ProfitUSD = profitUSD
		arbitrage.GasPrice = gasPrice
		arbitrage.Borrow = borrow
		arbitrage.Position = position
		arbitrage.BaseToken = baseToken

		return arbitrage, true
	} else {
		return nil, false
	}
}

// 搬平两个池的价格
// 在a池卖出(basetoken), 在b池买入
// 计算中包括了手续费, 如果profit大于0则可以套利
func (m *MovingBrick) calcArbitrage(monitor dt.IMonitor, aPool, bPool *dt.Pair) (deltaSell, profit, targetPrice float64) {
	// 计算目标价格
	targetPrice = (aPool.Price + bPool.Price) / 2
	// 计算卖出数量(以池子小的一个池来计算)
	deltaSell = (aPool.Price - targetPrice) * min(aPool.Reserve0, bPool.Reserve0) / targetPrice / 2

	if deltaSell < 0 { //流动性太低(如何用于交易的base token超过流动性的一半)
		profit = -1
		return
	}
	deltaSell = deltaSell + deltaSell*monitor.Config().DeltaCoefficient //DeltaCoefficient 默认为0.4,手动系数,因为计算出来的值会偏小
	// 买入成本
	cost := ((bPool.Price + targetPrice) / 2) * deltaSell * (1.0 + bPool.Fee)
	// 卖出收益
	proceeds := ((aPool.Price + targetPrice) / 2) * deltaSell * (1.0 - aPool.Fee)
	// 利润
	profit = proceeds - cost
	info := fmt.Sprintf("%s cost: %f, proceeds: %f, profit: %f, deltaSell: %f,pa: %f, pb: %f, fa: %f, fb: %f\n", bPool.Symbol, cost, proceeds, profit, deltaSell, aPool.Price, bPool.Price, aPool.Fee, bPool.Fee)
	monitor.Logger().Debug(info)
	return
}

func (m *MovingBrick) Do(monitor dt.IMonitor, arbitrage *dt.Arbitrage) {
	monitor.Logger().Debug("Do: ", arbitrage)
	go monitor.SendToTG(fmt.Sprintf("%s: BuyPrice: %.6f, SellPrice: %.6f, Amount: %.6f, Estimated: %.4f, Block: %d, BuyPool: %s, SellPool: %s",
		arbitrage.BuyPool.Symbol, arbitrage.BuyPool.Price, arbitrage.SellPool.Price, arbitrage.Amount, arbitrage.ProfitUSD, arbitrage.BlockNumber,
		arbitrage.BuyPool.Pool, arbitrage.SellPool.Pool))

	params := dt.SwapParams{
		BuyPool:   arbitrage.BuyPool.Pool,
		SellPool:  arbitrage.SellPool.Pool,
		Amount:    arbitrage.Amount,
		Deadline:  arbitrage.BlockNumber + 1,
		BuyFee:    uint16(arbitrage.BuyPool.Fee * 1e4),
		SellFee:   uint16(arbitrage.SellPool.Fee * 1e4),
		GasPrice:  arbitrage.GasPrice,
		Borrow:    arbitrage.Borrow,
		BaseToken: arbitrage.BaseToken,
		Position:  arbitrage.Position,
	}
	monitor.DoSwap(params)
}
