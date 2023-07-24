package dingding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ws-quant/pkg/util"
)

type Text struct {
	Content string `json:"content"`
}

type Data struct {
	Msgtype string `json:"msgtype"`
	Text    Text   `json:"text"`
}

func Send(message string) {
	url := "https://oapi.dingtalk.com/robot/send?access_token=2563e9fef6f59d1619b486387b84f3e8a8e28aa384e7b2fe4e2b967b27c7fd9a"
	msg := Data{
		Msgtype: "text",
		Text:    Text{Content: fmt.Sprintf("txl-%s", message)},
	}
	msgBytes, _ := json.Marshal(&msg)
	util.HttpRequest(http.MethodPost, url, string(msgBytes), map[string]string{"Content-Type": "application/json; charset=utf-8"})
}
