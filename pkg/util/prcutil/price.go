package prcutil

import (
	"fmt"
	"strconv"
	"strings"
)

// AdjustPriceFloat 调整价格, pips 基准点
func AdjustPriceFloat(price float64, bigger bool, pips float64) float64 {
	// 多加一位精确度
	priceAdjusted := 0.0
	if bigger {
		priceAdjusted = price * (1 + 0.0001*pips)
	} else {
		priceAdjusted = price * (1 - 0.0001*pips)
	}
	return priceAdjusted
}

func AdjustPrice(price float64, side string, curDiff float64) string {
	priceStr := fmt.Sprintf("%v", price)
	// 多加一位精确度
	decimalLen := calDecimalLen(priceStr) + 1

	step := curDiff * 0.001
	if step <= 0.002 {
		step = 0.002
	}
	priceAdjusted := 0.0
	if side == "buy" {
		priceAdjusted = price * (1 + step)
	} else {
		priceAdjusted = price * (1 - step)
	}
	return strconv.FormatFloat(priceAdjusted, 'f', decimalLen, 64)
}

func calDecimalLen(fStr string) int {
	if strings.Contains(fStr, "e-") {
		ary := strings.Split(fStr, "e-")
		front := ary[0]
		end := ary[1]
		frontNum := len(strings.Split(front, ".")[1])
		endNum, _ := strconv.ParseInt(end, 10, 64)
		return frontNum + int(endNum)
	} else {
		return len(strings.Split(fStr, ".")[1])
	}
}
