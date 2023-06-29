package kucoin

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"time"
)

type Subscribe struct {
	Id             int64  `json:"id"`
	Type           string `json:"type"`
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response"`
}

func subscribe(topic string, conn *websocket.Conn, private bool) {
	subscribe := Subscribe{
		Id:             time.Now().UnixMilli(),
		Type:           "subscribe",
		Topic:          topic,
		PrivateChannel: private,
		Response:       true,
	}
	subscribeBytes, _ := json.Marshal(subscribe)
	connIsNull := conn == nil
	log.Info("connIsNull=%v, topic=%v,private=%v", connIsNull, topic, private)
	err := conn.WriteMessage(websocket.TextMessage, subscribeBytes)
	if err != nil {
		log.Error("subscribe失败: %v\n", err)
	} else {
		log.Info("订阅prv主题%v成功", topic)
	}
}
