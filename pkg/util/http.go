package util

import (
	"bytes"
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"io/ioutil"
	"log"
	"net/http"
)

func SendPost(url string, data map[string]interface{}) string {
	dataBytes, _ := json.Marshal(data)
	reader := bytes.NewBuffer(dataBytes)

	resp, err := http.Post(url, "application/json;charset=utf-8", reader)
	if err != nil {
		log.Panic("http post err: ", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return string(body)

}

func HttpRequest(method, url, body string, headers map[string]string) []byte {
	client := resty.New()
	resp, err := client.R().SetHeaders(headers).SetBody(body).Execute(method, url)
	if err != nil {
		log.Panic("请求失败", err)
	}
	return resp.Body()
}
