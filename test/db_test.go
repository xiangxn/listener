package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/xiangxn/listener/config"
	"github.com/xiangxn/listener/database"
)

// go test -v -run ^TestDB$ github.com/xiangxn/listener/test
func TestDB(t *testing.T) {
	cfg := config.GetConfig("../config.yaml")
	db := database.GetClient(cfg).Database(fmt.Sprintf("%slistener", cfg.NetName))
	ctx := context.Background()

	DB := database.Actions{
		DB:     db,
		Mctx:   ctx,
		Logger: logrus.New(),
	}

	min, max := DB.GetGas("0xc5be99A02C6857f9Eac67BbCE58DF5572498F40c", "0xCb2286d9471cc185281c4f763d34A962ED212962")
	fmt.Printf("min:%d, max:%d\n", min, max)
}

// go test -v -run ^TestGetPoolsByTokens$ github.com/xiangxn/listener/test
func TestGetPoolsByTokens(t *testing.T) {
	cfg := config.GetConfig("../bsc.config.yaml")
	db := database.GetClient(cfg).Database(fmt.Sprintf("%slistener", cfg.NetName))
	ctx := context.Background()

	DB := database.Actions{
		DB:     db,
		Mctx:   ctx,
		Logger: logrus.New(),
	}

	data := DB.GetPoolsByTokens([]string{"0x2Bf83D080d8Bc4715984e75E5b3D149805d11751", "0x55d398326f99059fF775485246999027B3197955"})
	fmt.Println(len(data), data)
}
