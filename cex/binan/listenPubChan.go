package binan

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
	"strings"
	"ws-quant/common/symb"
)

func (s *service) ListenAndNotifyPublic() {
	s.connectAndSubscribePublic()
	s.listenAndNotifyPublic()

}

// 连接和订阅是在一起的
func (s *service) connectAndSubscribePublic() {
	baseUrl := "wss://stream.binance.com:9443"
	template := "%susdt@ticker/"
	tickers := ""
	for _, s := range symb.GetAllSymb() {
		ticker := fmt.Sprintf(template, strings.ToLower(s))
		tickers = tickers + ticker
	}
	tickers = tickers[:len(tickers)-1]
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s/stream?streams=%s", baseUrl, tickers), nil)
	if err != nil {
		log.Panic("ws connect err=", err)
	}
	s.pubCon = conn
}

func (s *service) listenAndNotifyPublic() {
	for {
		if s.pubCon == nil {
			log.Panic("pubCon == nil")
		}
		_, msgBytes, err := s.pubCon.ReadMessage()
		if err != nil {
			log.Panic("read msg err=", err)
		}
		instTicker := fastjson.GetString(msgBytes, "stream")
		if strings.HasSuffix(instTicker, "usdt@ticker") {
			symbolLower := instTicker[:len(instTicker)-len("usdt@ticker")]
			ask := fastjson.GetString(msgBytes, "data", "a")
			bid := fastjson.GetString(msgBytes, "data", "b")
			log.Info("symbol=%s, ask=%s, bid=%s\n", symbolLower, ask, bid)
		} else {
			log.Info("收到msg: %v\n", string(msgBytes))
		}
	}
}
