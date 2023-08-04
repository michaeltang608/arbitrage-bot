package oke

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
	"strconv"
	"strings"
	"time"
	"ws-quant/cex/models"
	"ws-quant/common/consts"
	"ws-quant/core"
	"ws-quant/models/bean"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
	"ws-quant/pkg/util"
)

func (s *Service) connectAndLoginPrivate() {
	socketUrl := "wss://ws.okx.com:8443/ws/v5/private"
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Panic("oke socket 连续两次连接失败", err.Error())
	}
	s.prvCon = conn
	s.login()
}

func (s *Service) login() {
	// login
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timestamp + "GET" + "/users/self/verify"
	sign := util.Sha256AndBase64(message, apiSecret)

	loginArg := make(map[string]interface{})
	loginArg["apiKey"] = apiKey
	loginArg["passphrase"] = pwd
	loginArg["timestamp"] = timestamp
	loginArg["sign"] = sign
	loginReq := Req{
		Op: "login",
		Args: []map[string]interface{}{
			loginArg,
		},
	}
	req, _ := json.Marshal(loginReq)
	err := s.prvCon.WriteMessage(websocket.TextMessage, req)
	if err != nil {
		panic("发送login数据失败")
	} else {
		log.Info("发送login数据成功")
	}

}
func (s *Service) startPing() {
	go func() {
		ticker := time.NewTicker(time.Second * 15)
		for range ticker.C {
			err := s.prvCon.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				log.Error("发送ping失败")
			}
		}
	}()
}

/*
*
监听 login事件，触发 订阅余额
监听 余额变动事件更新usdt余额
*/
func (s *Service) listenAndNotifyPrivate() {
	errCnt := 0
	for {
		if s.prvCon == nil {
			time.Sleep(time.Second)
			continue
		}
		_, msgBytes, err := s.prvCon.ReadMessage()
		// 0 读取诗句失败
		if err != nil {
			log.Error("Error in receive:", err)
			time.Sleep(time.Second)
			s.ConnectAndSubscribe()
			errCnt++
			if errCnt > 10 {
				log.Info("读取失败累计超过10次，开始重连")
				time.Sleep(time.Second * 2)
				s.connectAndLoginPrivate()
				time.Sleep(time.Second)
			}
			continue
		}
		// 1 接受到 pong 数据
		msg := string(msgBytes)
		if string(msgBytes) == "pong" {
			//log.Info("获取pong数据")
			continue
		}
		resp := make(map[string]interface{})
		err = json.Unmarshal(msgBytes, &resp)
		if err != nil {
			log.Panic("反序列化数据失败 ", err)
		}

		// 2 收到event 数据
		event := fastjson.GetString(msgBytes, "event")
		if event != "" {
			switch event {
			case "login":
				log.Info("是login数据，准备发送subscribe")
				s.subscribeBalanceAndPos()
				s.subscribeOrder()
				s.subscribePosition()
				s.startPing()
			case "subscribe":
				log.Info("收到subscribe数据: %v\n", msg)
			default:
				log.Info("收到其他event数据: %v\n", msg)
			}
			continue
		}
		// 3 收到balance数据
		if fastjson.GetString(msgBytes, "arg", "channel") == "balance_and_position" {
			// 解析 余额map
			s.processBalance(msgBytes)
			continue
		}

		//4 收到账户数据，如需还款的总额度，用于平仓操作
		if fastjson.GetString(msgBytes, "arg", "channel") == "positions" {
			//log.Info("收到持仓更新数据, %v\n", msg)
			val, _ := fastjson.ParseBytes(msgBytes)
			val = val.Get("data")
			valAry, _ := val.Array()
			for _, v := range valAry {
				ccy := string(v.GetStringBytes("liabCcy"))
				num := string(v.GetStringBytes("liab"))
				debtMsg := fmt.Sprintf("oke还欠%v的%v待还", num, ccy)
				// 只需要打印一次即可
				if _, ok := s.debtMsgMap[debtMsg]; ok {
					//ignore
				} else {
					s.debtMsgMap[debtMsg] = "y"
					log.Info(debtMsg)
				}
			}
			continue
		}

		// 5.1 收到order 新建数据
		if fastjson.GetString(msgBytes, "op") == "order" {
			//process new order: ignore
			continue
		}

		// 5.2 收到订单状态更新数据,
		if fastjson.GetString(msgBytes, "arg", "channel") == "orders" {
			s.processUpdateOrder(msgBytes)
			continue
		}
		//other
		log.Info("OKEX接收其他未知业务数据：%v\n", msg)
	}
}

