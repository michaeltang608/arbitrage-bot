package kucoin

import (
	"github.com/valyala/fastjson"
	"net/http"
	"strconv"
	"strings"
)

func (s *service) MarginBalance() float64 {

	url := "/api/v1/margin/account"
	val := fastjson.MustParse(string(authHttpRequest(url, http.MethodGet, "")))
	valAccs := val.Get("data", "accounts")
	accArr, err := valAccs.Array()
	if err != nil {
		log.Panic("accounts 转 ary 失败", err)
	}
	for _, acc := range accArr {
		currency := acc.GetStringBytes("currency")
		if strings.ToUpper(string(currency)) == "USDT" {
			totalBal := acc.GetStringBytes("totalBalance")
			totalBalFloat, _ := strconv.ParseFloat(string(totalBal), 64)
			return totalBalFloat
		}
	}
	return 0
}
