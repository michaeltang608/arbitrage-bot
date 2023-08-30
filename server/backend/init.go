package backend

import (
	"strings"
	"time"
	"ws-quant/common/bean"
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

	bs.execStates = []string{"", "", "", ""}

	// 初始化 chan
	bs.tickerChan = make(chan bean.TickerBean, 200)
	bs.trackTickerChan = make(chan bean.TickerBean, 100)

	bs.trackBeanChan = make(chan bean.TrackBean, 100)
	bs.orderStateChan = make(chan bean.ExecState, 20)
}

func (bs *backendServer) PostInit() {
	go func() {
		defer e.Recover()()
		bs.marginTrack = nil
		bs.futureTrack = nil
		//保证 service完成初始化
		time.Sleep(time.Second * 5)
		openMarginOrder := bs.okeService.GetOpenOrder(insttype.Margin)
		openFutureOrder := bs.okeService.GetOpenOrder(insttype.Future)

		closeMarginOrder := bs.okeService.GetCloseOrder(insttype.Margin)
		closeFutureOrder := bs.okeService.GetCloseOrder(insttype.Future)

		// init executing symbol
		if openMarginOrder != nil {
			bs.executingSymbol = strings.Split(openMarginOrder.InstId, "-")[0]
		}

		// init execStates，记录 failed, live, canceled, filled
		if openMarginOrder != nil {
			bs.execStates[0] = openMarginOrder.State
		}
		if closeMarginOrder != nil {
			bs.execStates[1] = closeMarginOrder.State
		}
		if openFutureOrder != nil {
			bs.execStates[2] = openFutureOrder.State
		}
		if closeFutureOrder != nil {
			bs.execStates[3] = closeFutureOrder.State
		}

		// init track bean
		if openMarginOrder != nil && closeMarginOrder == nil {
			if openMarginOrder.State == orderstate.Filled || openMarginOrder.State == orderstate.Live {
				bs.marginTrack = &bean.TrackBean{
					State:     openMarginOrder.State,
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
			if openFutureOrder.State == orderstate.Filled || openFutureOrder.State == orderstate.Live {
				bs.futureTrack = &bean.TrackBean{
					State:     openMarginOrder.State,
					PosSide:   openFutureOrder.PosSide,
					OpenPrc:   openFutureOrder.Price,
					Symbol:    strings.Split(openFutureOrder.InstId, "-")[0],
					Side:      openFutureOrder.Side,
					InstType:  openFutureOrder.OrderType,
					MyOidOpen: openFutureOrder.MyOid,
				}
			}
		}
	}()

}
