package util

import (
	"crypto/rand"
	"math/big"
	"strings"
)

func GenerateOrder() string {
	n := 15 // 生成的字符串长度
	// 定义字母和数字的字符集
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// 生成随机字符串
	result := make([]string, n)
	for i := range result {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = string(charset[randomIndex.Int64()])
	}
	// 将结果连接成字符串
	randomString := strings.Join(result, "")
	return randomString
}
