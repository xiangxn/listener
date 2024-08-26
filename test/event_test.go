package main

import (
	"math/big"
	"testing"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/monitor"
	"github.com/xiangxn/listener/strategies"
	dt "github.com/xiangxn/listener/types"
)

// go test -v -run ^TestEvent$ github.com/xiangxn/listener/test
func TestEvent(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}

	cfg := config.GetConfig("../bsc.config.yaml")
	// fmt.Println(cfg)

	BlockNumber := big.NewInt(40756466).Uint64()

	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if cfg.Debug {
		l.Level = logrus.DebugLevel
	}

	opt := &dt.Options{
		Cfg:     cfg,
		Handler: &strategies.MovingBrick{},
		Logger:  l,
	}
	monitor, err := monitor.New(opt)
	if err != nil {
		panic(err)
	}
	eventPool := monitor.DB().GetSimplePool("0x4f55423de1049d3CBfDC72f8A40f8A6f554f92aa")
	monitor.TestEvent(eventPool, BlockNumber)
}
