package service

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
	"strconv"
	"time"
	"ws-quant/cex"
	"ws-quant/common/bean"
	"ws-quant/common/symb"
)

type ArgBit struct {
	InstType string `json:"instType"`
	Channel  string `json:"channel"`
	InstId   string `json:"instId"`
}
type PubReqBit struct {
	Op   string   `json:"op"`
	Args []ArgBit `json:"args"`
}

func (s *Service) connectAndSubscribePublicBit() {

	s.connectPubBit()
	s.subscribeFutureBit()

}

/*
*
symbol: "PENDLEUSDT_UMCBL",
symbolName: "PENDLEUSDT",
*/
func (s *Service) subscribeFutureBit() {
	var err error
	argList := make([]ArgBit, 0)
	for _, symbol_ := range symb.GetMergeFutureList() {

		arg := ArgBit{
			InstType: "MC",
			Channel:  "ticker",
			InstId:   symbol_ + "USDT",
		}
		argList = append(argList, arg)
	}

	req := &PubReqBit{
		Op:   "subscribe",
		Args: argList,
	}
	reqBytes, _ := json.Marshal(req)
	err = s.pubConBit.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		log.Panic("发送OKEX订阅消息失败 ", err)
	}
	log.Info("订阅全部tickers数据成功")

}
func (s *Service) connectPubBit() {
	// 可能会重连
	log.Info("开始连接pub con")
	var err error
	socketUrl := "wss://ws.bitget.com/mix/v1/stream"
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil || conn == nil {
		// 第二次尝试连接，提高胜率
		conn, _, err = websocket.DefaultDialer.Dial(socketUrl, nil)
		if err != nil || conn == nil {
			log.Panic("service socket 连续两次连接失败", err.Error())
		}
	}
	if conn == nil {
		log.Info("奇怪，conn 还是 null")
	}
	s.pubConBit = conn
	s.pubConLastConnectTime = time.Now().Second()
	log.Info("连接pubCon bit 成功，开始监听消息了, pubCon==nil, %v", s.pubConBit == nil)
}

func (s *Service) listenAndNotifyPubBit() {
	errCnt := 0
	for {
		if s.pubConBit == nil {
			time.Sleep(time.Second)
			continue
		}
		_, msgBytes, err := s.pubConBit.ReadMessage()
		if err != nil {
			log.Error("Error in receive:", err)
			time.Sleep(time.Second)
			errCnt++
			if errCnt > 10 {
				log.Info("pubConBit 读取失败累计超过10次，开始重启")
				log.Panic("service read pub err")
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

		if fastjson.GetString(msgBytes, "event") == "subscribe" {
			//ignore
			log.Info("收到 subscribe event")
			continue
		}

		//_, ok := resp["data"]
		if fastjson.GetString(msgBytes, "action") == "snapshot" {
			// 收到 ticker 数据
			// 2 获取价格
			bestAsk := fastjson.GetString(msgBytes, "data", "0", "bestAsk")
			price := fastjson.GetString(msgBytes, "data", "0", "last")
			bestBid := fastjson.GetString(msgBytes, "data", "0", "bestBid")
			instId := fastjson.GetString(msgBytes, "data", "0", "instId")

			symbolStr := instId[:len(instId)-4]
			priceBeatAskFloat, _ := strconv.ParseFloat(bestAsk, 64)
			priceFloat, _ := strconv.ParseFloat(price, 64)
			priceBestBidFloat, _ := strconv.ParseFloat(bestBid, 64)

			// todo 这里可优化
			for _, symbol_ := range symb.GetMergeFutureList() {
				if symbol_ == symbolStr {
					tickerBean := bean.TickerBean{
						CexName:      cex.BIT,
						InstId:       instId,
						SymbolName:   symbolStr,
						Price:        priceFloat,
						PriceBestBid: priceBestBidFloat,
						PriceBestAsk: priceBeatAskFloat,
						Ts0:          time.Now().UnixMilli(),
					}
					s.tickerChan <- tickerBean
				}
			}
			continue
		}
		log.Info("收到未知类型消息:%s\n", msg)
	}
}
