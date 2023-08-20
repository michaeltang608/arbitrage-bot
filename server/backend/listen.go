package backend

import (
	"fmt"
	"math"
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
	"ws-quant/pkg/util/numutil"
	"ws-quant/pkg/util/prcutil"
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

			//send to strategy monitor
			if tickerBean.SymbolName == bs.executingSymbol {
				log.Info("tickerBean send to strategy monitor")
				bs.trackTickerChan <- tickerBean
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

// combine ticker and track bean to exec st or tp
func (bs *backendServer) listenAndExecStTp() {
	for {
		select {
		case ticker := <-bs.trackTickerChan:
			if bs.marginTrack == nil && bs.futureTrack == nil {
				continue
			}
			if bs.marginTrack != nil && bs.marginTrack.State == orderstate.Filled {
				if bs.marginTrack.Symbol != ticker.SymbolName {
					feishu.Send("exec symbol 和 marginTrack中的symbol不一致")
					continue
				}
				if checkAndModifySl(ticker, bs.marginTrack) {
					log.Info("触发 stop loss")
					bs.okeService.CloseOrder(bs.marginTrack.InstType)
				}
			}
		}
	}
}

// check if Sl triggered and move/modify sl if necessary
func checkAndModifySl(ticker bean.TickerBean, track *bean.TrackBean) bool {
	side := track.Side
	sl := track.SlPrc
	if side == consts.Buy {
		if ticker.PriceBestBid <= sl {
			log.Info("目前bestBid 已经触发该买单止损")
			return true
		} else {
			openPrcFloat := numutil.Parse(track.OpenPrc)
			if ticker.PriceBestBid > openPrcFloat {
				//has profit, step by 0.5 pct
				pct := (ticker.PriceBestBid/openPrcFloat - 1) * 100
				pctFloor := math.Floor(pct)
				if pctFloor >= 1 {
					if pct-pctFloor > 0.5 {
						track.SlPrc = openPrcFloat * (1 + 0.01*pctFloor)
					} else {
						track.SlPrc = openPrcFloat * (1 + 0.01*(pctFloor-0.5))
					}
					log.Info("buy单移动止损-提高,最新 sl=%v", track.SlPrc)
				}
			}
		}
	} else {
		//side = sell
		if ticker.PriceBestAsk >= sl {
			log.Info("目前bestBid 已经触发该买单止损")
			return true
		} else {
			openPrcFloat := numutil.Parse(track.OpenPrc)
			if ticker.PriceBestAsk < openPrcFloat {
				//has profit, step by 0.5 pct
				pct := (openPrcFloat/ticker.PriceBestAsk - 1) * 100
				pctFloor := math.Floor(pct)
				if pctFloor >= 1 {
					if pct-pctFloor > 0.5 {
						track.SlPrc = openPrcFloat * (1 - 0.01*pctFloor)
					} else {
						track.SlPrc = openPrcFloat * (1 - 0.01*(pctFloor-0.5))
					}
					log.Info("sell单移动止损-降低,最新 sl=%v", track.SlPrc)
				}
			}
		}
	}
	return false
}

func (bs *backendServer) listenTrackBean() {
	for {
		select {
		case trackBean := <-bs.trackBeanChan:
			msg := fmt.Sprintf("收到最新的 trackBean: %+v", trackBean)
			feishu.Send(msg)

			if bs.getTrackBean(trackBean.InstType) == nil {
				if trackBean.State != orderstate.TRIGGER {
					errMsg := fmt.Sprintf("Alert! trackBean %s 未初始化，直接收到state = %s",
						trackBean.InstType, trackBean.State)
					feishu.Send(errMsg)
					log.Error(errMsg)
					continue
				}
				newTrackBean := &bean.TrackBean{
					InstType:  trackBean.InstType,
					Side:      trackBean.Side,
					MyOidOpen: trackBean.MyOidOpen,
					Symbol:    trackBean.Symbol,
				}
				if trackBean.InstType == insttype.Margin {
					bs.marginTrack = newTrackBean
				} else {
					bs.futureTrack = newTrackBean
				}
			}

			currentTrackBean := bs.getTrackBean(trackBean.InstType)
			if currentTrackBean == nil {
				feishu.Send("track bean 逻辑错误")
				continue
			}
			if currentTrackBean.MyOidOpen != trackBean.MyOidOpen {
				errMsg := fmt.Sprintf("Alert! trackBean %s myOid不符合，myOid= %s", trackBean.InstType, trackBean.MyOidOpen)
				feishu.Send(errMsg)
				log.Error(errMsg)
				continue
			}
			currentTrackBean.State = trackBean.State
			if trackBean.OpenPrc != "" {
				currentTrackBean.OpenPrc = trackBean.OpenPrc
				currentTrackBean.SlPrc = prcutil.AdjustPriceFloat(
					numutil.Parse(trackBean.OpenPrc), currentTrackBean.Side == consts.Sell, 100)

			}

			if trackBean.State == orderstate.Closed {
				log.Info("平仓，关闭追踪")
				if trackBean.InstType == insttype.Margin {
					bs.marginTrack = nil
				}
				if trackBean.InstType == insttype.Future {
					bs.futureTrack = nil
				}
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
