package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	event := types.Log{
		Address:     common.HexToAddress("0xd8C8a2B125527bf97c8e4845b25dE7e964468F77"),
		BlockNumber: 20259344,
	}

	cfg := config.GetConfig("../config.yaml")
	// fmt.Println(cfg)

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
	monitor.TestEvent(event)
}
