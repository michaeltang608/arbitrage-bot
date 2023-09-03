package backend

import (
	"fmt"
	"ws-quant/common/insttype"
	"ws-quant/common/symb"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/util"
	"ws-quant/pkg/util/numutil"
	"ws-quant/pkg/util/prcutil"
)

func (bs *backendServer) execOpenLimit(openSignal int, t *MarginFutureTicker, curDiff float64) {
	msg := fmt.Sprintf("trigger&exec open limit strategy,curDiff=%v, ticker= %+v", curDiff, t)
	feishu.Send(msg)
	log.Info(msg)
	bs.executingSymbol = t.Symbol
	log.Info("signalOpen, triggerState=%v", bs.triggerState)

	// 先处理 margin, 再处理 future
	// 计算size
	symbolPrc := t.AskFuture
	numPerUnit := symb.GetFutureLot(t.Symbol)
	if numPerUnit == "" {
		log.Error("未找到该future 的unitNum, %v", t.Symbol)
		feishu.Send("未找到该future 的unitNum")
		return
	}

	tradeAmt, futureSize := bs.calFutureSizeAndTradeAmt(t.Symbol, symbolPrc, numutil.Parse(numPerUnit))
	if futureSize == 0 {
		feishu.Send("futureSize=0, require manual attention!")
		bs.triggerState = 0
		return
	}
	go func(tradeAmt float64) {
		// margin
		priceF := 0.0
		size := util.NumTrunc(tradeAmt / t.AskFuture)
		side := "buy"
		priceF = t.AskMargin
		if openSignal > 0 {
			side = "sell"
			priceF = t.BidMargin
		}
		price := prcutil.AdjustPrice(priceF, side, curDiff)
		log.Info("ok margin prepare open pos, side=%v, symbol=%v, price=%v, size=%v\n", side, t.Symbol, price, size)
		openResult := bs.okeService.OpenOrderLimit(insttype.Margin, t.Symbol, price, size, side)
		log.Info("ok margin-open pos result:" + openResult)
	}(tradeAmt)
	go func(size int) {
		// future
		side := "sell"
		priceF := t.BidFuture
		if openSignal > 0 {
			side = "buy"
			priceF = t.AskFuture
		}
		price := prcutil.AdjustPrice(priceF, side, curDiff)
		log.Info("prepare to open pos, side=%v, symbol=%v, price=%v, size=%v\n", side, t.Symbol, price, size)
		openResult := bs.okeService.OpenOrderLimit(insttype.Future, t.Symbol, price, numutil.FormatInt(size), side)
		log.Info("ok future open-pos result:" + openResult)

	}(futureSize)
	feishu.Send("strategy open triggered")
}

func (bs *backendServer) execCloseMarket(t *MarginFutureTicker) {
	feishu.Send(fmt.Sprintf("trigger&exec close market strategy, symb=%sA", t.Symbol))
	log.Info("signal close, strategyState=%v", bs.triggerState)
	go func(askPrc float64) {
		msg := bs.okeService.CloseOrder(insttype.Margin)
		log.Info("exec close margin market result: %v\n", msg)
	}(t.AskMargin)

	go func() {
		msg := bs.okeService.CloseOrder(insttype.Future)
		log.Info("exec close future market result: %v\n", msg)
	}()
}
