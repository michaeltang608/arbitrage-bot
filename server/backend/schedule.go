package backend

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"time"
	"ws-quant/cex/models"
	"ws-quant/pkg/mapper"
)

func (bs *backendServer) scheduleJobs() {
	c := cron.New()
	//每隔1h 清零一次并通知
	_, _ = c.AddFunc("0/5 * * * *", func() {
		//feishu.Send(fmt.Sprintf("前1h的 max是%.2f ", bs.curMax))
		maxOkMarginFutureDiff := fmt.Sprintf("%.2f", bs.curMaxMarginFuture)
		//feishu.Send(fmt.Sprintf("前1h的 margin/future max是%.2f ", bs.curMaxMarginFuture))
		m := &models.Oppor{
			InstId:  "okMarginFuture",
			Cex:     "ok",
			MaxDiff: maxOkMarginFutureDiff,
			Created: time.Now(),
		}
		err := mapper.Insert(bs.db, m)
		if err != nil {
			log.Error("insert err=", err)
		}
		bs.curMax = 0.0
		bs.curMaxMarginFuture = 0.0

	})
	c.Start()

}
