package oke

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/common/bean"
	"ws-quant/common/consts"
	"ws-quant/common/insttype"
	"ws-quant/common/orderstate"
	"ws-quant/common/symb"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
	"ws-quant/pkg/util"
)

//本go主要用户close pos

type CloseReq struct {
	InstId  string `json:"instId"`
	MgnMode string `json:"mgnMode"` // 默认 cross
	Ccy     string `json:"ccy"`
	ClOrdId string `json:"clOrdId"`
}

func (s *Service) CloseOrder(instType string) string {
	openOrder := s.GetOpenOrder(instType)

	go func(instType string) {
		//check and close the other live
		otherInstType := util.Select(instType == insttype.Margin, insttype.Future, insttype.Margin)
		otherOpen := s.GetOpenOrder(otherInstType)
		if otherOpen != nil && otherOpen.State == orderstate.Live {
			s.CancelOrder(otherInstType)
		}
	}(instType)

	if openOrder == nil || openOrder.State != orderstate.Filled {
		msg := fmt.Sprintf("收到close margin, but no open %s found", instType)
		feishu.Send(msg)
		return msg
	}

	myOid := util.GenerateOrder("CL")
	// 先持久化
	s.insertCloseOrder(openOrder, myOid)

	// report to track strategy
	s.trackBeanChan <- bean.TrackBean{
		PosSide:  consts.Close,
		InstType: instType,
	}
	instId := openOrder.InstId
	// do request
	api := "/api/v5/trade/close-position"
	req := CloseReq{
		InstId:  instId,
		MgnMode: "cross",
		Ccy:     "USDT",
		ClOrdId: myOid,
	}
	reqBytes, _ := json.Marshal(&req)
	body := string(reqBytes)
	resp := execOrder(body, http.MethodPost, api)
	return resp
}

func (s *Service) insertCloseOrder(openOrder *models.Orders, myOid string) {
	side := consts.Buy
	if openOrder.Side == consts.Buy {
		side = consts.Sell
	}

	instId := openOrder.InstId
	order := &models.Orders{
		InstId:    instId,
		Cex:       cex.OKE,
		Side:      side,
		PosSide:   consts.Close,
		State:     orderstate.TRIGGER,
		MyOid:     myOid,
		OrderType: openOrder.OrderType,
		IsDeleted: "N",
		Created:   time.Now(),
		Updated:   time.Now(),
	}
	if strings.HasSuffix(instId, "SWAP") {
		numPerSize := symb.GetFutureLotByInstId(instId)
		order.NumPerSize = numPerSize
	}
	_ = mapper.Insert(s.db, order)
}

type CancelReq struct {
	InstId string `json:"instId"`
	OrdId  string `json:"ordId"`
}

func (s *Service) CancelOrder(instType string) string {

	order := s.GetOpenOrder(instType)
	if order == nil || order.State != orderstate.Live {
		return "open 非live, 但收到cancel"
	}
	api := "/api/v5/trade/cancel-order"
	req := CancelReq{
		InstId: order.InstId,
		OrdId:  order.OrderId,
	}

	reqBytes, _ := json.Marshal(&req)
	body := string(reqBytes)
	resp := execOrder(body, http.MethodPost, api)
	return resp
}

func (s *Service) QueryLiveOrder(instId string) string {
	/*
		GET /api/v5/trade/order
		GET /api/v5/trade/order?ordId=590910403358593111&instId=BTC-US

	*/
	api := fmt.Sprintf("/api/v5/trade/order?instId=%v", instId)
	return execOrder("", http.MethodGet, api)
}
func execOrder(body, method, api string) string {
	log.Info("开始execOrder: body=%s, method=%s, api=%s", body, method, api)
	now := time.Now()
	utcTime := now.Add(-time.Hour * 8)
	formatTime := utcTime.Format("2006-01-02T15:04:05.000Z")

	signStr := fmt.Sprintf("%s%s%s%s", formatTime, "POST", api, body)
	signature := util.Sha256AndBase64(signStr, apiSecret)

	headers := map[string]string{
		"OK-ACCESS-KEY":        apiKey,
		"OK-ACCESS-SIGN":       signature,
		"OK-ACCESS-TIMESTAMP":  formatTime,
		"OK-ACCESS-PASSPHRASE": pwd,
		"CONTENT-TYPE":         "application/json",
	}

	respBytes := util.HttpRequest(method, baseHttpUrl+api, body, headers)
	resp := string(respBytes)
	log.Info("返回的close的结果是=%v, size=%v", resp, len(resp))
	return resp
}
