package kucoin

import (
	"time"
	"ws-quant/pkg/feishu"
)

/*
检查 pong的及时性，如果超时则告警，后期重新连接
*/

func (s *service) checkPong() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		// 5分钟超时则重连
		if s.lastPong > 0 && (s.lastPong+int64(time.Minute*5)) < time.Now().Unix() {
			feishu.Send("kucoin pong expire, began to restart")
			s.ConnectAndSubscribe()
		}
	}
}
