package e

import (
	"fmt"
	"log"
	"time"
	"ws-quant/pkg/dingding"
)

// Recover 此处抓住异常，打印日志并告警然后直接退出
func Recover() func() {
	return func() {
		if any_ := recover(); any_ != nil {
			log.Printf("程序异常，准备退出: %v", any_)
			dingding.Send(fmt.Sprintf("程序异常，退出,%v", any_))
			time.Sleep(time.Second * 3)
			log.Panic(any_)
		}
	}
}
