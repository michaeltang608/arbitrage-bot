package service

import (
	"encoding/json"
	"github.com/gorilla/websocket"
)

func (s *Service) MarginBalance() float64 {
	return s.usdtBal
}

// 暂时用不到
func (s *Service) subscribeAccount() {
	s.doSubscribe("account")

}

// 订阅仓位信息，更清楚地了解，目前该还的债务，用于平仓
func (s *Service) subscribePosition() {
	err := s.doSubscribeV2(map[string]interface{}{
		"channel":  "positions",
		"instType": "MARGIN",
	})
	if err != nil {
		log.Info("发送订阅position失败: " + err.Error())
	} else {
		log.Info("发送订阅position成功")
	}

}
func (s *Service) subscribeBalanceAndPos() {
	s.doSubscribe("balance_and_position")
}

func (s *Service) doSubscribe(channelName string) {
	accountArg := make(map[string]interface{})
	accountArg["channel"] = channelName

	accountReq := Req{
		Op: "subscribe",
		Args: []map[string]interface{}{
			accountArg,
		},
	}
	req2, _ := json.Marshal(accountReq)
	err := s.prvCon.WriteMessage(websocket.TextMessage, req2)
	if err != nil {
		panic("发送订阅账户余额数据失败")
	} else {
		log.Info("发送订阅%v数据成功", channelName)
	}

}

func (s *Service) doSubscribeV2(arg map[string]interface{}) error {
	accountReq := Req{
		Op: "subscribe",
		Args: []map[string]interface{}{
			arg,
		},
	}
	req2, _ := json.Marshal(accountReq)
	return s.prvCon.WriteMessage(websocket.TextMessage, req2)
}
