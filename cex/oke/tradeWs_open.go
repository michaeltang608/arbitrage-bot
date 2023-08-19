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
	orderReq := bean.OrderReq{
		InstType: instType,
		Symbol:   symbol,
		Price:    price,
		Size:     size,
		Side:     side,
		PosSide:  consts.Open,
	}
	return s.TradeLimit(orderReq)
}

func (s *Service) TradeLimit(r bean.OrderReq) (msg string) {
	instId := util.Select(r.InstType == insttype.Margin,
		fmt.Sprintf("%s-USDT", r.Symbol), fmt.Sprintf("%s-USDT-SWAP", r.Symbol))

	myOid := util.GenerateOrder("OP")
	arg := map[string]interface{}{
		"tdMode":     "cross", // 全仓币币， 全仓永续
		"side":       strings.ToLower(r.Side),
		"instId":     instId,
		"sz":         r.Size,
		"px":         r.Price,
		"clOrdId":    myOid,
		"ccy":        "USDT",
		"ordType":    "limit", //market, limit, ioc
		"reduceOnly": r.PosSide == consts.Close,
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
	order := buildOrder(r, instId, myOid)

	// report state
	s.trackBeanChan <- bean.TrackBean{
		State:    orderstate.TRIGGER,
		Side:     r.Side,
		InstType: r.InstType,
	}

	// 异步以便提高主流程效率
	go func() {
		_ = mapper.Insert(s.db, order)
	}()

	err := s.prvCon.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		panic("发送订阅账户余额数据失败")
	} else {
		return "trigger trade成功, 最终结果见推送数据"
	}
}

func buildOrder(r bean.OrderReq, instId, myOid string) *models.Orders {
	order := &models.Orders{
		InstId:     instId,
		Cex:        cex.OKE,
		LimitPrice: r.Price,
		Size:       r.Size,
		Side:       r.Side,
		PosSide:    r.PosSide,
		State:      orderstate.TRIGGER,
		OrderId:    "",
		MyOid:      myOid,
		OrderType:  r.InstType,
		Closed:     "N",
		Created:    time.Now(),
		Updated:    time.Now(),
	}
	if r.InstType == insttype.Future {
		numPerSize := symb.GetFutureLotByInstId(instId)
		order.NumPerSize = numPerSize
	}
	return order
}
