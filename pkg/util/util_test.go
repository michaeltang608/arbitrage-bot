package util

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
)

func TestA(t *testing.T) {
	println(GenerateOrder("O"))
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
