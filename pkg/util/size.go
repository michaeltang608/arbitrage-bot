package util

import (
	"fmt"
	"math"
)

// NumTrunc 将一个浮点数保留三位有效数字， 如 12233.44-》12200， 0.22344-》0.223
func NumTrunc(f float64) string {
	origin := f
	isBig := true
	if f < 1 {
		isBig = false
	}
	cnt := 0
	cnt = recur(f, isBig, cnt)
	if isBig {
		f = f / math.Pow10(cnt)
	} else {
		f = f * math.Pow10(cnt)
	}
	if cnt == 0 {
		return fmt.Sprintf("%.2f", f)
	}
	if isBig {
		if cnt == 1 {
			return fmt.Sprintf("%.1f", origin)
		} else {
			// ->int->取模
			intNum := int(origin)
			intNum = intNum / int(math.Pow10(cnt-2))
			intNum = intNum * int(math.Pow10(cnt-2))
			return fmt.Sprintf("%v", intNum)
		}
	} else {
		//small 1 3, 2 4
		a := fmt.Sprintf("%v%vf", "%.", cnt+2)
		return fmt.Sprintf(a, origin)
	}

}

func recur(f float64, isBig bool, cnt int) int {
	if f >= 1 && f < 10 {
		return cnt
	}
	if isBig {
		f /= 10
	} else {
		f *= 10
	}
	cnt++
	return recur(f, isBig, cnt)
}
