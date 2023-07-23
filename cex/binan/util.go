package binan

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"github.com/valyala/fastjson"
	"strconv"
	"time"
	"ws-quant/pkg/util"
)

func CreateListenKey() string {
	url := "https://api.binance.com/sapi/v1/userDataStream"
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	url = fmt.Sprintf("%s?timestamp=%s&signature=%s", url, ts, Sha256("signature="+ts, apiSecret))
	respBytes := util.HttpRequest("POST", url, "", map[string]string{"X-MBX-APIKEY": apiKey})
	resp := string(respBytes)
	log.Info("创建listenKey的resp=%v", resp)
	return fastjson.GetString(respBytes, "listenKey")
}

func Sha256(text, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(text))
	return fmt.Sprintf("%x", h.Sum(nil))
}
