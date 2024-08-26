package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/go-multicall"
	"github.com/xiangxn/listener/config"
	"go.mongodb.org/mongo-driver/mongo"
)

type TelegramRequestBody struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type SwapParams struct {
	BuyPool     string
	SellPool    string
	Amount      float64
	BlockNumber uint64
	Deadline    uint64
	BuyType     uint8
	SellType    uint8
	BuyFee      uint16
	SellFee     uint16
	GasPrice    float64
	Borrow      string
	Position    uint8
	BaseToken   string
}

type Options struct {
	Cfg     config.Configuration
	Handler EventHandler // 业务(策略)处理handler
	Logger  logrus.FieldLogger
	Cipher  [32]byte
}

type DB = mongo.Database
type MC = multicall.Caller

type ResAddress struct {
	common.Address
}

type ResString struct {
	Result string
}

type ResBigInt struct {
	*big.Int
}

type ResHash struct {
	common.Hash
}
