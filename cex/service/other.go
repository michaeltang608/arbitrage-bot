package service

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"strings"
	"ws-quant/cex"
)

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
