package kucoin

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
	"net/http"
	"strconv"
	"time"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/common/consts"
	"ws-quant/core"
	"ws-quant/models/bean"
	"ws-quant/pkg/e"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
)

func (s *service) ConnectAndSubscribePrivate() {
	s.connectPrivate()
	log.Info("准备订阅消息")
	subscribe("/spotMarket/tradeOrders", s.prvCon, true)
	go func() {
		defer e.Recover()()
		s.pingPrivate()
	}()
}

func (s *service) ListenAndNotifyPrivate() {
	errCnt := 0
	log.Info("准备监听prv消息")
	for {
		_, msgBytes, err := s.prvCon.ReadMessage()

		// 0 读取诗句失败
		if err != nil {
			log.Error("Error in receive:", err)
			s.ConnectAndSubscribe()
			errCnt++
			if errCnt > 500 {
				log.Panic("oke 累计读取失败超过10次，准备退出")
			}
			continue
		}
		msg := string(msgBytes)

		if fastjson.GetString(msgBytes, "type") == "pong" {
			//ignore pong
			continue
		}

		if fastjson.GetString(msgBytes, "topic") == "/spotMarket/tradeOrders" {
			//receive order change
			orderId := fastjson.GetString(msgBytes, "data", "orderId")
			if orderId == "" {
				log.Error("收到订单推送，但无订单号")
				feishu.Send("收到订单推送，但无订单号")
				continue
			}

			state := fastjson.GetString(msgBytes, "data", "type")
			orderDb := &models.Orders{OrderId: orderId}
			has := mapper.Get(s.db, orderDb)
			if state == "open" || (!has && state == "match") {
				//新订单
				log.Info("新订单数据:" + msg)
				posSide := "open"
				if s.openOrder != nil {
					posSide = "close"
				}

				orderType := fastjson.GetString(msgBytes, "data", "orderType")
				price := fastjson.GetString(msgBytes, "data", "price")
				if orderType == "market" {
					price = fastjson.GetString(msgBytes, "data", "matchPrice")
				}
				orderInsert := &models.Orders{
					InstId:    fastjson.GetString(msgBytes, "data", "symbol"),
					Cex:       s.GetCexName(),
					Price:     price,
					Size:      fastjson.GetString(msgBytes, "data", "size"),
					Side:      fastjson.GetString(msgBytes, "data", "side"),
					OrderType: orderType,
					PosSide:   posSide,
					State:     core.TRIGGER.State(),
					OrderId:   orderId,
					Closed:    "N",
					Created:   time.Now(),
					Updated:   time.Now(),
				}
				_ = mapper.Insert(s.db, orderInsert)
				s.ReloadOrders()
				continue
			}
			// 订单更新
			log.Info("订单更新数据:" + msg)
			if !has {
				log.Info("开始更新订单状态, orderId=%v\n", orderId)
				continue
			}
			log.Info("开始更新订单状态, orderId=%v\n", orderId)
			closed := "N"
			isFilled := state == core.FILLED.State()
			if isFilled {
				if orderDb.PosSide == "open" {
					s.execStateChan <- bean.ExecState{
						PosSide:   consts.Open,
						CexName:   cex.KUCOIN,
						Side:      orderDb.Side,
						OrderType: orderDb.OrderType,
					}
				}
				if orderDb.PosSide == "close" {
					s.execStateChan <- bean.ExecState{
						PosSide:   consts.Close,
						CexName:   cex.KUCOIN,
						Side:      orderDb.Side,
						OrderType: orderDb.OrderType,
					}
				}
			}

			if state == core.CANCELED.State() {
				closed = "Y"
			}
			// 如果是平仓且生效，则该次策略完成
			if orderDb.PosSide == "close" && state == core.FILLED.State() {
				log.Info("该次策略完成")
				closed = "Y"
				// 同时也 close 开仓
				if s.openOrder == nil {
					log.Error("找不到开仓订单")
				} else {
					updateOpen := &models.Orders{Closed: "Y", Updated: time.Now()}
					_ = mapper.UpdateById(s.db, s.openOrder.ID, updateOpen)
				}
			}
			updateModel := &models.Orders{
				State:   state,
				Closed:  closed,
				Updated: time.Now(),
			}
			if isFilled {
				updateModel.FilledTime = time.Now()
			}
			_ = mapper.UpdateById(s.db, orderDb.ID, updateModel)

			s.ReloadOrders()
			continue
		}
		log.Info("prv收到未知业务数据: " + msg)
	}
}

func (s *service) connectPrivate() {
	s.lastPong = 0
	if s.prvCon != nil {
		log.Info("开始连接前，关闭旧的private连接")
		_ = s.prvCon.Close()
	}
	resp := authHttpRequest("/api/v1/bullet-private", http.MethodPost, "")
	token := fastjson.GetString(resp, "data", "token")
	if token == "" {
		panic("ws连接token获取失败")
	}
	socketUrl := fmt.Sprintf("wss://ws-api-spot.kucoin.com?token=%s", token)
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		panic("连接异常" + err.Error())
	}
	log.Info("连接prvChannel成功")
	s.prvCon = conn

}
func (s *service) pingPrivate() {
	// 每隔15s ping一次
	ticker := time.NewTicker(time.Second * 15)
	for range ticker.C {
		pingMsg := Ping{
			Id:   strconv.FormatInt(time.Now().UnixMilli(), 10),
			Type: "ping",
		}
		msgBytes, _ := json.Marshal(pingMsg)
		err := s.prvCon.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			log.Error("发送ping失败: %v\n", err)
			//结束本次 ping goroutine, 会有监测机制重启新的ping的 goroutine
			return
		} else {
			//log.Info("发送ping数据成功,数据是%s", string(msgBytes))
		}
	}
}
