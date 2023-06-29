package feishu

import (
	"ws-quant/pkg/util"
)

var (
	sendChan = make(chan string, 100)
	URL      = "https://open.feishu.cn/open-apis/bot/v2/hook/6a353f36-5ea5-43db-becc-5f2da50931ee"
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
	data := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": "[txl]-" + msg,
		},
	}
	util.SendPost(URL, data)
	//log.Info("resp: %v\n", resp)
}
