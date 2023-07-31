package numutil

import (
	"fmt"
	"strconv"
)

func Format(f float64) string {
	return fmt.Sprintf("%f", f)
}

func FormatInt(f int) string {
	return fmt.Sprintf("%d", f)
}
func FormatByPrecision(f float64, precision int) string {
	return strconv.FormatFloat(f, 'f', precision, 64)
}

func Parse(s string) float64 {
	resultF, _ := strconv.ParseFloat(s, 64)
	return resultF
}
