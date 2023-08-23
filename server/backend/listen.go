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
	"ws-quant/pkg/util"
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
			//log.Info("listenAndExec收到ticker=%+v", tickerBean)
			if tickerBean.SymbolName == bs.executingSymbol {
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

// combine ticker and track bean to exec st or tp；
func (bs *backendServer) listenAndExecStTp() {
	for {
		select {
		case ticker := <-bs.trackTickerChan:
			if bs.marginTrack == nil && bs.futureTrack == nil {
				continue
			}
			//log.Info("listenAndExecStTp 收到 ticker=%+v", ticker)
			isFuture := strings.HasSuffix(ticker.InstId, "SWAP")
			if isFuture {
				if bs.futureTrack != nil && bs.futureTrack.State == orderstate.Filled {
					if bs.futureTrack.Symbol != ticker.SymbolName {
						feishu.Send("exec symbol 和 future Track中的symbol不一致")
						continue
					}
					if bs.futureTrack.SlPrc <= 0 {
						bs.futureTrack.SlPrc = calInitSlPrc(bs.futureTrack.OpenPrc, bs.futureTrack.Side)
					}
					if checkAndModifySl(ticker, bs.futureTrack) {
						log.Info("触发 future stop loss")
						bs.okeService.CloseOrder(bs.futureTrack.InstType)
					}
				}
			} else {
				//receive margin price ticker
				if bs.marginTrack != nil && bs.marginTrack.State == orderstate.Filled {
					if bs.marginTrack.Symbol != ticker.SymbolName {
						feishu.Send("exec symbol 和 marginTrack中的symbol不一致")
						continue
					}
					if bs.marginTrack.SlPrc <= 0 {
						bs.marginTrack.SlPrc = calInitSlPrc(bs.marginTrack.OpenPrc, bs.marginTrack.Side)
					}
					if checkAndModifySl(ticker, bs.marginTrack) {
						log.Info("触发 margin stop loss")
						bs.okeService.CloseOrder(bs.marginTrack.InstType)
					}
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
				//has profit,
				pct := (ticker.PriceBestBid/openPrcFloat - 1) * 100
				pctFloor := math.Floor(pct)
				if pctFloor >= 1 && ticker.PriceBestBid > track.SlPrc {
					newSlPrc := 0.0
					if pct-pctFloor > 0.5 {
						newSlPrc = openPrcFloat * (1 + 0.01*pctFloor)
					} else {
						newSlPrc = openPrcFloat * (1 + 0.01*(pctFloor-0.5))
					}
					if newSlPrc > track.SlPrc {
						track.SlPrc = newSlPrc
						log.Info("buy单移动止损-提高,最新 sl=%v", track.SlPrc)
					}

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
				if pctFloor >= 1 && ticker.PriceBestAsk < track.SlPrc {
					newSlPrc := 0.0
					if pct-pctFloor > 0.5 {
						newSlPrc = openPrcFloat * (1 - 0.01*pctFloor)
					} else {
						newSlPrc = openPrcFloat * (1 - 0.01*(pctFloor-0.5))
					}
					if newSlPrc < track.SlPrc {
						track.SlPrc = newSlPrc
						log.Info("sell单移动止损-降低,最新 sl=%v", track.SlPrc)
					}
				}
			}
		}
	}
	return false
}

// 监听 订单状态: open trigger, open filled 和 close trigger 3个状态
func (bs *backendServer) listenTrackBeanTriggerAndFilled() {
	for {
		select {
		case trackBean := <-bs.trackBeanChan:
			msg := fmt.Sprintf("收到最新的 trackBean: %+v", trackBean)
			feishu.Send(msg)
			if trackBean.PosSide == consts.Close {
				if trackBean.InstType == insttype.Margin {
					bs.marginTrack = nil
				} else if trackBean.InstType == insttype.Future {
					bs.futureTrack = nil
				}
				continue
			}
			//接下来处理 open trigger/filled
			switch trackBean.State {
			case orderstate.TRIGGER:
				if bs.getTrackBean(trackBean.InstType) != nil {
					errMsg := fmt.Sprintf("Alert! 接受到trigger，但是trackBean不为null, track myOid = %s",
						trackBean.MyOidOpen)
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
			case orderstate.Filled:
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
				if trackBean.OpenPrc == "" {
					log.Error("无openPrc value")
					feishu.Send("无openPrc value")
					continue
				}
				currentTrackBean.State = orderstate.Filled
				currentTrackBean.OpenPrc = trackBean.OpenPrc
				currentTrackBean.SlPrc = calInitSlPrc(
					currentTrackBean.OpenPrc, currentTrackBean.Side)

			default:
				log.Error("track listen unknown order state=%v", trackBean.State)
			}
		}
	}
}

func calInitSlPrc(openPrc string, side string) float64 {
	return prcutil.AdjustPriceFloat(numutil.Parse(openPrc), side == consts.Sell, 100)
}

// 监听并流转 策略状态, 接受订单的final 状态 => filled, canceled
func (bs *backendServer) listenOrderState() {
	for {
		select {
		case execState := <-bs.execStateChan:
			msg := fmt.Sprintf("收到最新的state: %+v", execState)
			feishu.Send(msg)
			log.Info(msg)
			if execState.PosSide == consts.Open {
				// 开仓情况下 只会收到 filled 和 canceled 两种
				if execState.OrderState == orderstate.Filled {
					if bs.strategyState != int32(StateOpenSignalled) && bs.strategyState != int32(StateOpenFilledPart) {
						msg := fmt.Sprintf("strategyState 非signal或 partially filled, 但收到了openFilled")
						log.Error(msg)
						feishu.Send(msg)
						continue
					}
					if bs.executingSymbol == "" {
						feishu.Send("已收到 open filled，但是 executingSymbol 为空")
					}
					atomic.AddInt32(&bs.strategyState, 1)
				} else if execState.OrderState == orderstate.Cancelled {
					// 另一个一定 close 且 filled
					otherInstType := util.Select(execState.InstType == insttype.Margin, insttype.Future, insttype.Margin)
					otherClose := bs.okeService.GetCloseOrder(otherInstType)
					if otherClose == nil || otherClose.State != orderstate.Filled {
						feishu.Send("收到cancel, 但是另一个不是 close filled")
						continue
					} else {
						bs.AfterComplete("canceled-closed(SL)")
						continue
					}
				}
			} else if execState.PosSide == consts.Close {
				if execState.InstType == insttype.Margin {
					// track 使命结束
					bs.marginTrack = nil
				}
				if execState.InstType == insttype.Future {
					// track 使命结束
					bs.futureTrack = nil
				}
				/*
					此时有几种情况，
					1 另一个 open 是 live, 则 执行 cancel
						- 等待 cancelled 后执行 all finish, (也就是说cancel只有可能在另一个执行close成功后执行，而且很可能是 SL close)
					2 另一个 open 是 failed, 则 执行 all finish 逻辑
					3 另一个 open 是 filled 等待另一个 close, 之后执行 all finish 逻辑
				*/
				otherInstType := util.Select(execState.InstType == insttype.Margin, insttype.Future, insttype.Margin)
				otherOpen := bs.okeService.GetOpenOrder(otherInstType)
				if otherOpen != nil && otherOpen.State == orderstate.Live {
					bs.okeService.CancelOrder(otherInstType)
					continue
				}
				if otherOpen != nil && otherOpen.State == orderstate.Failed {
					bs.AfterComplete("closed-failed")
					continue
				}
				//将close signal -> closePartFilled
				atomic.AddInt32(&bs.strategyState, 1)

			}
			log.Info("监听上报的订单更新,strategyState=%v", bs.strategyState)
		}
	}
}

func (bs *backendServer) AfterComplete(desc string) {
	msg := "策略全部完成: " + desc
	log.Info(msg)
	feishu.Send(msg)
	bs.strategyState = 0
	bs.executingSymbol = ""
	bs.marginTrack = nil
	bs.futureTrack = nil
	bs.config.StrategyOpenThreshold = 2
	_ = mapper.UpdateById(bs.db, 1, models.Config{StrategyOpenThreshold: 2.0})
	_ = mapper.UpdateByWhere(bs.db, &models.Orders{Closed: "Y"}, "id > ?", 1)
	bs.okeService.ReloadOrders()
	go func() {
		time.Sleep(time.Second * 5)
		bs.persistBalance("strategy-finish")
	}()
}
