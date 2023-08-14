package util

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

func TestA(t *testing.T) {
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

	println(randomString)
}

func TestB2(t *testing.T) {
	// target 7
	f := 0.0000236

	//2.36e-05
	fStr := fmt.Sprintf("%v", f)
	if strings.Contains(fStr, "e-") {
		ary := strings.Split(fStr, "e-")
		front := ary[0]
		end := ary[1]
		frontNum := len(strings.Split(front, ".")[1])
		endNum, _ := strconv.ParseInt(end, 10, 64)
		log.Printf("frontNum: %v\n", frontNum)
		log.Printf("endNum: %v\n", endNum)
	}
}
