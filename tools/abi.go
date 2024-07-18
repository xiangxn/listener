package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func ReadABI(name string) *abi.ABI {
	cPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	cPath = strings.TrimSuffix(cPath, "/test")
	cPath = fmt.Sprintf("%s/abis/%s.json", cPath, name)
	abiData, err := os.ReadFile(cPath)
	if err != nil {
		fmt.Println("os.ReadFile error ,", err)
		return nil
	}
	contractAbi, err := abi.JSON(bytes.NewReader(abiData))
	if err != nil {
		fmt.Println("abi.JSON error ,", err)
		return nil
	}
	return &contractAbi
}

func ReadABIString(name string) string {
	cPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	cPath = strings.TrimSuffix(cPath, "/test")
	cPath = fmt.Sprintf("%s/abis/%s.json", cPath, name)
	abiData, err := os.ReadFile(cPath)
	if err != nil {
		fmt.Println("os.ReadFile error ,", err)
		return ""
	}
	return string(abiData)
}

func SaveJson(name string, obj interface{}) {
	file, err := os.Create(name)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
	// 创建一个JSON编码器
	encoder := json.NewEncoder(file)

	// 将对象编码为JSON并写入文件
	if err := encoder.Encode(obj); err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
}
