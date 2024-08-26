package types

import "time"

type Pool struct {
	Factory string `bson:"factory"`
	Token0  Token  `bson:"token0"`
	Token1  Token  `bson:"token1"`
	Address string `bson:"address"`
}

type SimplePool struct {
	Factory string `bson:"factory"`
	Token0  string `bson:"token0"`
	Token1  string `bson:"token1"`
	Address string `bson:"address"`
}

type Token struct {
	Address     string  `bson:"address"`
	Name        string  `bson:"name"`
	TotalSupply float64 `bson:"totalSupply"`
	Decimals    uint64  `bson:"decimals"`
	Symbol      string  `bson:"symbol"`
}

type Pair struct {
	Pool        string  `bson:"pool"`
	Symbol      string  `bson:"symbol"`
	Price       float64 `bson:"price"`
	Reserve0    float64 `bson:"reserve0"`
	Reserve1    float64 `bson:"reserve1"`
	BlockNumber uint64  `bson:"blockNumber"`
	Token0      string  `bson:"token0"`
	Token1      string  `bson:"token1"`
	DexName     string  `bson:"dexName"`
	Fee         float64 `bson:"fee"`
	UpdateTimes int32   `bson:"updateTimes,omitempty"`
}

type Pairs []*Pair

type Arbitrage struct {
	BuyPool     Pair
	SellPool    Pair
	Amount      float64
	ProfitUSD   float64
	BlockNumber uint64
	GasPrice    float64
	Borrow      string
	Position    uint8
	BaseToken   string
}

func (p Pairs) Len() int           { return len(p) }
func (p Pairs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Pairs) Less(i, j int) bool { return p[i].Price < p[j].Price }

// 只用于对交易对去重，token0/token1、token1/token0都试为一个交易对
func (x SimplePool) Equal(y SimplePool) bool {
	b := (x.Token0 == y.Token0 && x.Token1 == y.Token1) || (x.Token0 == y.Token1 && x.Token1 == y.Token0)
	return b
}
func (p Pool) Equal(y Pool) bool {
	return p.Address == y.Address
}

type Transaction struct {
	Tx         string    `bson:"tx"`
	Ok         bool      `bson:"ok"`
	Simulation bool      `bson:"simulation"`
	Cost       float64   `bson:"cost"`
	Income     float64   `bson:"income"`
	Confirm    bool      `bson:"confirm"`
	BuyPool    string    `bson:"buy_pool"`
	SellPool   string    `bson:"sell_pool"`
	UseGas     uint64    `bson:"use_gas,omitempty"`
	GasPrice   uint64    `bson:"gas_price"`
	BaseToken  string    `bson:"base_token"`
	EventBlock uint64    `bson:"event_block"`
	CreatedAt  time.Time `bson:"created_at"`
	Error      string    `bson:"error"`
}
