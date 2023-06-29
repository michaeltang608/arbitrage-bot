package e

import (
	"fmt"
	"ws-quant/pkg/feishu"
	logger "ws-quant/pkg/log"
)

var (
	log2 = logger.NewLog("kucoinLog")
)

// Recover 此处抓住异常，打印日志并告警然后直接退出。所以需要再新的 goroutine一开始调用一次即可
func Recover() func() {
	return func() {
		if any_ := recover(); any_ != nil {
			log2.Info("程序异常，准备退出: %v", any_)
			feishu.Send(fmt.Sprintf("程序异常，退出,%v", any_))
		}
	}
}
