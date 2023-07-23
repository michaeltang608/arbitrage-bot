package binan

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"github.com/valyala/fastjson"
	"strconv"
	"strings"
	"time"
	"ws-quant/pkg/util"
)

func CreateListenKey() string {
	url := "/sapi/v1/userDataStream"
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	url = fmt.Sprintf("%s?timestamp=%s", url, ts)
	respBytes := http("POST", url)
	log.Info("创建listenKey的resp=%v", string(respBytes))
	return fastjson.GetString(respBytes, "listenKey")
}

// 封装 binance 的http请求，添加请求头 apiKey, 并组装 全路径url
/**
- 添加请求头 apiKey
- 组装 全路径url
- 添加签名
*/
func http(method, url string) []byte {
	ary := strings.Split(url, "?")
	if len(ary) >= 2 {
		url = fmt.Sprintf("%s&signature=%s", url, sign(ary[1]))
	}
	fullUrl := fmt.Sprintf("%s%s", baseUrl, url)
	return util.HttpRequest(method, fullUrl, "", map[string]string{"X-MBX-APIKEY": apiKey})
}

func sign(text string) string {
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(text))
	return fmt.Sprintf("%x", h.Sum(nil))
}
