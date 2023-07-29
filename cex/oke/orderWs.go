package oke

import (
	"encoding/json"
	"github.com/gorilla/websocket"
)

// 订单更新
func (s *Service) subscribeOrder() {
	accountArg := make(map[string]interface{})
	accountArg["channel"] = "orders"
	accountArg["instType"] = "ANY"
	//accountArg["ccy"] = "usdt"

	accountReq := Req{
		Op: "subscribe",
		Args: []map[string]interface{}{
			accountArg,
		},
	}
	req2, _ := json.Marshal(accountReq)
	err := s.prvCon.WriteMessage(websocket.TextMessage, req2)
	if err != nil {
		panic("发送订阅订单数据失败")
	} else {
		log.Info("发送订阅订单数据成功")
	}
}
