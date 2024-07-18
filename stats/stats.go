package stats

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/elliotchance/pie/v2"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/database"
	"github.com/xiangxn/listener/strategies"
	dt "github.com/xiangxn/listener/types"
)

type Stats struct {
	DB dt.IActions
}

func New(conf config.Configuration) *Stats {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if conf.Debug {
		l.Level = logrus.DebugLevel
	}
	return &Stats{
		DB: &database.Actions{
			DB:     database.GetClient(conf).Database(fmt.Sprintf("%slistener", conf.NetName)),
			Mctx:   context.Background(),
			Logger: l,
		},
	}
}

func (s *Stats) getETHPrice() (price float64) {
	baseToken := strategies.BaseTokens[0]
	quoteToken := strategies.BaseTokens[1]
	data := s.DB.GetPairsByTokens([]string{baseToken, quoteToken})
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

func (s *Stats) SearchTransacttion(simulation bool, start time.Time, end time.Time) {
	// fmt.Println(simulation, start, end)
	txs := s.DB.SearchTransacttion(simulation, start, end)
	coins := make(map[string]float64)
	gas := float64(0)
	income := float64(0)
	success := 0
	failure := 0
	confirmed := 0
	unconfirmed := 0
	for _, tx := range txs {
		if tx.BaseToken != "" {
			if tx.Confirm {
				if tx.Ok {
					coins[tx.BaseToken] += tx.Income
					success += 1
				} else {
					coins[tx.BaseToken] += 0
					failure += 1
				}
				confirmed += 1
			} else {
				unconfirmed += 1
			}
		}
		gas += float64(tx.UseGas) * float64(tx.GasPrice)
	}
	if len(coins) < 1 {
		fmt.Println("还没有数据!")
		return
	}
	ethPrice := s.getETHPrice()
	fmt.Println("\n================统计结果================")
	fmt.Printf("时间从[%s]到[%s]\n", start.Format(time.DateTime), end.Format(time.DateTime))
	for key, amount := range coins {
		fmt.Printf("\t%s: %.8f\n", key, amount)
		if key == strategies.BaseTokens[0] { //amount为ETH
			income += ethPrice * amount
		} else { //amount为USD
			income += amount
		}
	}
	totalGas := float64(gas) / math.Pow(10, 18)
	totalGas = totalGas * ethPrice
	fmt.Printf("消耗金额: %.8f\n", totalGas)
	fmt.Printf("毛利金额: %.8f\n", income)
	fmt.Printf("净利金额: %.8f\n", income-totalGas)
	fmt.Printf("成功数量: %d\n", success)
	fmt.Printf("失败数量: %d\n", failure)
	fmt.Printf("已确认数量: %d\n", confirmed)
	fmt.Printf("未确认数量: %d\n", unconfirmed)
}
