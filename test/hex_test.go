package main

import (
	"fmt"
	"testing"
)

func TestHex(t *testing.T) {
	var num uint64 = 10000000000
	hexStr := fmt.Sprintf("0x%x", num)
	fmt.Println(hexStr)
}
