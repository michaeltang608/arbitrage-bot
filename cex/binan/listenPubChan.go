package binan

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
	"strconv"
	"strings"
	"time"
	"ws-quant/common/symb"
	"ws-quant/models/bean"
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
	errCnt := 0
	for {
		if s.pubCon == nil {
			log.Panic("pubCon == nil")
		}
		_, msgBytes, err := s.pubCon.ReadMessage()
		if err != nil {
			errCnt += 1
			log.Error("read msg err=", err)
			time.Sleep(time.Second)
			if errCnt > 10 {
				log.Panic("累计多次读取失败，退出")
			}
		}
		errCnt = 0
		instTicker := fastjson.GetString(msgBytes, "stream")
		if strings.HasSuffix(instTicker, "usdt@ticker") {
			instId := instTicker[:len(instTicker)-len("@ticker")]
			symbolLower := instTicker[:len(instTicker)-len("usdt@ticker")]
			ask := fastjson.GetString(msgBytes, "data", "a")
			bid := fastjson.GetString(msgBytes, "data", "b")
			cur := fastjson.GetString(msgBytes, "data", "c")

			askFloat, _ := strconv.ParseFloat(ask, 64)
			bidFloat, _ := strconv.ParseFloat(bid, 64)
			curFloat, _ := strconv.ParseFloat(cur, 64)

			if symb.SymbolExist(symbolLower) {
				tickerBean := bean.TickerBean{
					CexName:      s.GetCexName(),
					InstId:       instId,
					SymbolName:   strings.ToUpper(symbolLower),
					Price:        curFloat,
					PriceBestBid: bidFloat,
					PriceBestAsk: askFloat,
					Ts0:          time.Now().UnixMilli(),
				}
				s.tickerChan <- tickerBean
			}
		} else {
			log.Info("binan收到其他未知msg: %v\n", string(msgBytes))
		}
	}
}
