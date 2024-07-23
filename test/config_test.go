package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/xiangxn/listener/config"
	"gopkg.in/yaml.v3"
)

// go test ^TestConfig$ github.com/xiangxn/listener/test
func TestConfig(t *testing.T) {
	conf := config.GetConfig("../bsc.config.json")

	file, err := os.Create("../bsc.config.yaml")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	yamlData, err := yaml.Marshal(&conf)
	if err != nil {
		fmt.Printf("Error marshaling to YAML: %v\n", err)
		return
	}

	_, err = file.Write(yamlData)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}
}
