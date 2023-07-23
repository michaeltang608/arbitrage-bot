package binan

import (
	"fmt"
	"github.com/gorilla/websocket"
	"time"
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
	errCnt := 0
	for {
		if s.prvCon == nil {
			log.Panic("prvCon == nil")
		}
		_, msgBytes, err := s.prvCon.ReadMessage()
		if err != nil {
			errCnt += 1
			log.Error("read msg err=", err)
			time.Sleep(time.Second)
			if errCnt > 10 {
				log.Panic("累计多次读取失败，退出")
			}
		}
		errCnt = 0
		log.Info("接受到的prv数据是%v\n", string(msgBytes))
	}
}
