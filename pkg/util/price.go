package util

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
