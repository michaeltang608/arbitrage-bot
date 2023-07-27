package backend

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"time"
	"ws-quant/cex/models"
	"ws-quant/pkg/dingding"
	"ws-quant/pkg/mapper"
)

func (bs *backendServer) scheduleJobs() {
	c := cron.New()
	//每隔1h 清零一次并通知
	_, _ = c.AddFunc("0/5 * * * *", func() {
		maxDiff := fmt.Sprintf("%.2f ", bs.curMax)
		if bs.curMax >= 1.0 {
			dingding.Send("maxDiff=" + maxDiff)
			err := mapper.Insert(bs.db, &models.Oppor{
				InstId:  "margins",
				Cex:     "bi-ok",
				MaxDiff: maxDiff,
				Created: time.Now(),
			})

			if err != nil {
				log.Error("insert err=", err)
				dingding.Send(fmt.Sprintf("insert error:%v", err.Error()))
			}
		}

		bs.curMax = 0.0
		bs.curMaxMarginFuture = 0.0

	})
	c.Start()

}
