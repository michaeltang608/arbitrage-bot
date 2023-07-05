package oke

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"strconv"
	"strings"
	"time"
	"ws-quant/cex"
	"ws-quant/core"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/util"
)

func (s *service) startPing() {
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

func (s *service) OpenPosLimit(symbol, price, size, side string) (msg string) {
	if s.openOrder != nil {
		errMsg := fmt.Sprintf("已开仓中，勿再重复开仓:%+v", *s.openOrder)
		feishu.Send("ok已开仓，勿重复开")
		return errMsg
	}
	return s.TradeLimit(symbol, price, size, side, "open")
}

func (s *service) OpenPosMarket(symbol, size, side string) (msg string) {
	if s.openOrder != nil {
		errMsg := fmt.Sprintf("已开仓中，勿再重复开仓:%+v", *s.openOrder)
		feishu.Send("ok已开仓，勿重复开")
		return errMsg
	}
	return s.TradeMarket(symbol, size, side, "open")
}
func (s *service) ClosePosMarket(askPrc float64, bidPrc float64) (msg string) {
	if s.openOrder == nil || s.openOrder.State != string(core.FILLED) {
		feishu.Send("ok receive signal close, but found no open")
		return "no position to close"
	}
	if s.closeOrder != nil {
		return "close order already placed"
	}
	side := cex.Buy
	if s.openOrder.Side == cex.Buy {
		side = cex.Sell
	}
	sizeFloat, _ := strconv.ParseFloat(s.openOrder.Size, 64)
	if side == cex.Buy {
		sizeFloat = sizeFloat * askPrc
	}

	// if buy in market mode, size refer to the amount of U
	size := util.AdjustClosePosSize(sizeFloat, side, cex.OKE)
	symbol := strings.Split(s.openOrder.InstId, "-")[0]
	return s.TradeMarket(symbol, size, side, cex.Close)
}

func (s *service) ClosePosLimit(price string) (msg string) {
	// 为开仓或者 开的仓位未成交
	if s.openOrder == nil || s.openOrder.State != string(core.FILLED) {
		feishu.Send("ok receive signal close, but found no open")
		return "无仓位需要平仓"
	}
	if s.closeOrder != nil {
		return "无需重复平仓"
	}
	side := cex.Buy
	if s.openOrder.Side == cex.Buy {
		side = cex.Sell
	}
	sizeFloat, _ := strconv.ParseFloat(s.openOrder.Size, 64)
	size := util.AdjustClosePosSize(sizeFloat, side, cex.OKE)
	symbol := strings.Split(s.openOrder.InstId, "-")[0]
	return s.TradeLimit(symbol, price, size, side, cex.Close)
}

func (s *service) TradeMarket(symbol, size, side, posSide string) (msg string) {
	closePos := posSide == cex.Close
	instId := fmt.Sprintf("%s-USDT", strings.ToUpper(symbol))
	arg := map[string]interface{}{
		"side":       strings.ToLower(side),
		"instId":     instId,
		"tdMode":     "cross",
		"ordType":    "market",
		"sz":         size,
		"ccy":        "USDT",
		"reduceOnly": closePos,
	}

	req := Req{
		Id: "001",
		Op: "order",
		Args: []map[string]interface{}{
			arg,
		},
	}
	reqBytes, _ := json.Marshal(req)
	log.Info("准备下单信息:%v\n", string(reqBytes))
	err := s.prvCon.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		panic("发送订阅账户余额数据失败")
	} else {
		return "trigger trade成功, 最终结果见推送数据"
	}
}
func (s *service) TradeLimit(symbol, price, size, side, posSide string) (msg string) {
	closePos := posSide == cex.Close
	instId := fmt.Sprintf("%s-USDT", strings.ToUpper(symbol))
	arg := map[string]interface{}{
		"side":       strings.ToLower(side),
		"instId":     instId,
		"sz":         size,
		"px":         price,
		"tdMode":     "cross",
		"ccy":        "USDT",
		"ordType":    "limit", //market, limit, ioc
		"reduceOnly": closePos,
	}

	req := Req{
		Id: "001",
		Op: "order",
		Args: []map[string]interface{}{
			arg,
		},
	}
	reqBytes, _ := json.Marshal(req)
	log.Info("准备下单信息:%v\n", string(reqBytes))

	//order := &models.Orders{
	//	InstId:  instId,
	//	Cex:     s.GetCexName(),
	//	Price:   price,
	//	Size:    size,
	//	Side:    side,
	//	PosSide: posSide,
	//	State:   string(core.TRIGGER),
	//	OrderId: "",
	//	Closed:  "N",
	//	Created: time.Now(),
	//	Updated: time.Now(),
	//}
	//_ = mapper.Insert(s.db, order)

	//s.ReloadOrders()
	err := s.prvCon.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		panic("发送订阅账户余额数据失败")
	} else {
		return "trigger trade成功, 最终结果见推送数据"
	}
}

func (s *service) login() {
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
