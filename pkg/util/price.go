package util

import (
	"fmt"
	"strconv"
	"strings"
)

// AdjustClosePosSize 调整平仓的size, 因为要反向操作，size在原基础上 * 1.0002
func AdjustClosePosSize(size float64, side string, cexName string) string {

	sizeAdjusted := 0.0
	if side == "sell" {
		//如果是 oke的话，买的时候被扣除了 0.1%, 所以没那么多卖,而且oke可以一键平仓
		if cexName == "oke" {
			sizeAdjusted = size * (1 - 0.001)
			//保留三位有效数字
			return NumTrunc(sizeAdjusted)
		} else {
			return NumTrunc(size)
		}
	} else {
		// 买币还债
		sizeAdjusted = size * (1 + 0.0002) / (1 - 0.001)
	}
	// 默认保留 4位
	return NumTrunc(sizeAdjusted)
}

// AdjustPrice 调整价格 千分之一便于成交
func AdjustPrice(price float64, side string) string {
	priceStr := fmt.Sprintf("%v", price)
	// 多加一位精确度
	decimalLen := calDecimalLen(priceStr) + 1

	priceAdjusted := 0.0
	if side == "buy" {
		priceAdjusted = price * (1 + 0.002)
	} else {
		priceAdjusted = price * (1 - 0.002)
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
