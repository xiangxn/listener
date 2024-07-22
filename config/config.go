package config

import (
	"encoding/json"
	"os"
)

type DexConfig struct {
	Name    string  `json:"name"`
	Event   string  `json:"event"`
	Topic   string  `json:"topic"`
	Factory string  `json:"factory"`
	Fee     float64 `json:"fee"`
}

type TGConfig struct {
	ChatID string `json:"chat_id"`
	Token  string `json:"token"`
}

type Configuration struct {
	NetName string `json:"net_name"`
	Dburl   string `json:"dburl"`
	Rpcs    struct {
		// 合适的值:["","alchemy","flashbot"]
		Flashbots string `json:"flashbots"`
		Http      string `json:"http"`
		Ws        string `json:"ws"`
	} `json:"rpcs"`
	Simulation struct {
		Enable bool   `json:"enable"`
		Funds  string `json:"funds"`
	} `json:"simulation"`
	Strategies struct {
		BaseTokens map[string][]string `json:"base_tokens"`
	} `json:"strategies"`
	// 事件等待时间，单位毫秒
	EventWaitingTime uint32  `json:"event_waiting_time"`
	GasPrice         float64 `json:"gas_price"`
	// gas的倍数
	GasTimes float64 `json:"gas_times"`
	GasLimit uint64  `json:"gas_limit"`
	EIP1559  bool    `json:"eip1559"`
	// 交易合约地址
	TraderContract string `json:"trader_contract"`
	//基础token的最小储备量，如ETH
	BaseMinReserve   float64     `json:"base_min_reserve"`
	ChunkLength      int         `json:"chunk_length"`
	MaxConcurrent    int         `json:"max_concurrent"`
	Debug            bool        `json:"debug"`
	Dexs             []DexConfig `json:"dexs"`
	MinProfitUSD     float64     `json:"min_profit_usd"`
	DeltaCoefficient float64     `json:"delta_coefficient"`
	TG               TGConfig    `json:"tg"`
}

func GetConfig(fileName string) (conf Configuration) {
	file, _ := os.Open(fileName)
	defer file.Close()
	conf = Configuration{}
	err := json.NewDecoder(file).Decode(&conf)
	if err != nil {
		panic(err)
	}
	return
}
