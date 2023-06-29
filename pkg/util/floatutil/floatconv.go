package floatutil

import "strconv"

func Format(f float64) string {
	return strconv.FormatFloat(f, 'f', 4, 64)
}
