package oke

import (
	"encoding/json"
	"fmt"
	"net/http"
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

//本go主要用户close pos

type CloseReq struct {
	InstId  string `json:"instId"`
	PosSide string `json:"posSide"` // 平多是 long, 平空时 short
	MgnMode string `json:"mgnMode"` // 默认 cross
	Ccy     string `json:"ccy"`
}

func (s *Service) CloseOrder(orderType string) string {
	openOrder := s.openMarginOrder
	if orderType == consts.Future {
		openOrder = s.openFutureOrder
	}
	if openOrder == nil || openOrder.State != consts.Filled {
		msg := fmt.Sprintf("收到close margin, but no open %s found", orderType)
		feishu.Send(msg)
		return msg
	}

	instId := openOrder.InstId
	side := consts.Buy
	if openOrder.Side == consts.Buy {
		side = consts.Sell
	}
	order := &models.Orders{
		InstId:    instId,
		Cex:       cex.OKE,
		Side:      side,
		PosSide:   "close",
		State:     string(core.TRIGGER),
		OrderId:   "",
		OrderType: consts.Market,
		Closed:    "N",
		Created:   time.Now(),
		Updated:   time.Now(),
	}
	if strings.HasSuffix(instId, "SWAP") {
		numPerSize := symb.GetFutureLotByInstId(instId)
		order.NumPerSize = numPerSize
	}
	_ = mapper.Insert(s.db, order)

	// do request
	api := "/api/v5/trade/close-position"
	req := CloseReq{
		InstId:  instId,
		MgnMode: "cross",
		Ccy:     "USDT",
	}
	reqBytes, _ := json.Marshal(&req)
	body := string(reqBytes)
	return execPostOrder(body, api)
}

type CancelReq struct {
	InstId string `json:"instId"`
	OrdId  string `json:"ordId"`
}

func cancelOrder(instId, orderId string) string {
	api := "/api/v5/trade/cancel-order"
	req := CancelReq{
		InstId: instId,
		OrdId:  orderId,
	}
	reqBytes, _ := json.Marshal(&req)
	body := string(reqBytes)
	return execPostOrder(body, api)
}
func execPostOrder(body string, api string) string {
	now := time.Now()
	utcTime := now.Add(-time.Hour * 8)
	formatTime := utcTime.Format("2006-01-02T15:04:05.000Z")
	log.Info("formatTime: %v\n", formatTime)

	signStr := fmt.Sprintf("%s%s%s%s", formatTime, "POST", api, body)
	signature := util.Sha256AndBase64(signStr, apiSecret)

	headers := map[string]string{
		"OK-ACCESS-KEY":        apiKey,
		"OK-ACCESS-SIGN":       signature,
		"OK-ACCESS-TIMESTAMP":  formatTime,
		"OK-ACCESS-PASSPHRASE": pwd,
		"CONTENT-TYPE":         "application/json",
	}
	respBytes := util.HttpRequest(http.MethodPost, baseHttpUrl+api, body, headers)
	return string(respBytes)
}
