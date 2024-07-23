package config

import (
	"encoding/json"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type DexConfig struct {
	Name    string  `json:"name" yaml:"name"`
	Event   string  `json:"event" yaml:"event"`
	Topic   string  `json:"topic" yaml:"topic"`
	Factory string  `json:"factory" yaml:"factory"`
	Fee     float64 `json:"fee" yaml:"fee"`
}

type TGConfig struct {
	ChatID string `json:"chat_id" yaml:"chat_id"`
	Token  string `json:"token" yaml:"token"`
}

type Configuration struct {
	NetName string `json:"net_name" yaml:"net_name"`
	Dburl   string `json:"dburl" yaml:"dburl"`
	Rpcs    struct {
		// 合适的值:["","alchemy","flashbot"]
		Flashbots string `json:"flashbots" yaml:"flashbots"`
		Http      string `json:"http" yaml:"http"`
		Ws        string `json:"ws" yaml:"ws"`
	} `json:"rpcs" yaml:"rpcs"`
	Simulation struct {
		// 是否开起模拟交易
		Enable bool `json:"enable" yaml:"enable"`
		// 用于模拟交易时给测试地址支持gas基础token
		Funds string `json:"funds" yaml:"funds"`
	} `json:"simulation" yaml:"simulation"`
	Strategies struct {
		// 配置要使用的base token, 键为base token的地址，值为可以借贷basetoken的交易池
		BaseTokens map[string][]string `json:"base_tokens" yaml:"base_tokens"`
	} `json:"strategies" yaml:"strategies"`
	// 事件等待时间，单位毫秒
	EventWaitingTime uint32  `json:"event_waiting_time" yaml:"event_waiting_time"`
	GasPrice         float64 `json:"gas_price" yaml:"gas_price"`
	// gas的倍数
	GasTimes float64 `json:"gas_times" yaml:"gas_times"`
	GasLimit uint64  `json:"gas_limit" yaml:"gas_limit"`
	EIP1559  bool    `json:"eip1559" yaml:"eip1559"`
	// 交易合约地址
	TraderContract string `json:"trader_contract" yaml:"trader_contract"`
	//基础token的最小储备量，如ETH
	BaseMinReserve float64 `json:"base_min_reserve" yaml:"base_min_reserve"`
	// 对批量请求分组时分组的大小
	ChunkLength int `json:"chunk_length" yaml:"chunk_length"`
	// 最大并发数据
	MaxConcurrent int `json:"max_concurrent" yaml:"max_concurrent"`
	// 是否开起高度
	Debug bool `json:"debug" yaml:"debug"`
	// 受支持的交易对
	Dexs []DexConfig `json:"dexs" yaml:"dexs"`
	// 最小收益,以USD计算
	MinProfitUSD float64 `json:"min_profit_usd" yaml:"min_profit_usd"`
	// 为参与交易的basetoken设置倍数
	DeltaCoefficient float64 `json:"delta_coefficient" yaml:"delta_coefficient"`
	// TG消息服务配置
	TG TGConfig `json:"tg" yaml:"tg"`
}

func GetConfig(fileName string) (conf Configuration) {
	if strings.HasSuffix(fileName, ".yaml") {
		conf = GetConfigYAML(fileName)
	} else {
		conf = GetConfigJSON(fileName)
	}
	return
}

func GetConfigJSON(fileName string) (conf Configuration) {
	file, _ := os.Open(fileName)
	defer file.Close()
	conf = Configuration{}
	err := json.NewDecoder(file).Decode(&conf)
	if err != nil {
		panic(err)
	}
	return
}

func GetConfigYAML(fileName string) (conf Configuration) {
	file, _ := os.Open(fileName)
	defer file.Close()

	conf = Configuration{}
	err := yaml.NewDecoder(file).Decode(&conf)
	if err != nil {
		panic(err)
	}
	return
}
