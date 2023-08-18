package oke

import (
	"encoding/json"
	"net/http"
	"strings"
	"ws-quant/common/consts"
	"ws-quant/pkg/util"
)

type OpenReq struct {
	InstId        string `json:"instId"`
	TdMode        string `json:"tdMode"`
	Ccy           string `json:"ccy"`
	Side          string `json:"side"`
	OrdType       string `json:"ordType"` //
	Sz            string `json:"sz"`
	AlgoClOrdId   string `json:"algoClOrdId"`
	CallbackRatio string `json:"callbackRatio"`
	ActivePx      string `json:"activePx"`
	QuickMgnType  string `json:"quickMgnType"`
}

func (s *Service) StrategyOpenLimit(instId, price, size, side, posSide string) string {
	api := "/api/v5/trade/order-algo"

	quickMgnType := "manual"
	if posSide == consts.Close {
		if strings.HasSuffix(instId, "SWAP") {
			quickMgnType = "auto_borrow"
		}
	}
	req := OpenReq{
		InstId:        instId,
		TdMode:        "cross",
		Ccy:           "USDT",
		Side:          strings.ToLower(side),
		OrdType:       "move_order_stop",
		Sz:            size,
		AlgoClOrdId:   util.GenerateOrder("Strategy"),
		CallbackRatio: "0.01",
		ActivePx:      price,
		QuickMgnType:  quickMgnType,
	}
	reqBytes, _ := json.Marshal(&req)
	body := string(reqBytes)
	resp := execOrder(body, http.MethodPost, api)
	return resp
}
