package backend

import (
	"strings"
	"time"
	"ws-quant/common/bean"
	"ws-quant/common/consts"
	"ws-quant/common/insttype"
	"ws-quant/common/orderstate"
	"ws-quant/common/symb"
	"ws-quant/pkg/e"
)

func (bs *backendServer) initMapAndChan() {
	// 初始化要监控的 ticker
	// init okex margin-future map，第一个idx存储
	bs.OkMarginFutureMap = make(map[string]*MarginFutureTicker)
	for _, sym := range symb.GetAllOkFuture() {
		bs.OkMarginFutureMap[sym] = &MarginFutureTicker{Symbol: sym}
	}

	// 初始化 chan
	bs.tickerChan = make(chan bean.TickerBean, 200)
	bs.trackTickerChan = make(chan bean.TickerBean, 100)

	bs.trackBeanChan = make(chan bean.TrackBean, 100)
	bs.execStateChan = make(chan bean.ExecState)
}

func (bs *backendServer) PostInit() {
	go func() {
		defer e.Recover()()
		bs.marginTrack = nil
		bs.futureTrack = nil
		time.Sleep(time.Second * 5)
		openMarginOrder := bs.okeService.GetOpenOrder(insttype.Margin)
		openFutureOrder := bs.okeService.GetOpenOrder(insttype.Future)

		closeMarginOrder := bs.okeService.GetCloseOrder(insttype.Margin)
		closeFutureOrder := bs.okeService.GetCloseOrder(insttype.Future)

		if openMarginOrder != nil {
			bs.executingSymbol = strings.Split(openMarginOrder.InstId, "-")[0]
		}
		if openMarginOrder != nil && closeMarginOrder == nil {
			if openMarginOrder.State == orderstate.Filled {
				bs.marginTrack = &bean.TrackBean{
					State:     orderstate.Filled,
					PosSide:   openMarginOrder.PosSide,
					OpenPrc:   openMarginOrder.Price,
					Symbol:    strings.Split(openMarginOrder.InstId, "-")[0],
					Side:      openMarginOrder.Side,
					InstType:  openMarginOrder.OrderType,
					MyOidOpen: openMarginOrder.MyOid,
				}
			}
		}
		if openFutureOrder != nil && closeFutureOrder == nil {
			if openFutureOrder.State == orderstate.Filled {
				bs.futureTrack = &bean.TrackBean{
					State:     orderstate.Filled,
					PosSide:   openFutureOrder.PosSide,
					OpenPrc:   openFutureOrder.Price,
					Symbol:    strings.Split(openFutureOrder.InstId, "-")[0],
					Side:      openFutureOrder.Side,
					InstType:  openFutureOrder.OrderType,
					MyOidOpen: openFutureOrder.MyOid,
				}
			}
		}

		// init strategy state
		if closeMarginOrder != nil || closeFutureOrder != nil {
			bs.strategyState = int32(StateCloseSignalled)
			if closeMarginOrder != nil && closeMarginOrder.State == consts.Filled {
				bs.strategyState = int32(StateCloseFilledPart)
			} else if closeFutureOrder != nil && closeFutureOrder.State == consts.Filled {
				bs.strategyState = int32(StateCloseFilledPart)
			}

		} else if openMarginOrder != nil || openFutureOrder != nil {
			bs.strategyState = int32(StateOpenSignalled)
			if openMarginOrder != nil && openMarginOrder.State == consts.Filled {
				bs.strategyState = int32(StateOpenFilledPart)
			} else if openFutureOrder != nil && openFutureOrder.State == consts.Filled {
				bs.strategyState = int32(StateOpenFilledPart)
			}
		}
		log.Info("程序启动，strategyState=%v", bs.strategyState)
	}()

}
