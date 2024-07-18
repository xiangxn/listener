package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"github.com/xiangxn/listener/simulation"
)

// go test -v -run ^TestSimulation$ github.com/xiangxn/listener/test
func TestSimulation(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}
	richAddress := "0x8EB8a3b98659Cce290402893d0123abb75E3ab28"
	testAddress := "0x000000e280780A6925C5376B6ccF5be200d066a4"
	wethAddress := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"

	privateKey := os.Getenv("PRIVATE_KEY")
	rpcURL := os.Getenv("RPC_MAINNET")
	port := simulation.RandomPort()

	block := uint64(20194874)

	ctx, cancel := context.WithCancel(context.Background())
	simulation.StartAnvil(ctx, rpcURL, block, port)
	defer cancel()

	simulation.WaitForAnvil(port)

	client := simulation.GetClient(port)
	balanceETH, _ := client.BalanceAt(context.Background(), common.HexToAddress(richAddress), big.NewInt(int64(block)))
	fmt.Println("richAddress balance ETH: ", balanceETH)

	simulation.Impersonate(port, richAddress)
	simulation.ImpersonateTransfer(port, wethAddress, richAddress, testAddress, 0.1, 18)
	simulation.ImpersonateTransferETH(port, richAddress, testAddress, 1.0)
	simulation.StopImpersonate(port, richAddress)

	simulation.DeployTrader(port, simulation.GetPrivateKey(privateKey))

	balance := simulation.BalanceOf(client, wethAddress, testAddress)
	fmt.Println("testAddress balance: ", balance)

	balanceETH, _ = client.BalanceAt(context.Background(), common.HexToAddress(testAddress), big.NewInt(int64(block)))
	fmt.Println("testAddress balance ETH: ", balanceETH)
}
