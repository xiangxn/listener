package main

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"testing"

	"github.com/xiangxn/listener/tools"
)

// go test -v -run ^TestCrypto$ github.com/xiangxn/listener/test
func TestCrypto(t *testing.T) {
	password := "12345678"

	// 对密码进行 SHA-256 哈希处理
	key := sha256.Sum256([]byte(password))

	// 明文
	plaintext := []byte("This is a secret message")

	// 加密
	ciphertext, err := tools.Encrypt(plaintext, key[:])
	if err != nil {
		fmt.Println("Error encrypting:", err)
		return
	}

	b32 := base32.StdEncoding.EncodeToString(ciphertext)
	// fmt.Printf("Ciphertext: %s\n", base64.StdEncoding.EncodeToString(ciphertext))
	fmt.Printf("Ciphertext: %s\n", b32)

	ciphertext, err = base32.StdEncoding.DecodeString(b32)
	if err != nil {
		fmt.Println("Base32 error decrypting:", err)
		return
	}
	// 解密
	decryptedText, err := tools.Decrypt(ciphertext, key[:])
	if err != nil {
		fmt.Println("Error decrypting:", err)
		return
	}

	fmt.Printf("Decrypted text: %s\n", decryptedText)
}
