package backend

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"time"
	"ws-quant/cex/models"
	"ws-quant/common/insttype"
	"ws-quant/common/orderstate"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
)

func (bs *backendServer) scheduleJobs() {
	c := cron.New()
	//每隔10m, check and cancel live order that does not come to a deal after an hour
	_, _ = c.AddFunc("0/5 * * * *", func() {
		bs.checkAndCancelExpiredLiveOrder()
	})

	// 每隔五分钟统计 max diff
	_, _ = c.AddFunc("0/5 * * * *", func() {
		maxOkMarginFutureDiff := fmt.Sprintf("%.2f", bs.maxDiffMarginFuture)
		if bs.maxDiffMarginFuture >= 1.0 {
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
			feishu.Send(fmt.Sprintf("前2m的 margin/future max是%.2f ", bs.maxDiffMarginFuture))

		}
		bs.maxDiffMarginFuture = 0.0

	})
	c.Start()

}

func (bs *backendServer) checkAndCancelExpiredLiveOrder() {
	openMargin := bs.okeService.GetOpenOrder(insttype.Margin)
	openFuture := bs.okeService.GetOpenOrder(insttype.Future)
	if openMargin != nil && openMargin.OrderType == orderstate.Live {
		if openMargin.Created.Before(time.Now().Add(-time.Minute * 20)) {
			msg := "cancel expired live " + insttype.Margin
			feishu.Send(msg)
			bs.okeService.CancelOrder(insttype.Margin)
		}
	}
	if openFuture != nil && openFuture.OrderType == orderstate.Live {
		if openFuture.Created.Before(time.Now().Add(-time.Minute * 20)) {
			msg := "cancel expired live " + insttype.Future
			feishu.Send(msg)
			bs.okeService.CancelOrder(insttype.Future)
		}
	}
}
