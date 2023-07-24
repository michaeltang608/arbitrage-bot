package binan

import (
	"fmt"
	"github.com/valyala/fastjson"
	"strconv"
	"time"
)

// 获取margin u的余额
func (s *service) queryMarginBal() float64 {
	url := "/sapi/v1/margin/account"
	url = fmt.Sprintf("%s?timestamp=%s", url, strconv.FormatInt(time.Now().UnixMilli(), 10))
	respBytes := http("GET", url)
	v1, _ := fastjson.ParseBytes(respBytes)
	v2 := v1.Get("userAssets")
	ary1, _ := v2.Array()
	for _, v := range ary1 {
		if string(v.GetStringBytes("asset")) == "USDT" {
			freeAmtStr := v.GetStringBytes("free")
			freeFloat, _ := strconv.ParseFloat(string(freeAmtStr), 64)
			return freeFloat
		}
	}
	return 0.0
}
