package backend

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"ws-quant/pkg/feishu"
)

func (bs *backendServer) scheduleJobs() {
	c := cron.New()
	//每隔1h 清零一次并通知
	_, _ = c.AddFunc("0 * * * *", func() {
		feishu.Send(fmt.Sprintf("前1h的 max是%.2f ", bs.curMax))
		feishu.Send(fmt.Sprintf("前1h的 margin future max是%.2f ", bs.curMaxMarginFuture))
		bs.curMax = 0.0
		bs.curMaxMarginFuture = 0.0

	})
	c.Start()

}
