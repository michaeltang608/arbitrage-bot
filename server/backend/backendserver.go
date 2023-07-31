package backend

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strings"
	"sync/atomic"
	"time"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/cex/oke"
	"ws-quant/common/consts"
	"ws-quant/common/symb"
	"ws-quant/core"
	"ws-quant/models/bean"
	"ws-quant/pkg/db"
	"ws-quant/pkg/e"
	"ws-quant/pkg/feishu"
	logger "ws-quant/pkg/log"
	"ws-quant/pkg/mapper"
	"ws-quant/pkg/middleware"
	"ws-quant/pkg/router"
	"ws-quant/pkg/util"
	"ws-quant/pkg/util/numutil"
	"ws-quant/server"
	"xorm.io/xorm"
)

var log = logger.NewLog("backend")

type MarginFutureTicker struct {
	Symbol    string
	AskMargin float64
	BidMargin float64
	AskFuture float64
	BidFuture float64
}

type backendServer struct {
	config *models.Config

	tickerChan chan bean.TickerBean //负责监听接收数据
	//TickerDataMap     map[string]map[string]*bean.Ticker //存储 symbol-cex-prices，
	OkMarginFutureMap map[string]*MarginFutureTicker
	CalChan           chan SignalCalBean //负责分析数据
	//cexServiceMap     map[string]cex.Service
	okeService *oke.Service
	stopChan   chan struct{}

	//curMax             float64
	maxDiffMarginFuture float64
	db                  *xorm.Engine
	engine              *gin.Engine
	executingSymbol     string //如eos

	//strategyExecOrdersChan chan *models.Orders //cex => server 两个执行的交易所向server上传执行结果的chan
	strategyState int32 //0: 默认, 1 触发开仓策略，2 某cex完成open单，3 both cex完成open单；11 触发平仓；12 某cex完成close; 13 both cex 完成cex, 然后转0
	execStateChan chan bean.ExecState

	okOpenBuyMarketFunc func()
}

type Oppor struct {
	Symbol   string  //交易对，大写，如 EOS
	OpenDiff float64 // 如 1.0
	MaxDiff  float64 // 真实中最大的max diff
	MaxPrice float64
	MinPrice float64
	MaxCex   string
	MinCex   string
}

func New() server.Server {
	bs := &backendServer{}

	bs.initMap()
	return bs
}

func (bs *backendServer) QuantRun() error {
	// 连db
	bs.dbClient()
	bs.okeService = oke.New(bs.tickerChan, bs.execStateChan, bs.db)
	go func() {
		defer e.Recover()()
		bs.okeService.Run()
	}()
	// listen ticker
	for i := 0; i < 100; i++ {
		go func() {
			defer e.Recover()()
			bs.listenAndCal()
		}()
	}

	go func() {
		defer e.Recover()()
		bs.listenState()
	}()
	// schedule 一些定时任务
	bs.scheduleJobs()
	bs.PostInit()
	// router
	bs.router()
	feishu.Send("program start successfully")
	err := bs.engine.Run(":8083")
	return err
}

func (bs *backendServer) router() {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())
	r.ForwardedByClientIP = true
	router.AddRouteGroup(r, bs.tradeRouteGroup())
	router.AddRouteGroup(r, bs.configRouteGroup())
	router.AddRouteGroup(r, bs.testRouteGroup())
	bs.engine = r
}

// 负责监听和搜集数据
func (bs *backendServer) listenAndCal() {
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
			//todo 策略执行 open position
			if openSignal != 0 {
				if atomic.CompareAndSwapInt32(&bs.strategyState, 0, 1) {
					bs.execOpenLimit(openSignal, ticker)
					//log.Info("执行后的延迟是:%d毫秒", time.Now().UnixMilli()-signalCalBean.Ts0)
				}
			}

			// close position
			//if bs.strategyState == int32(StateOpenFilledAll) {
			//	if strings.ToUpper(symbol) == strings.ToUpper(bs.executingSymbol) {
			//		if bs.shouldClose(prcList) {
			//			if atomic.CompareAndSwapInt32(&bs.strategyState, int32(StateOpenFilledAll), int32(StateCloseSignalled)) {
			//				bs.execCloseMarket(prcList, symbol)
			//
			//			}
			//		}
			//	}
			//}

			if bs.config.LogTicker == LogOke && tickerBean.CexName == cex.OKE {
				log.Info("收到ticker数据，%+v", tickerBean)
			}
		}
	}
}

