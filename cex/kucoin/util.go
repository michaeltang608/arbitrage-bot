package kucoin

import (
	"net/http"
	"strconv"
	"time"
	"ws-quant/pkg/util"
)

/*
封装好的公共的函数
*/

// 带签名的http请求
func authHttpRequest(apiUrl, method string, body string) []byte {
	baseUrl := "https://api.kucoin.com"
	url := apiUrl
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)

	signText := ts + method + url + body
	sign := util.Sha256AndBase64(signText, apiSecret)
	headers := map[string]string{
		"KC-API-KEY":         apiKey,
		"KC-API-SIGN":        sign,
		"KC-API-TIMESTAMP":   ts,
		"KC-API-PASSPHRASE":  util.Sha256AndBase64(apiPass, apiSecret),
		"KC-API-KEY-VERSION": "2",
	}
	if method == http.MethodPost {
		headers["Content-Type"] = "application/json"
	}
	return util.HttpRequest(method, baseUrl+url, body, headers)
}
