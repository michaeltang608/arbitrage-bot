package oke

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
	"strconv"
	"strings"
	"time"
	"ws-quant/common/symb"
	"ws-quant/models/bean"
	"ws-quant/pkg/feishu"
)

func (s *service) connectAndSubscribePublic() {

	s.connectPublic()
	s.subscribeTickers()
	s.subscribeFutures()
	//s.subscribeInstruments()

}

// 产品交易对，目前无需订阅
func (s *service) subscribeInstruments() {
	// subscribe trade products
	arg := make(map[string]interface{})
	arg["channel"] = "instruments"
	arg["instType"] = "MARGIN"
	req := &Req{
		Op: "subscribe",
		Args: []map[string]interface{}{
			arg,
		},
	}
	reqBytes, _ := json.Marshal(req)
	err := s.pubCon.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		feishu.Send("准备退出,发送OKEX订阅消息失败")
		log.Panic("发送OKEX订阅消息失败 ", err)
	}

}
func (s *service) subscribeFutures() {
	var err error
	argList := make([]map[string]interface{}, 0)
	for _, symbol_ := range symb.GetAllOkFuture() {

		arg := make(map[string]interface{})
		arg["channel"] = "tickers"
		arg["instId"] = fmt.Sprintf("%s-USDT-SWAP", strings.ToUpper(symbol_))
		argList = append(argList, arg)
	}

	req := &Req{
		Op:   "subscribe",
		Args: argList,
	}
	reqBytes, _ := json.Marshal(req)
	err = s.pubCon.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		log.Panic("发送OKEX订阅futures消息失败 ", err)
	}
	log.Info("订阅全部futures数据成功")
}
func (s *service) subscribeTickers() {
	var err error
	argList := make([]map[string]interface{}, 0)
	for _, symbol_ := range symb.GetAllSymb() {

		arg := make(map[string]interface{})
		arg["channel"] = "tickers"
		arg["instId"] = fmt.Sprintf("%s-USDT", strings.ToUpper(symbol_))
		argList = append(argList, arg)
	}

	req := &Req{
		Op:   "subscribe",
		Args: argList,
	}
	reqBytes, _ := json.Marshal(req)
	err = s.pubCon.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		log.Panic("发送OKEX订阅消息失败 ", err)
	}
	log.Info("订阅全部tickers数据成功")

}
func (s *service) connectPublic() {
	// 可能会重连
	s.pubConLock.Lock()
	defer s.pubConLock.Unlock()
	if s.pubConLastConnectTime >= (time.Now().Second() - 10) {
		//刚刷新不处理
		return
	}
	if s.pubCon != nil {
		_ = s.pubCon.Close()
	}
	var err error
	socketUrl := "wss://ws.okx.com:8443/ws/v5/public"
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		// 第二次尝试连接，提高胜率
		conn, _, err = websocket.DefaultDialer.Dial(socketUrl, nil)
		if err != nil {
			log.Panic("oke socket 连续两次连接失败", err.Error())
		}
	}
	s.pubCon = conn
	s.pubConLastConnectTime = time.Now().Second()
	log.Info("连接pubCon 成功，开始监听消息了")
}

func (s *service) listenAndNotifyPublic() {
	errCnt := 0
	for {
		if s.pubCon == nil {
			time.Sleep(time.Second)
			continue
		}
		_, msgBytes, err := s.pubCon.ReadMessage()
		if err != nil {
			log.Error("Error in receive:", err)
			time.Sleep(time.Second)
			errCnt++
			if errCnt > 10 {
				log.Info("读取失败累计超过10次，开始重启")
				log.Panic("oke read pub err")
			}
			continue
		}

		errCnt = 0
		/*
			接受到的数据有如下几种场景
			1 接受到 pong
			2 接受到event
				- 如果是login, 立刻订阅
		*/
		msg := string(msgBytes)
		if msg == "pong" {
			log.Info("获取pong数据")
			continue
		}

		//_, ok := resp["data"]
		if fastjson.GetString(msgBytes, "arg", "channel") == "tickers" {
			// 收到 ticker 数据
			// 2 获取价格
			bestAsk := fastjson.GetString(msgBytes, "data", "0", "askPx")
			price := fastjson.GetString(msgBytes, "data", "0", "last")
			bestBid := fastjson.GetString(msgBytes, "data", "0", "bidPx")

			instId := fastjson.GetString(msgBytes, "data", "0", "instId")
			symbolStr := strings.Split(instId, "-")[0]
			priceBeatAskFloat, _ := strconv.ParseFloat(bestAsk, 64)
			priceFloat, _ := strconv.ParseFloat(price, 64)
			priceBestBidFloat, _ := strconv.ParseFloat(bestBid, 64)

			for _, symbol_ := range symb.GetAllSymb() {
				if strings.ToUpper(symbol_) == strings.ToUpper(symbolStr) {
					tickerBean := bean.TickerBean{
						CexName:      s.GetCexName(),
						InstId:       instId,
						SymbolName:   symbol_,
						Price:        priceFloat,
						PriceBestBid: priceBestBidFloat,
						PriceBestAsk: priceBeatAskFloat,
						Ts0:          time.Now().UnixMilli(),
					}
					s.tickerChan <- tickerBean
				}
			}

		} else {
			if fastjson.GetString(msgBytes, "event") == "subscribe" {
				//ignore
			} else if fastjson.GetString(msgBytes, "event") == "error" {
				log.Info("public 接收到订阅失败事件：%v\n", string(msgBytes))
			} else {
				log.Info("public 接收到未知业务数据：%v\n", string(msgBytes))
			}
		}
	}
}
