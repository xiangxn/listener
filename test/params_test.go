package main

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// go test -v -run ^TestParams$ github.com/xiangxn/listener/test
func TestParams(t *testing.T) {
	// 0x8119c065
	// 00000000000000000000000011b815efb8f581194ae79006d24e0d814b7697f6
	// 00000000000000000000000074c99f3f5331676f6aec2756e1f39b4fc029a83e
	// 000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
	// 00000000000000000000000088e6a0c2ddd26feeb64f039a2c41296fcb3f5640
	// 0000000000000000000000000000000000000000000000000000000000989680
	// 0000000000000000000000000000000000000001350ec10100010002001e0019

	hash := crypto.Keccak256Hash([]byte("swap()")).Hex()
	methodID := hash[:10]
	tmp := new(big.Int).Lsh(big.NewInt(int64(20254401)), 72)
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(1)), 64))
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(1)), 48))
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(2)), 32))
	tmp = tmp.Or(tmp, new(big.Int).Lsh(big.NewInt(int64(30)), 16))
	tmp = tmp.Or(tmp, big.NewInt(int64(25)))

	var data []byte
	data = append(data, hexutil.MustDecode(methodID)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress("0x11b815efB8f581194ae79006d24E0d814B7697F6").Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress("0x74C99F3f5331676f6AEc2756e1F39b4FC029a83E").Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2").Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(common.HexToAddress("0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640").Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(10000000).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(tmp.Bytes(), 32)...)

	fmt.Println(hexutil.Encode(data))

}
