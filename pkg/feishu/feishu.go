package feishu

import (
	"ws-quant/pkg/util"
)

var (
	sendChan = make(chan string, 100)
	URL      = ""
)

func init() {
	go doSend()
}

// Send 异步发送，提高tps
func Send(msg string) {
	sendChan <- msg
}

// 异步发送
func doSend() {
	for {
		select {
		case msg := <-sendChan:
			exec(msg)
		}
	}

}

func exec(msg string) {
	// 配置自己的url 地址
	if URL == "" {
		return
	}
	data := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": "[txl]-" + msg,
		},
	}
	util.SendPost(URL, data)
	//log.Info("resp: %v\n", resp)
}