func (s *Service) processUpdateOrder(msgBytes []byte) {

	val, _ := fastjson.ParseBytes(msgBytes)
	val = val.Get("data", "0")
	orderId := string(val.GetStringBytes("ordId"))
	state := string(val.GetStringBytes("state"))
	//price := string(val.GetStringBytes("avgPx"))

	side := fastjson.GetString(msgBytes, "data", "0", "side")
	instId := fastjson.GetString(msgBytes, "data", "0", "instId")
	size := fastjson.GetString(msgBytes, "data", "0", "accFillSz")
	prc := fastjson.GetString(msgBytes, "data", "0", "avgPx")
	log.Info("订单状态更新: instId=%s,state=%s, %v\n", instId, state, string(msgBytes))

	// 先查询数据库订单
	condBean := &models.Orders{InstId: instId, Closed: "N"}
	orderList := make([]*models.Orders, 0)
	err := mapper.FindLast(s.db, &orderList, condBean)
	if err != nil || orderList[0] == nil {
		log.Error("未知订单信息，数据库未查到: %v\n", orderId)
		return
	}
	orderDb := orderList[0]
	orderType := orderDb.OrderType
	log.Info("找到数据库数据开始更新订单状态, orderId=%v\n", orderId)
	isCanceled := state == core.CANCELED.State()
	closed := "N"
	if isCanceled {
		closed = "Y"
	}
	// 发送 signal 给上级
	if state == consts.Filled {
		s.uploadOrder(orderDb.PosSide, side, orderDb.OrderType)
		// 如果是平仓且生效，则该次策略完成
		if orderDb.PosSide == "close" {
			log.Info("该次策略完成")
			closed = "Y"
			// 同时也 close 开仓
			openOrder := s.GetOpenOrder(orderType)
			if openOrder == nil {
				msg := fmt.Sprintf("找不到开仓订单, orderStat=%v", s.GetOrderStat())
				log.Error(msg)
				feishu.Send(msg)
			} else {
				updateOpen := &models.Orders{Closed: "Y", Updated: time.Now()}
				_ = mapper.UpdateById(s.db, openOrder.ID, updateOpen)
				log.Info("close掉开仓订单")
			}
		}
	}

	updateModel := &models.Orders{
		InstId:  instId,
		Side:    side,
		State:   state,
		Closed:  closed,
		OrderId: orderId,
		Updated: time.Now(),
	}

	if size != "0" && size != "" {
		updateModel.Size = size
	}
	if prc != "0" && prc != "" {
		updateModel.Price = prc
	}
	isFilled := state == core.FILLED.State()
	if isFilled {
		updateModel.FilledTime = time.Now()
	}
	_ = mapper.UpdateById(s.db, orderDb.ID, updateModel)
	s.ReloadOrders()

}

//func (s *service) processNewOrder(msgBytes []byte) {
//	log.Info("新订单数据：%v", string(msgBytes))
//	if fastjson.GetString(msgBytes, "code") == "1" {
//		feishu.Send("ok_order fail:" + fastjson.GetString(msgBytes, "data", "0", "sMsg"))
//	} else if fastjson.GetString(msgBytes, "code") == "0" {
//		orderId := fastjson.GetString(msgBytes, "data", "0", "ordId")
//		var id uint32
//		if s.openOrder != nil && s.openOrder.OrderId == "" {
//			s.openOrder.OrderId = orderId
//			s.openOrder.PosSide = string(core.PLACED)
//			id = s.openOrder.ID
//
//		} else if s.closeOrder != nil && s.closeOrder.OrderId == "" {
//			s.closeOrder.OrderId = orderId
//			s.closeOrder.PosSide = string(core.PLACED)
//			id = s.closeOrder.ID
//
//		} else {
//			log.Info("未知新订单数据")
//		}
//		updateEle := &models.Orders{
//			PosSide:   string(core.PLACED),
//			OrderId: orderId,
//			Updated: time.Now(),
//		}
//		_ = mapper.UpdateById(s.db, id, updateEle)
//	}
//}

func (s *Service) processBalance(msgBytes []byte) {
	val, _ := fastjson.ParseBytes(msgBytes)
	data := val.Get("data")
	dataAry, _ := data.Array()
	dataFirst := dataAry[0]

	//log.Info("余额变动的事件类型是: %v\n", string(dataFirst.GetStringBytes("eventType")))
	log.Info("收到余额数据：%v", string(msgBytes))
	balData := dataFirst.Get("balData")
	balDataAry, _ := balData.Array()
	for _, balDataEle := range balDataAry {
		cashBal := balDataEle.GetStringBytes("cashBal")
		cashBalFloat, _ := strconv.ParseFloat(string(cashBal), 64)
		ccy := string(balDataEle.GetStringBytes("ccy"))
		if cashBalFloat >= 0.001 {
			log.Info("余额更新ccy=%v, cashBal: %v\n", ccy, cashBalFloat)
		}
		if strings.ToLower(ccy) == "usdt" {
			s.usdtBal = cashBalFloat
			log.Info("oke的usdt最新余额是%v\n", cashBalFloat)
		}
	}
}

func (s *Service) uploadOrder(posSide, side, orderType string) {
	s.execStateChan <- bean.ExecState{
		PosSide:   posSide,
		OrderType: orderType,
		Side:      side,
	}
}
