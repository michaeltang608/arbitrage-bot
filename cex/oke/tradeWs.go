package oke

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"strings"
	"time"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/common/consts"
	"ws-quant/common/symb"
	"ws-quant/core"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
	"ws-quant/pkg/util"
)

// 本文件主要用户 open pos

func (s *Service) OpenMarginLimit(symbol, price, size, side string) (msg string) {
	if s.openMarginOrder != nil {
		errMsg := fmt.Sprintf("已开仓中，勿再重复开仓:%+v", *s.openMarginOrder)
		feishu.Send("ok margin already opened，勿重复开")
		return errMsg
	}
	instId := fmt.Sprintf("%s-USDT", symbol)
	return s.TradeLimit(instId, price, size, side, "open")
}

func (s *Service) OpenFutureLimit(symbol, price, size, side string) (msg string) {
	if s.openFutureOrder != nil {
		errMsg := fmt.Sprintf("已开仓中，勿再重复开仓:%+v", *s.openFutureOrder)
		feishu.Send("ok future already opened，勿重复开")
		return errMsg
	}
	instId := fmt.Sprintf("%s-USDT-SWAP", symbol)
	return s.TradeLimit(instId, price, size, side, "open")
}

// TradeLimit instId: EOS-USDT, EOS-USDT-SWAP是合约
func (s *Service) TradeLimit(instId, price, size, side, posSide string) (msg string) {
	closePos := posSide == cex.Close
	myOid := util.GenerateOrder()
	arg := map[string]interface{}{
		"tdMode":     "cross", // 全仓币币， 全仓永续
		"side":       strings.ToLower(side),
		"instId":     instId,
		"sz":         size,
		"px":         price,
		"clOrdId":    myOid,
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
	orderType := consts.Margin
	if strings.HasSuffix(instId, "SWAP") {
		orderType = consts.Future
	}
	order := &models.Orders{
		InstId:     instId,
		Cex:        cex.OKE,
		LimitPrice: price,
		Size:       size,
		Side:       side,
		PosSide:    posSide,
		State:      string(core.TRIGGER),
		OrderId:    "",
		MyOid:      myOid,
		OrderType:  orderType,
		Closed:     "N",
		Created:    time.Now(),
		Updated:    time.Now(),
	}
	if strings.HasSuffix(instId, "SWAP") {
		numPerSize := symb.GetFutureLotByInstId(instId)
		order.NumPerSize = numPerSize
	}

	// 异步以便提高主流程效率
	go func() {
		_ = mapper.Insert(s.db, order)
	}()

	//s.ReloadOrders()
	err := s.prvCon.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		panic("发送订阅账户余额数据失败")
	} else {
		return "trigger trade成功, 最终结果见推送数据"
	}
}

func (s *Service) TradeMarket(instId, size, side, posSide string) (msg string) {
	closePos := posSide == cex.Close
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

//
//func (s *Service) CloseMarginMarket(askPrc float64) (msg string) {
//	if s.openMarginOrder == nil || s.openMarginOrder.State != consts.Filled {
//		feishu.Send("ok receive close future signal close, but found no open future")
//		return "no position to close"
//	}
//	if s.closeMarginOrder != nil {
//		return "close order already placed"
//	}
//	side := consts.Buy
//	if s.openMarginOrder.Side == cex.Buy {
//		side = consts.Sell
//	}
//	sizeFloat := numutil.Parse(s.openMarginOrder.Size)
//	if side == consts.Buy {
//		//if buy, size是 金额？？对于 market order 好像是的
//		sizeFloat = sizeFloat * askPrc
//	}
//
//	// if buy in market mode, size refer to the amount of U
//	size := util.AdjustClosePosSize(sizeFloat, side, cex.OKE)
//	return s.TradeMarket(s.openMarginOrder.InstId, size, side, cex.Close)
//}
//func (s *Service) CloseFutureMarket() (msg string) {
//	if s.openFutureOrder == nil || s.openFutureOrder.State != consts.Filled {
//		feishu.Send("ok receive close future signal close, but found no open future")
//		return "no position to close"
//	}
//	if s.closeFutureOrder != nil {
//		return "close order already placed"
//	}
//	side := consts.Buy
//	if s.openFutureOrder.Side == cex.Buy {
//		side = consts.Sell
//	}
//	return s.TradeMarket(s.openFutureOrder.InstId, s.openFutureOrder.Size, side, cex.Close)
//}

//func (s *Service) ClosePosLimit(price string) (msg string) {
//	// 为开仓或者 开的仓位未成交
//	if s.openOrder == nil || s.openOrder.State != string(core.FILLED) {
//		feishu.Send("ok receive signal close, but found no open")
//		return "无仓位需要平仓"
//	}
//	if s.closeOrder != nil {
//		return "无需重复平仓"
//	}
//	side := cex.Buy
//	if s.openOrder.Side == cex.Buy {
//		side = cex.Sell
//	}
//	sizeFloat, _ := strconv.ParseFloat(s.openOrder.Size, 64)
//	size := util.AdjustClosePosSize(sizeFloat, side, cex.OKE)
//	symbol := strings.Split(s.openOrder.InstId, "-")[0]
//	return s.TradeLimit(symbol, price, size, side, cex.Close)
//}
