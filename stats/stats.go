package stats

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/elliotchance/pie/v2"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/database"
	dt "github.com/xiangxn/listener/types"
)

type Stats struct {
	DB         dt.IActions
	Conf       config.Configuration
	baseTokens map[string]dt.Token
}

func New(conf config.Configuration) (stats *Stats) {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if conf.Debug {
		l.Level = logrus.DebugLevel
	}
	stats = &Stats{
		Conf: conf,
		DB: &database.Actions{
			DB:     database.GetClient(conf).Database(fmt.Sprintf("%slistener", conf.NetName)),
			Mctx:   context.Background(),
			Logger: l,
		},
	}
	stats.init()
	return
}

func (s *Stats) init() {
	if len(s.baseTokens) == 0 {
		s.baseTokens = make(map[string]dt.Token)
	}
	baseTokens := pie.Keys(s.Conf.Strategies.BaseTokens)
	ts := s.DB.GetTokens(baseTokens)
	for _, t := range ts {
		s.baseTokens[t.Address] = t
	}
}

func (s *Stats) isUSD(token string) bool {
	return strings.Contains(s.baseTokens[token].Symbol, "USD")
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
	gasPrice := s.DB.GetBasePrice(s.Conf.Strategies.GasToken.Base, s.Conf.Strategies.GasToken.Quote)
	fmt.Println("\n================统计结果================")
	fmt.Printf("时间从[%s]到[%s]\n", start.Format(time.DateTime), end.Format(time.DateTime))
	for key, amount := range coins {
		fmt.Printf("\t%s: %.8f\n", key, amount)
		if key == s.Conf.Strategies.GasToken.Base { //amount为与gas一样的token
			income += gasPrice * amount
		} else if s.isUSD(key) { //amount为USD
			income += amount
		} else {
			basePrice := s.DB.GetBasePrice(key, s.Conf.Strategies.GasToken.Quote)
			quotePrice := gasPrice / basePrice
			income += quotePrice * amount
		}
	}
	totalGas := float64(gas) / math.Pow(10, 18)
	totalGas = totalGas * gasPrice
	fmt.Printf("消耗金额: %.8f\n", totalGas)
	fmt.Printf("毛利金额: %.8f\n", income)
	fmt.Printf("净利金额: %.8f\n", income-totalGas)
	fmt.Printf("成功数量: %d\n", success)
	fmt.Printf("失败数量: %d\n", failure)
	fmt.Printf("已确认数量: %d\n", confirmed)
	fmt.Printf("未确认数量: %d\n", unconfirmed)
}
