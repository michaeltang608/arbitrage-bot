package binan

import (
	"fmt"
	"github.com/gorilla/websocket"
)

func (s *service) ListenAndNotifyPrivate() {
	s.connectAndSubsPrv()
	s.listenAndNotifyPrv()
}
func (s *service) connectAndSubsPrv() {

	url := "wss://stream.binance.com:9443/ws/%s"
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf(url, CreateListenKey()), nil)
	if err != nil {
		log.Panic("连接并订阅 prv err=", err)
	}
	if conn == nil {
		log.Panic("排查，为啥 prvConn为nil")
	}
	s.prvCon = conn
}

func (s *service) listenAndNotifyPrv() {
	for {
		if s.pubCon == nil {
			log.Panic("pubCon == nil")
		}
		_, msgBytes, err := s.pubCon.ReadMessage()
		if err != nil {
			log.Panic("read msg err=", err)
		}
		log.Info("接受到的prv数据是%v\n", string(msgBytes))
	}
}
