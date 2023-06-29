package kucoin

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
	"ws-quant/pkg/e"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/util"
)

func (s *service) ConnectAndSubscribePublic() {
	s.connectPublic()
	if s.pubCon == nil {
		log.Error("pubCon为null, 很奇怪")
		feishu.Send("奇怪，ku error in pub con")
	}
	s.subscribeTickers()
	go func() {
		defer e.Recover()
		s.pingPublic()
	}()
}

func (s *service) connectPublic() {
	s.lastPong = 0

	s.pubConLock.Lock()
	defer s.pubConLock.Unlock()
	if s.pubConLastConnectTime >= (time.Now().Second() - 10) {
		//刚刷新不处理
		return
	}
	if s.pubCon != nil {
		log.Info("开始连接前，关闭旧的连接")
		_ = s.pubCon.Close()
	}
	resp := util.SendPost("https://api.kucoin.com/api/v1/bullet-public", nil)
	token := fastjson.GetString([]byte(resp), "data", "token")
	if token == "" {
		panic("ws连接token获取失败")
	}
	socketUrl := fmt.Sprintf("wss://ws-api-spot.kucoin.com?token=%s", token)
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		panic("连接异常" + err.Error())
	}
	s.pubCon = conn
	s.pubConLastConnectTime = time.Now().Second()
}
func (s *service) subscribeTickers() {
	topic := "/market/ticker:"
	for _, symbol := range symb.GetAllSymb() {
		symbol = strings.ToUpper(symbol) + "-USDT,"
		topic = topic + symbol
	}
	topic = topic[:len(topic)-1]
	subscribe(topic, s.pubCon, false)
}

func (s *service) ListenAndNotifyPublic() {
	go s.processMsg()
	readErrCnt := 0
	for {
		_, msg, err := s.pubCon.ReadMessage()
		if err != nil {
			readErrCnt++
			s.ConnectAndSubscribePublic()
			if readErrCnt > 100 {
				// 三次都读取异常才放弃，准备退出
				log.Panic("oke累计100次读错误，Error in receive:", err)
			}
			time.Sleep(time.Second)
			continue
		}
		readErrCnt = 0
		//log.Printf("客户端收到信息:%v\n", string(msg))
		d1 := make(map[string]interface{})
		_ = json.Unmarshal(msg, &d1)
		_, ok := d1["data"]
		if !ok {
			if fastjson.GetString(msg, "type") == "pong" {
				lastPongStr := fastjson.GetString(msg, "id")
				lastPongStr = lastPongStr[:len(lastPongStr)-3]
				parseInt, _ := strconv.ParseInt(lastPongStr, 10, 64)
				s.lastPong = parseInt
			} else {
				log.Info("kucoin收到非业务数据:%v", string(msg))
			}
		} else {
			//todo 考虑减少数据推送频率
			select {
			case MsgChan <- msg:
			default:
			}
		}
	}
}

// ExtractMsg ,处理业务数据， 每隔500ms处理一次
func (s *service) processMsg() {
	//ticker := time.NewTicker(time.Millisecond * 50)
	for {
		select {
		case msg := <-MsgChan:
			//log.Info("收到ticker 业务数据, 准备解析回传%v\n", string(msg))

			priceBestAsk := fastjson.GetString(msg, "data", "bestAsk")
			priceStr := fastjson.GetString(msg, "data", "price")
			priceBestBid := fastjson.GetString(msg, "data", "bestBid")

			priceBestAskFloat, _ := strconv.ParseFloat(priceBestAsk, 64)
			priceFloat, _ := strconv.ParseFloat(priceStr, 64)
			priceBestBidFloat, _ := strconv.ParseFloat(priceBestBid, 64)

			topic := fastjson.GetString(msg, "topic")

			topic = strings.Split(topic, ":")[1]
			topic = strings.Split(topic, "-")[0]
			topic = strings.ToLower(topic)
			//log.Info("%v price is %v\n", topic, priceFloat)
			for _, symbol_ := range symb.GetAllSymb() {
				if strings.ToUpper(symbol_) == strings.ToUpper(topic) {
					tickerBean := bean.TickerBean{
						CexName:      s.GetCexName(),
						SymbolName:   symbol_,
						PriceBestAsk: priceBestAskFloat,
						Price:        priceFloat,
						PriceBestBid: priceBestBidFloat,
						Ts0:          time.Now().UnixMilli(),
					}
					//log.Info("回传数据给server")
					s.tickerChan <- tickerBean
				}
			}
		}
	}
}

func (s *service) pingPublic() {
	// 每隔15s ping一次
	ticker := time.NewTicker(time.Second * 15)
	for range ticker.C {
		pingMsg := Ping{
			Id:   strconv.FormatInt(time.Now().UnixMilli(), 10),
			Type: "ping",
		}
		msgBytes, _ := json.Marshal(pingMsg)
		err := s.pubCon.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			log.Error("发送ping失败: %v\n", err)
			//结束本次 ping goroutine, 会有监测机制重启新的ping的 goroutine
			return
		} else {
			//log.Info("发送ping数据成功,数据是%s", string(msgBytes))
		}
	}
}
