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
	"ws-quant/core"
	"ws-quant/pkg/mapper"
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
			s.processNewOrder(msgBytes)
			continue
		}

		// 5.2 收到订单状态更新数据,
		// todo 这里有个问题，这里的数据肯能比上面改的 new_order 早点到达，所以计划同步插入数据，异步以此处收到数据做修改，逐渐淘汰上面收到的数据
		if fastjson.GetString(msgBytes, "arg", "channel") == "orders" {
			s.processUpdateOrder(msgBytes)
			continue
		}
		//other
		log.Info("OKEX接收其他未知业务数据：%v\n", msg)
	}
}

func (s *Service) processUpdateOrder(msgBytes []byte) {
	log.Info("订单状态更新: %v\n", string(msgBytes))
	val, _ := fastjson.ParseBytes(msgBytes)
	val = val.Get("data", "0")
	orderId := string(val.GetStringBytes("ordId"))
	state := string(val.GetStringBytes("state"))
	//price := string(val.GetStringBytes("avgPx"))

	side := fastjson.GetString(msgBytes, "data", "0", "side")
	instId := fastjson.GetString(msgBytes, "data", "0", "instId")
	size := fastjson.GetString(msgBytes, "data", "0", "accFillSz")
	prc := fastjson.GetString(msgBytes, "data", "0", "avgPx")

	// 先查询数据库订单
	condBean := &models.Orders{InstId: instId, Closed: "N"}
	orderList := make([]*models.Orders, 0)
	err := mapper.FindLast(s.db, &orderList, condBean)
	if err != nil || orderList[0] == nil {
		log.Error("未知订单信息，数据库未查到: %v\n", orderId)
		return
	}
	orderDb := orderList[0]
	log.Info("找到数据库数据开始更新订单状态, orderId=%v\n", orderId)
	isCanceled := state == core.CANCELED.State()
	closed := "N"
	if isCanceled {
		closed = "Y"
	}
	// 发送 signal 给上级
	if state == core.FILLED.State() {
		s.uploadOrder(orderDb.PosSide, side)
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
			log.Info("close掉开仓订单")
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
func (s *Service) processNewOrder(msgBytes []byte) {
	log.Info("新订单数据:" + string(msgBytes))
	return
	//if fastjson.GetString(msgBytes, "code") == "1" {
	//	errMsg := fastjson.GetString(msgBytes, "data", "0", "sMsg")
	//	feishu.Send("oke new order fail: " + errMsg)
	//	return
	//}
	//posSide := "open"
	//if s.openOrder != nil {
	//	posSide = "close"
	//}
	//
	//orderType := fastjson.GetString(msgBytes, "data", "0", "ordType")
	//price := fastjson.GetString(msgBytes, "data", "0", "avgPx")
	//
	//orderInsert := &models.Orders{
	//	Cex:       cex.OKE,
	//	Price:     price,
	//	OrderType: orderType,
	//	PosSide:   posSide,
	//	State:     core.TRIGGER.State(),
	//	OrderId:   fastjson.GetString(msgBytes, "data", "0", "ordId"),
	//	Closed:    "N",
	//	Created:   time.Now(),
	//	Updated:   time.Now(),
	//}
	//_ = mapper.Insert(s.db, orderInsert)
	//s.ReloadOrders()
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
