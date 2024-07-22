package strategies

import (
	"fmt"
	"log"
	"sort"

	"github.com/elliotchance/pie/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/go-multicall"
	dt "github.com/xiangxn/listener/types"
)

var BaseTokens = []string{
	// WETH
	"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
	// USDC
	"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
	// USDT
	"0xdAC17F958D2ee523a2206206994597C13D831ec7"}

const (
	// USD
	USDCUSDT = "0x3416cF6C708Da44DB2624D63ea0AAef7113527C6"
	DAIUSDT  = "0x48DA0965ab2d2cbf1C17C09cFB5Cbe67Ad5B1406"
	DAIUSDC  = "0x5777d92f208679DB4b9778590Fa3CAB3aC9e2168"

	//ETH
	ETHUSDT = "0x11b815efB8f581194ae79006d24E0d814B7697F6"
	USDCETH = "0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640"
)

var BorrowPools = []string{USDCUSDT, DAIUSDT, DAIUSDC, ETHUSDT, USDCETH}

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
	return token == BaseTokens[1] || token == BaseTokens[2]
}

func (m *MovingBrick) GetBaseToken(token0, token1 string) string {
	if pie.Contains(BaseTokens, token0) {
		return token0
	} else if pie.Contains(BaseTokens, token1) {
		return token1
	}
	return ""
}

// 获取基础token报价的平均价格,默认:ETH/USDC
func (m *MovingBrick) GetBasePrice(monitor dt.IMonitor, baseToken string) (price float64) {
	var quoteToken string
	if baseToken == BaseTokens[0] { // 如果baseToken是ETH, quoteToken则用USDC
		quoteToken = BaseTokens[1]
	} else { // 如果baseToken是USDC或USDT, quoteToken则用ETH
		quoteToken = BaseTokens[0]
	}
	data := monitor.DB().GetPairsByTokens([]string{baseToken, quoteToken})
	if len(data) < 1 {
		return
	}
	var total float64
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
		total += p.Price
	})
	price = total / float64(len(data))
	return
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
		var gasUSD float64
		// ETH/USDC的平均价格,如果baseToken是ETH时价格表示为ETH/USDC,如果是USDC时价格表示为USDC/ETH
		avgBasePrice := m.GetBasePrice(monitor, baseToken)
		gas := monitor.GetUseGas(buyPool, sellPool, amount)
		borrow, position := m.getBorrowPool(buyPool, sellPool, baseToken)
		if m.isUSD(baseToken) {
			profitUSD = profit / avgPrice

			gasUSD = float64(gas) * gasPrice / avgBasePrice
		} else {
			// 报价token相对于QuoteToken的价格
			quotePrice := avgBasePrice / avgPrice
			profitUSD = quotePrice * profit
			gasUSD = float64(gas) * gasPrice * avgBasePrice
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
