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
				if atomic.CompareAndSwapInt32(&bs.triggerState, 0, 1) {
					bs.execOpenLimit(openSignal, ticker, curDiff)
				}
			}

			// close position
			if bs.execStates[0] == orderstate.Filled && bs.execStates[2] == orderstate.Filled {
				if strings.ToUpper(ticker.Symbol) == strings.ToUpper(bs.executingSymbol) {
					if bs.shouldClose(ticker) {
						if atomic.CompareAndSwapInt32(&bs.triggerState, 1, 2) {
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

// 监听 订单状态: open trigger/filled 和 close trigger/filled 4个状态
func (bs *backendServer) listenTrackBeanTriggerAndFilled() {
	for {
		select {
		case trackBean := <-bs.trackBeanChan:
			msg := fmt.Sprintf("收到最新的 trackBean: %+v", trackBean)
			log.Info(msg)
			if trackBean.PosSide == consts.Close {
				if trackBean.InstType == insttype.Margin {
					bs.marginTrack = nil
				} else if trackBean.InstType == insttype.Future {
					bs.futureTrack = nil
				}
				continue
			}
			feishu.Send(msg)
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

// 监听并流转 策略状态, 接受订单的 状态 => open/close live/filled, open canceled/failed, 不包含 trigger
func (bs *backendServer) listenOrderState() {
	for {
		select {
		case orderStateBean := <-bs.orderStateChan:
			msg := fmt.Sprintf("收到最新的state: %+v", orderStateBean)
			feishu.Send(msg)
			log.Info(msg)
			if orderStateBean.InstType == insttype.Margin {
				if orderStateBean.PosSide == consts.Open {
					bs.execStates[0] = orderStateBean.OrderState
				} else if orderStateBean.PosSide == consts.Close {
					bs.execStates[1] = orderStateBean.OrderState
				} else {
					feishu.Send("unknown posSide=" + orderStateBean.PosSide)
				}
			} else if orderStateBean.InstType == insttype.Future {
				if orderStateBean.PosSide == consts.Open {
					bs.execStates[2] = orderStateBean.OrderState
				} else if orderStateBean.PosSide == consts.Close {
					bs.execStates[3] = orderStateBean.OrderState
				} else {
					feishu.Send("unknown posSide=" + orderStateBean.PosSide)
				}
			} else {
				feishu.Send("unknown instType=" + orderStateBean.InstType)
			}
			openMarginState := bs.execStates[0]
			openFutureState := bs.execStates[2]
			closeMarginState := bs.execStates[1]
			closeFutureState := bs.execStates[3]
			marginCompleted := openMarginState == orderstate.Failed || openMarginState == orderstate.Cancelled || closeMarginState == orderstate.Filled
			futureCompleted := openFutureState == orderstate.Failed || openFutureState == orderstate.Cancelled || closeFutureState == orderstate.Filled
			if marginCompleted && futureCompleted {
				bs.AfterComplete(strings.Join(bs.execStates, "-"))
			}
		}
	}
}

func (bs *backendServer) AfterComplete(desc string) {
	msg := "策略全部完成: " + desc
	bs.Refresh()
	log.Info(msg)
	feishu.Send(msg)
	bs.config.StrategyOpenThreshold = 2
	_ = mapper.UpdateById(bs.db, 1, models.Config{StrategyOpenThreshold: 2.0})
	_ = mapper.UpdateByWhere(bs.db, &models.Orders{IsDeleted: "Y"}, "id > ?", 1)
	go func() {
		time.Sleep(time.Second * 5)
		bs.persistBalance("strategy-finish")
	}()
}

func (bs *backendServer) Refresh() {
	_ = mapper.UpdateByWhere(bs.db, &models.Orders{IsDeleted: "Y"}, "id > ?", 0)
	bs.triggerState = 0
	bs.executingSymbol = ""
	bs.marginTrack = nil
	bs.futureTrack = nil
	bs.execStates = []string{"", "", "", ""}
	bs.okeService.ReloadOrders()
}
