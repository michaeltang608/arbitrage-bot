package backend

//func (bs *backendServer) listenOrderState() {
//	for {
//		select {
//		case execState := <-bs.orderStateChan:
//			msg := fmt.Sprintf("收到最新的state: %+v", execState)
//			feishu.Send(msg)
//			log.Info(msg)
//			/*
//				此时有几种情况，
//				1 另一个 open 是 cancelled 后执行 all finish, (也就是说cancel只有可能在另一个执行close成功后执行，而且很可能是 SL close)
//				2 另一个 open 是 failed, 则 执行 all finish 逻辑
//				3 另一个 open 是 filled 等待另一个 close, 之后执行 all finish 逻辑
//			*/
//			if execState.PosSide == consts.Open {
//
//				// 开仓情况下 只会收到 filled 和 canceled 两种
//				if execState.OrderState == orderstate.Filled {
//					if bs.strategyState != int32(StateOpenSignalled) && bs.strategyState != int32(StateOpenFilledPart) {
//						msg := fmt.Sprintf("strategyState 非signal或 partially filled, 但收到了openFilled")
//						log.Error(msg)
//						feishu.Send(msg)
//						continue
//					}
//					if bs.executingSymbol == "" {
//						feishu.Send("已收到 open filled，但是 executingSymbol 为空")
//					}
//					atomic.AddInt32(&bs.strategyState, 1)
//				} else if execState.OrderState == orderstate.Cancelled {
//					// 另一个一定 close 且 filled
//					otherInstType := util.Select(execState.InstType == insttype.Margin, insttype.Future, insttype.Margin)
//					otherClose := bs.okeService.GetCloseOrder(otherInstType)
//					if otherClose == nil || otherClose.State != orderstate.Filled {
//						feishu.Send("收到cancel, 但是另一个不是 close filled")
//						continue
//					} else {
//						bs.AfterComplete("canceled-closed(SL)")
//						continue
//					}
//				}
//			} else if execState.PosSide == consts.Close {
//				if execState.InstType == insttype.Margin {
//					// track 使命结束
//					bs.marginTrack = nil
//				}
//				if execState.InstType == insttype.Future {
//					// track 使命结束
//					bs.futureTrack = nil
//				}
//
//				otherInstType := util.Select(execState.InstType == insttype.Margin, insttype.Future, insttype.Margin)
//				otherOpen := bs.okeService.GetOpenOrder(otherInstType)
//				//if otherOpen != nil && otherOpen.State == orderstate.Live {
//				//	bs.okeService.CancelOrder(otherInstType)
//				//	continue
//				//}
//				if otherOpen != nil && otherOpen.State == orderstate.Failed {
//					bs.AfterComplete(util.Select(otherOpen.OrderType == insttype.Margin, "failed-closed", "closed-failed"))
//					continue
//				}
//				//将close signal -> closePartFilled
//				atomic.AddInt32(&bs.strategyState, 1)
//
//			}
//			log.Info("监听上报的订单更新,strategyState=%v", bs.strategyState)
//		}
//	}
//}
