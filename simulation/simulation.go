package simulation

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/xiangxn/listener/trader"
)

func RandomPort() (randomNum uint32) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 定义范围的上下界
	min := 8545
	max := 9545
	// 生成随机数
	randomNum = uint32(r.Intn(max-min+1) + min)
	return
}

func GetURL(port uint32) (url string) {
	url = fmt.Sprintf("http://localhost:%d", port)
	return
}

func GetClient(port uint32) *ethclient.Client {
	httpClient, err := ethclient.Dial(GetURL(port))
	if err != nil {
		log.Fatalf("Failed to get ethclient: %v", err)
	}
	return httpClient
}

func StartAnvil(parent context.Context, rpcUrl string, block uint64, port uint32) (cmd *exec.Cmd) {
	cmd = exec.CommandContext(parent, "anvil",
		"-f", rpcUrl,
		"-p", fmt.Sprint(port),
		"--fork-block-number", fmt.Sprint(block),
		"--no-rate-limit",
		// "--no-mining",
		"--balance", "1000")
	// 获取标准输出管道
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating stdout pipe: %v", err)
	}
	// 获取标准错误管道
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Error creating stderr pipe: %v", err)
	}
	// 使用 goroutine 来异步读取 stdout
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			fmt.Printf("stdout: %s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stdout: %v", err)
		}
	}()
	// 使用 goroutine 来异步读取 stderr
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			fmt.Printf("stderr: %s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stderr: %v", err)
		}
	}()
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start Anvil: %v", err)
	}
	return
}

func WaitForAnvil(port uint32) {
	for {
		client := GetClient(port)
		_, err := client.ChainID(context.Background())
		if err == nil {
			break
		}
		fmt.Println("Waiting for Anvil to start...", err)
		time.Sleep(500 * time.Millisecond)
	}
}

// 冒充指定地址
func Impersonate(port uint32, account string) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "anvil_impersonateAccount",
		"params":  []string{account},
		"id":      1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal JSON payload: %v", err)
	}
	resp, err := http.Post(GetURL(port), "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Fatalf("Failed to send POST request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %v", resp.StatusCode)
	}
	fmt.Println("Impersonation successful for account:", account)
}

// 停止冒充指定地址
func StopImpersonate(port uint32, account string) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "anvil_stopImpersonatingAccount",
		"params":  []string{account},
		"id":      1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal JSON payload: %v", err)
	}
	resp, err := http.Post(GetURL(port), "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Fatalf("Failed to send POST request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %v", resp.StatusCode)
	}
	fmt.Println("StopImpersonation successful for account:", account)
}

func GetPrivateKey(privateKey string) *ecdsa.PrivateKey {
	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		log.Fatalf("Failed to create PrivateKey: %v", err)
	}
	return key
}

func GetAddress(pKey *ecdsa.PrivateKey) common.Address {
	publicKey := pKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Invalid private key")
	}
	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

// 冒充指定地址给to转erc20
func ImpersonateTransfer(port uint32, erc20, from, to string, amount float64, desc uint64) {
	rpcUrl := GetURL(port)
	cmd := exec.Command("cast", "send", erc20,
		"-r", rpcUrl,
		"--unlocked",
		"-f", from,
		"transfer(address,uint256)(bool)",
		to,
		fmt.Sprintf("%d", uint64(amount*math.Pow(10, float64(desc)))))
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to call cast send: %v", err)
	}
	cmd.Process.Kill()
	fmt.Println("ImpersonateTransfer successful to:", to, amount)
}

// 冒充指定地址给to转erc20
func ImpersonateTransferETH(port uint32, from, to string, amount float64) {
	rpcUrl := GetURL(port)
	cmd := exec.Command("cast", "send",
		"-r", rpcUrl,
		"--unlocked",
		"-f", from,
		to,
		"--value", fmt.Sprintf("%fether", amount))
	// var stdout, stderr bytes.Buffer
	// cmd.Stdout = &stdout
	// cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to call cast send: %v", err)
		// log.Fatalf("Failed to call cast send: %v %s", err, stderr.String())
	}
	cmd.Process.Kill()
	fmt.Println("ImpersonateTransferETH successful to:", to, amount)
}

func Transfer(client *ethclient.Client, erc20, privateKey, to string, amount *big.Int) (txHash string) {
	hash := crypto.Keccak256Hash([]byte("transferFrom(address,address,uint256)")).Hex()
	methodID := hash[:10]
	pKey := GetPrivateKey(privateKey)
	from := GetAddress(pKey)

	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		log.Fatal(err)
		return
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
		return
	}
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatal(err)
		return
	}

	var data []byte
	data = append(data, hexutil.MustDecode(methodID)...)
	data = append(data, common.LeftPadBytes(from.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress(to).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(amount.Bytes(), 32)...)

	tx := types.NewTransaction(nonce, common.HexToAddress(erc20), big.NewInt(0), uint64(300000), gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), pKey)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}
	return signedTx.Hash().Hex()
}

func BalanceOf(client *ethclient.Client, erc20, account string) *big.Int {
	abiStr := `[{"constant":true,"inputs":[{"name":"who","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
	contractABI, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		log.Fatal(err)
	}
	data, err := contractABI.Pack("balanceOf", common.HexToAddress(account))
	if err != nil {
		log.Fatal(err)
	}
	contractAddress := common.HexToAddress(erc20)
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{To: &contractAddress, Data: data}, nil)
	if err != nil {
		log.Fatal(err)
	}
	var returnValue *big.Int
	err = contractABI.UnpackIntoInterface(&returnValue, "balanceOf", result)
	if err != nil {
		log.Fatal(err)
	}
	return returnValue
}

func GetReceipt(client *ethclient.Client, hash common.Hash) *types.Receipt {
	receipt, err := client.TransactionReceipt(context.Background(), hash)
	if err != nil {
		if err == ethereum.NotFound {
			fmt.Println(hash.Hex(), " Not yet on the chain or not synchronized.")
		} else {
			fmt.Println("GetReceipt error ", err)
		}
		return nil
	}
	return receipt
}

func GetRevert(ctx context.Context, client *ethclient.Client, receipt *types.Receipt, stx *types.Transaction) (result []byte, errMsg string) {
	if stx == nil {
		tx, _, _ := client.TransactionByHash(ctx, receipt.TxHash)
		stx = tx
	}
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: stx.To(), Data: stx.Data()}, receipt.BlockNumber)
	if err != nil {
		const errHead = "execution reverted: revert: "
		errMsg = err.Error()
		if strings.Contains(errMsg, errHead) {
			em, _ := strings.CutPrefix(errMsg, errHead)
			errMsg = em
		}
	}
	return
}

func DeployTrader(port uint32, pKey *ecdsa.PrivateKey) (addr string) {
	client := GetClient(port)
	fromAddress := GetAddress(pKey)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Failed to get chainId: %v", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(pKey, chainId)
	if err != nil {
		log.Fatalf("Failed to create transactor: %v", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	auth.GasPrice = gasPrice

	address, tx, _, err := trader.DeployTrader(auth, client)
	if err != nil {
		log.Fatalf("Failed to deploy new token contract: %v", err)
	}
	fmt.Printf("Contract deployed to: %s\n", address.Hex())
	fmt.Printf("Transaction: %s\n", tx.Hash().Hex())
	addr = address.Hex()
	return
}

func Mine(port uint32) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "evm_mine",
		"params":  []string{},
		"id":      1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal JSON payload: %v", err)
	}
	resp, err := http.Post(GetURL(port), "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Fatalf("Failed to send POST request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %v", resp.StatusCode)
	}
	fmt.Println("Mine successful")
}
