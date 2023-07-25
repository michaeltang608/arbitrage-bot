package dingding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ws-quant/pkg/util"
)

var (
	sendChan = make(chan string, 100)
	URL      = "https://oapi.dingtalk.com/robot/send?access_token=2563e9fef6f59d1619b486387b84f3e8a8e28aa384e7b2fe4e2b967b27c7fd9a"
)

type Text struct {
	Content string `json:"content"`
}

type Data struct {
	Msgtype string `json:"msgtype"`
	Text    Text   `json:"text"`
}

func init() {
	go doSend()
}
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
func exec(message string) {
	msg := Data{
		Msgtype: "text",
		Text:    Text{Content: fmt.Sprintf("txl-%s", message)},
	}
	msgBytes, _ := json.Marshal(&msg)
	util.HttpRequest(http.MethodPost, URL, string(msgBytes), map[string]string{"Content-Type": "application/json; charset=utf-8"})
}
