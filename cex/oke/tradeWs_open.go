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
	"ws-quant/common/insttype"
	"ws-quant/common/orderstate"
	"ws-quant/common/symb"
	"ws-quant/models/bean"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
	"ws-quant/pkg/util"
)

func (s *Service) OpenLimit(instType, symbol, price, size, side string) (msg string) {

	if s.GetOpenOrder(instType) != nil {
		errMsg := fmt.Sprintf("%s，勿再重复开仓", instType)
		feishu.Send(errMsg)
		return errMsg
	}
	return s.TradeLimit(instType, symbol, price, size, side, "open")
}

// TradeLimit instId: EOS-USDT, EOS-USDT-SWAP是合约
func (s *Service) TradeLimit(instType, symbol, price, size, side, posSide string) (msg string) {
	instId := util.Select(instType == insttype.Margin,
		fmt.Sprintf("%s-USDT", symbol), fmt.Sprintf("%s-USDT-SWAP", symbol))

	closePos := posSide == cex.Close
	myOid := util.GenerateOrder("OP")
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
		State:      orderstate.TRIGGER,
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
	s.trackBeanChan <- bean.TrackBean{
		State:     orderstate.TRIGGER,
		Side:      side,
		OrderType: "",
		OpenPrc:   "",
		SlPrc:     "",
		TpPrc:     "",
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
