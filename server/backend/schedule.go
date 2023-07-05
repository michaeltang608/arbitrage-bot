package backend

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"ws-quant/pkg/feishu"
)

func (bs *backendServer) scheduleJobs() {
	//定时同步账户余额, 每两个小时记录一次
	c := cron.New()
	//_, _ = c.AddFunc("0 0/6 * * *", func() {
	//	bs.persistBalance("cron")
	//})

	//每隔1h 清零一次并通知
	_, _ = c.AddFunc("0 * * * *", func() {
		feishu.Send(fmt.Sprintf("前1h的 max是%.2f ", bs.curMax))
		bs.curMax = 0.0

	})
	c.Start()

}