// 0 不开，1 max ku sell, -1 min ku buy
func (bs *backendServer) shouldClose(prices []float64) bool {
	kuAsk := prices[0]
	kuBid := prices[1]
	oke_ := prices[2]
	return oke_ >= kuBid && oke_ <= kuAsk
}

func (bs *backendServer) realDiff(t *MarginFutureTicker) (signal int, realDiffPct float64) {
	//从三个价格中判断是否可以 open position

	signal = 0
	realDiffPct = 0
	if t.BidMargin > t.AskFuture {
		realDiffPct = (t.BidMargin/t.AskFuture - 1) * 100
		if realDiffPct >= bs.config.StrategyOpenThreshold {
			signal = 1
		}
	} else if t.BidFuture > t.AskMargin {
		realDiffPct = (t.BidFuture/t.AskMargin - 1) * 100
		if realDiffPct >= bs.config.StrategyOpenThreshold {
			signal = -1
		}
	}
	return
}

// 监听并流转 策略状态
func (bs *backendServer) listenState() {
	for {
		select {
		case execState := <-bs.execStateChan:

			msg := fmt.Sprintf("收到来自%v的state: %v", execState.CexName, execState.PosSide)
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
						feishu.Send("strategyState已经是3，但是oppor为空")
					}
				}
			} else if execState.PosSide == consts.Close {
				if bs.strategyState != int32(StateCloseSignalled) && bs.strategyState != int32(StateCloseFilledPart) {
					msg := fmt.Sprintf("strategyState是%v, 但收到了closeFilled", bs.strategyState)
					log.Error(msg)
					feishu.Send(msg)
				}
				r := atomic.AddInt32(&bs.strategyState, 1)
				if r == int32(StateCloseFilledAll) {
					log.Info("策略全部完成")
					feishu.Send("strategy all done!, wait for manual reset")
					//等待人工重置，否则容易再次出发，大概率机会没了，无法双向交易
					bs.strategyState = -1
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

func (bs *backendServer) dbClient() {
	bs.db = db.New(&db.Config{
		DriverName: "mysql",
		Ip:         "localhost",
		Port:       3317,
		Usr:        "root",
		Pwd:        "58c974081d67",
		Schema:     "crypto",
	})
	_ = bs.db.Sync([]interface{}{models.Account{}, models.Orders{}, models.Config{}, models.Oppor{}}...)

	ele := &models.Config{ID: 1}
	has := mapper.Get(bs.db, ele)
	if !has {
		eleNew := &models.Config{
			TickerTimeout:          600,
			LogThreshold:           1.0,
			TradeAmt:               20.0,
			LogSymbol:              "",
			LogTicker:              0,
			StrategyOpenThreshold:  1.2,
			StrategyCloseThreshold: 0.1,
		}
		_ = mapper.Insert(bs.db, eleNew)
		bs.config = ele
	} else {
		bs.config = ele
	}

}

func (bs *backendServer) QuantClose() error {
	// 准备关闭资源
	feishu.Send("程序准备退出, 准备重启")
	bs.okeService.Close()
	// 通知main函数 退出
	bs.stopChan <- struct{}{}
	return nil
}

func (bs *backendServer) initMap() {
	// 初始化要监控的 ticker
	// init okex margin-future map，第一个idx存储
	bs.OkMarginFutureMap = make(map[string]*MarginFutureTicker)
	for _, sym := range symb.GetAllOkFuture() {
		bs.OkMarginFutureMap[sym] = &MarginFutureTicker{Symbol: sym}
	}

	// 初始化 chan
	bs.tickerChan = make(chan bean.TickerBean, 200)
	bs.CalChan = make(chan SignalCalBean, 200)
	bs.execStateChan = make(chan bean.ExecState)
}

func (bs *backendServer) PostInit() {
	//更新 strategyState， prepare close symb
	go func() {
		defer e.Recover()()
		time.Sleep(time.Second * 5)
		closeOrders := make([]*models.Orders, 0)
		openOrders := make([]*models.Orders, 0)
		//
		//for _, s := range bs.cexServiceMap {
		//	if s.GetOpenOrder() != nil {
		//		log.Info("openOrder:%v", s.GetOpenOrder())
		//		openOrders = append(openOrders, s.GetOpenOrder())
		//		bs.executingSymbol = strings.ToLower(strings.Split(s.GetOpenOrder().InstId, "-")[0])
		//	}
		//	if s.GetCloseOrder() != nil {
		//		log.Info("closeOrder:%v", s.GetCloseOrder())
		//		closeOrders = append(closeOrders, s.GetCloseOrder())
		//	}
		//}
		if len(closeOrders) > 0 {
			bs.strategyState = int32(StateCloseSignalled)
			for _, closeOrder := range closeOrders {
				if closeOrder.State == core.FILLED.State() {
					bs.strategyState = int32(StateCloseFilledPart)
				}
			}

		} else if len(openOrders) > 0 {
			bs.strategyState = int32(StateOpenSignalled)
			for _, openOrder := range openOrders {
				if openOrder.State == core.FILLED.State() {
					if bs.strategyState == int32(StateOpenSignalled) {
						bs.strategyState = int32(StateOpenFilledPart)
					} else if bs.strategyState == int32(StateOpenFilledPart) {
						bs.strategyState = int32(StateOpenFilledAll)
					}
				}
			}
		}
		log.Info("程序启动，strategyState=%v", bs.strategyState)
	}()

}

/**
1 这种方式经常拿不到最好的价格，
2 而且考虑到kucoin borrow经常失败，开单的话要先执行kucoin, 成功后再执行oke 这样就延误了战机
*/
//func (bs *backendServer) execOpenMarket(openSignal int, prcList []float64, symbol string) {
//	feishu.Send(fmt.Sprintf("trigger&exec open market strategy, symb=%sA, sig=%v, prcs: %v, %v, %v, %v", symbol, openSignal, prcList[0], prcList[1], prcList[2], prcList[3]))
//	bs.executingSymbol = symbol
//	log.Info("signalOpen, strategyState=%v", bs.strategyState)
//	if openSignal == 1 {
//		// sell kucoin first, then buy oke upon filled signal
//		for cexName, cexService := range bs.cexServiceMap {
//			go func(cexName string, service cex.Service) {
//				size := util.NumTrunc(bs.config.TradeAmt / prcList[0])
//				if cexName == cex.KUCOIN {
//					side := "sell"
//					log.Info("ku准备开仓, side=%v, symbol=%v,  size=%v\n", side, symbol, size)
//					msg := service.OpenPosMarket(symbol, size, side)
//					log.Info("ku开仓结果是:" + msg)
//				} else if cexName == cex.OKE {
//					/**
//					sz
//					交易数量，表示要购买或者出售的数量。
//					当币币/币币杠杆以限价买入和卖出时，指交易货币数量。
//					* 当币币/币币杠杆以市价买入时，指计价货币的数量。*
//					当币币/币币杠杆以市价卖出时，指交易货币的数量。
//					*/
//					// 以市价买入时，指计价货币的数量
//					side := "buy"
//					size = util.NumTrunc(bs.config.TradeAmt)
//					log.Info("okeLog 准备一步开仓, side=%v, symbol=%v, size=%v\n", side, symbol, size)
//					bs.okOpenBuyMarketFunc = func() {
//						msg := service.OpenPosMarket(symbol, size, side)
//						log.Info("okeLog 开仓结果是:" + msg)
//					}
//				}
//
//			}(cexName, cexService)
//		}
//		return
//	}
//
//	//concurrent exec
//	for cexName, cexService := range bs.cexServiceMap {
//		go func(cexName string, service cex.Service) {
//			size := util.NumTrunc(bs.config.TradeAmt / prcList[0])
//			if cexName == cex.KUCOIN {
//				side := "buy"
//				if openSignal > 0 {
//					side = "sell"
//				}
//				log.Info("ku准备开仓, side=%v, symbol=%v,  size=%v\n", side, symbol, size)
//				msg := service.OpenPosMarket(symbol, size, side)
//				log.Info("ku开仓结果是:" + msg)
//			} else if cexName == cex.OKE {
//				/**
//				sz
//				交易数量，表示要购买或者出售的数量。
//				当币币/币币杠杆以限价买入和卖出时，指交易货币数量。
//				* 当币币/币币杠杆以市价买入时，指计价货币的数量。*
//				当币币/币币杠杆以市价卖出时，指交易货币的数量。
//				*/
//
//				side := "sell"
//				if openSignal > 0 {
//					// 以市价买入时，指计价货币的数量
//					side = "buy"
//					size = util.NumTrunc(bs.config.TradeAmt)
//				}
//				log.Info("okeLog 准备开仓, side=%v, symbol=%v, size=%v\n", side, symbol, size)
//				msg := service.OpenPosMarket(symbol, size, side)
//				log.Info("okeLog 开仓结果是:" + msg)
//			}
//
//		}(cexName, cexService)
//	}
//	feishu.Send("strategy open triggered")
//}
func (bs *backendServer) execOpenLimit(openSignal int, t *MarginFutureTicker) {
	msg := fmt.Sprintf("trigger&exec open limit strategy,ticker= %+v", t)
	feishu.Send(msg)
	log.Info(msg)
	bs.executingSymbol = t.Symbol
	log.Info("signalOpen, strategyState=%v", bs.strategyState)

	// 先处理 margin, 再处理 future
	// 计算size
	symbolPrc := t.AskFuture
	numPerUnit := symb.GetFutureLot(t.Symbol)
	if numPerUnit == "" {
		log.Error("未找到该future 的unitNum, %v", t.Symbol)
		feishu.Send("未找到该future 的unitNum")
		return
	}

	tradeAmt, futureSize := calFutureSizeAndTradeAmt(bs.config.TradeAmt, symbolPrc, numutil.Parse(numPerUnit))
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
		price := util.AdjustPrice(priceF, side)
		log.Info("ok margin prepare open pos, side=%v, symbol=%v, price=%v, size=%v\n", side, t.Symbol, price, size)
		msg := bs.okeService.OpenPosLimit(t.Symbol, price, size, side)
		log.Info("ok margin-open pos result:" + msg)
	}(tradeAmt)
	go func(size int) {
		// future
		side := "sell"
		priceF := t.BidFuture
		if openSignal > 0 {
			side = "buy"
			priceF = t.AskFuture
		}
		price := util.AdjustPrice(priceF, side)
		log.Info("prepare to open pos, side=%v, symbol=%v, price=%v, size=%v\n", side, t.Symbol, price, size)
		msg := bs.okeService.OpenPosLimit(t.Symbol, price, numutil.FormatInt(size), side)
		log.Info("ok future open-pos result:" + msg)

	}(futureSize)
	feishu.Send("strategy open triggered")
}

func calFutureSizeAndTradeAmt(planTradeAmt, symbolPrc, numPerUnit float64) (actualTradeAmt float64, futureSize int) {
	//1 cal can buy sym num
	symNum := planTradeAmt / symbolPrc
	unitNum := symNum / numPerUnit
	futureSize = int(unitNum)
	if futureSize == 0 {
		futureSize = 1
	}
	//计算 actualAmtT
	actualSymNum := numPerUnit * float64(futureSize)
	actualTradeAmt = actualSymNum * symbolPrc
	return actualTradeAmt, futureSize
}

//func (bs *backendServer) execCloseMarket(prcList []float64, symbol string) {
//	feishu.Send(fmt.Sprintf("trigger&exec close market strategy, symb=%sA, prcs: %v, %v, %v,%v", symbol, prcList[0], prcList[1], prcList[2], prcList[3]))
//	log.Info("signal close, strategyState=%v", bs.strategyState)
//	for cexName, service_ := range bs.cexServiceMap {
//		go func(cexName string, service cex.Service, prcList []float64) {
//			if cexName == cex.KUCOIN {
//				log.Info("kucoinLog 执行关仓， market")
//				msg := service.ClosePosMarket(prcList[0], prcList[1])
//				log.Info("kucoinLog 关仓结果是:" + msg)
//			} else if cexName == cex.OKE {
//				log.Info("okeLog 执行关仓， market")
//				msg := service.ClosePosMarket(prcList[2], prcList[3])
//				log.Info("okeLog 关仓结果是:" + msg)
//			}
//
//		}(cexName, service_, prcList)
//	}
//}

// seek arbitrage oppor between oke margin and future(perpetual)
