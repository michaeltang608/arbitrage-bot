package backend

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/common/bean"
	"ws-quant/common/consts"
	"ws-quant/common/insttype"
	"ws-quant/common/orderstate"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
)

// 负责监听和搜集数据
func (bs *backendServer) listenAndExec() {
	for {
		select {
		case tickerBean := <-bs.tickerChan:
			ticker, ok := bs.OkMarginFutureMap[tickerBean.SymbolName]
			if !ok {
				log.Error("未能找到%v中map数据", tickerBean.SymbolName)
				continue
			}

			if strings.HasSuffix(tickerBean.InstId, "-SWAP") {
				// future perpetual
				ticker.AskFuture = tickerBean.PriceBestAsk
				ticker.BidFuture = tickerBean.PriceBestBid
			} else {
				// margin
				ticker.AskMargin = tickerBean.PriceBestAsk
				ticker.BidMargin = tickerBean.PriceBestBid
			}
			if ticker.AskFuture <= 0 || ticker.AskMargin <= 0 {
				// 还有一半的数据未收到不进行计算
				continue
			}

			openSignal, curDiff := bs.realDiff(ticker)
			if bs.maxDiffMarginFuture < curDiff {
				bs.maxDiffMarginFuture = curDiff
				log.Info("curMaxMarginFuture=%v, symbol=%v\n", bs.maxDiffMarginFuture, tickerBean.SymbolName)
			}

			if openSignal != 0 {
				if atomic.CompareAndSwapInt32(&bs.strategyState, 0, 1) {
					bs.execOpenLimit(openSignal, ticker, curDiff)
				}
			}

			// close position
			if bs.strategyState == int32(StateOpenFilledAll) {
				if strings.ToUpper(ticker.Symbol) == strings.ToUpper(bs.executingSymbol) {
					if bs.shouldClose(ticker) {
						if atomic.CompareAndSwapInt32(&bs.strategyState, int32(StateOpenFilledAll), int32(StateCloseSignalled)) {
							bs.execCloseMarket(ticker)
						}
					}
				}
			}

			if bs.config.LogTicker == LogOke && tickerBean.CexName == cex.OKE {
				log.Info("收到ticker数据，%+v", tickerBean)
			}
		}
	}
}

func (bs *backendServer) listenTrackBean() {
	for {
		select {
		case trackBean := <-bs.trackBeanChan:
			msg := fmt.Sprintf("收到最新的 trackBean: %+v", trackBean)
			feishu.Send(msg)

			if trackBean.InstType == insttype.Margin {
				if bs.marginTrack == nil {
					if trackBean.State != orderstate.TRIGGER {
						errMsg := fmt.Sprintf("Alert! trackBean margin未初始化，直接收到state = %s", trackBean.State)
						feishu.Send(errMsg)
						log.Error(errMsg)
						continue
					}
					bs.marginTrack = &bean.TrackBean{InstType: insttype.Margin}
					bs.marginTrack.Side = trackBean.Side
					bs.marginTrack.MyOidOpen = trackBean.MyOidOpen
				}

				if bs.marginTrack.MyOidOpen != trackBean.MyOidOpen {
					errMsg := fmt.Sprintf("Alert! trackBean margin myOid不符合，myOid= %d", trackBean.MyOidOpen)
					feishu.Send(errMsg)
					log.Error(errMsg)
					continue
				}
				bs.marginTrack.State = trackBean.State
			}

			if trackBean.InstType == insttype.Future {
				if bs.futureTrack == nil {
					if trackBean.State != orderstate.TRIGGER {
						errMsg := fmt.Sprintf("Alert! trackBean future 未初始化，直接收到state = %s", trackBean.State)
						feishu.Send(errMsg)
						log.Error(errMsg)
						continue
					}
					bs.futureTrack = &bean.TrackBean{InstType: insttype.Future}
					bs.futureTrack.Side = trackBean.Side
					bs.futureTrack.MyOidOpen = trackBean.MyOidOpen
				}

				if bs.futureTrack.MyOidOpen != trackBean.MyOidOpen {
					errMsg := fmt.Sprintf("Alert! trackBean future myOid不符合，myOid= %d", trackBean.MyOidOpen)
					feishu.Send(errMsg)
					log.Error(errMsg)
					continue
				}
				bs.futureTrack.State = trackBean.State
			}
		}
	}
}

// 监听并流转 策略状态
func (bs *backendServer) listenState() {
	for {
		select {
		case execState := <-bs.execStateChan:

			msg := fmt.Sprintf("收到最新的state: %+v", execState)
			feishu.Send(msg)
			log.Info(msg)
			if execState.PosSide == consts.Open {
				if bs.strategyState != int32(StateOpenSignalled) && bs.strategyState != int32(StateOpenFilledPart) {
					msg := fmt.Sprintf("strategyState是%v, 但收到了openFilled", bs.strategyState)
					log.Error(msg)
					feishu.Send(msg)
				}
				r := atomic.AddInt32(&bs.strategyState, 1)
				if r == int32(StateOpenFilledAll) {
					//该监听 symbol的close 阈值了
					if bs.executingSymbol == "" {
						feishu.Send("strategyState已经是3，但是 executingSymbol 为空")
					}
				}

			} else if execState.PosSide == consts.Close {
				//if bs.strategyState != int32(StateCloseSignalled) && bs.strategyState != int32(StateCloseFilledPart) {
				//	msg := fmt.Sprintf("strategyState是%v, 但收到了closeFilled", bs.strategyState)
				//	log.Error(msg)
				//	feishu.Send(msg)
				//}
				//r := atomic.AddInt32(&bs.strategyState, 1)
				if bs.allFinished() {
					log.Info("策略全部完成")
					// 调高下次触发的条件，防止立即再次触发
					bs.config.StrategyOpenThreshold = 2
					//防止程序重启数据丢失
					mapper.UpdateById(bs.db, 1, &models.Config{StrategyOpenThreshold: 2})
					bs.strategyState = 0
					bs.executingSymbol = ""
					go func() {
						time.Sleep(time.Second * 5)
						bs.persistBalance("strategy-finish")
					}()
				}

			}
			log.Info("监听上报的订单更新,strategyState=%v", bs.strategyState)
			feishu.Send(fmt.Sprintf("strategyState最新值:%v", bs.strategyState))
		}
	}
}

func (bs *backendServer) allFinished() bool {

	return false
}
