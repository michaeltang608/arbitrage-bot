package backend

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strings"
	"sync/atomic"
	"time"
	"ws-quant/cex"
	"ws-quant/cex/kucoin"
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
	"ws-quant/server"
	"xorm.io/xorm"
)

var log = logger.NewLog("backend")

type backendServer struct {
	config *models.Config

	tickerChan    chan bean.TickerBean               //负责监听接收数据
	TickerDataMap map[string]map[string]*bean.Ticker //存储 symbol-cex-prices，
	CalChan       chan SignalCalBean                 //负责分析数据
	cexServiceMap map[string]cex.Service
	stopChan      chan struct{}

	curMax          float64
	db              *xorm.Engine
	engine          *gin.Engine
	executingSymbol string //如eos

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
	// create cex service
	serviceList := make([]cex.Service, 0)
	serviceList = append(serviceList, oke.New(bs.tickerChan, bs.execStateChan, bs.db))
	serviceList = append(serviceList, kucoin.New(bs.tickerChan, bs.execStateChan, bs.db))
	bs.cexServiceMap = make(map[string]cex.Service)
	for _, service := range serviceList {
		bs.cexServiceMap[service.GetCexName()] = service
		go func(s cex.Service) {
			defer e.Recover()()
			s.Run()
		}(service)
	}
	// listen ticker
	for i := 0; i < 100; i++ {
		go func() {
			defer e.Recover()()
			bs.listenData()
		}()
	}

	// calculate and trigger trade
	for i := 0; i < 100; i++ {
		go func() {
			defer e.Recover()()
			bs.listenAndCalPctDiff()
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
func (bs *backendServer) listenData() {
	for {
		select {
		case tickerBean := <-bs.tickerChan:
			//接收数据并填装， 同时预警长时间未更新的 ticker 数据, 和 计算价格之差
			ticker := bs.TickerDataMap[tickerBean.SymbolName][tickerBean.CexName]
			ticker.PriceBestAsk = tickerBean.PriceBestAsk
			ticker.Price = tickerBean.Price
			ticker.PriceBestBid = tickerBean.PriceBestBid
			ticker.LastTime = ticker.CurTime
			ticker.CurTime = time.Now().Unix()

			//组装好数据，赶紧先传到下一层
			bs.CalChan <- SignalCalBean{
				Symbol: tickerBean.SymbolName,
				Ts0:    tickerBean.Ts0,
				Ts1:    time.Now().UnixMilli(),
			}

			if bs.config.LogTicker == 1 && tickerBean.CexName == cex.KUCOIN {
				log.Info("收到ticker数据，%+v", tickerBean)
			}
			if bs.config.LogTicker == 2 && tickerBean.CexName == cex.OKE {
				log.Info("收到ticker数据，%+v", tickerBean)
			}
			if ticker.LastTime > 0 && ticker.CurTime > (ticker.LastTime+bs.config.TickerTimeout) {
				msg := fmt.Sprintf("超时未推送的数据,curTime=%d, lastTime=%d, cex=%s\n", ticker.CurTime, ticker.LastTime, tickerBean.CexName+"x")
				log.Info(msg)
				feishu.Send(msg)
			}

		}
	}
}

// cal pct diff among different CEXs, 可由定时器或价格更新的 chan 触发
func (bs *backendServer) listenAndCalPctDiff() {
	for {
		select {
		case signalCalBean := <-bs.CalChan:
			if bs.config.LogTicker == 3 {
				now := time.Now().UnixMilli()
				delay := now - signalCalBean.Ts0
				if delay > 0 {
					//log.Info("处理时间间隔%d毫秒", delay)
					if delay >= 3 {
						log.Info("具体处理时间间隔ts0=%d, ts1=%d, now=%d\n", signalCalBean.Ts0, signalCalBean.Ts1, now)
					}
				}
			}

			symbol := signalCalBean.Symbol
			m, ok := bs.TickerDataMap[symbol]
			if !ok {
				errMsg := "收到价格通知，但是没找到初始化数据"
				log.Error(errMsg)
				feishu.Send(errMsg)
				continue
			}
			//目前只考虑两家cex: kucoin& ok
			kucoinTicker := m[cex.KUCOIN]
			okeTicker := m[cex.OKE]
			if kucoinTicker.Price <= 0 || okeTicker.Price <= 0 {
				continue
			}
			prcList := make([]float64, 0)
			prcList = append(prcList, kucoinTicker.PriceBestAsk)
			prcList = append(prcList, kucoinTicker.PriceBestBid)
			prcList = append(prcList, okeTicker.PriceBestAsk)
			prcList = append(prcList, okeTicker.PriceBestBid)

			openSignal, realDiff := bs.realDiff(prcList)
			if bs.curMax < realDiff {
				bs.curMax = realDiff
				log.Info("curMax is:%v\n", bs.curMax)
			}

			if bs.strategyState == int32(StateOpenFilledAll) {
				if strings.ToUpper(symbol) == strings.ToUpper(bs.executingSymbol) {
					if bs.shouldClose(prcList) {
						if atomic.CompareAndSwapInt32(&bs.strategyState, int32(StateOpenFilledAll), int32(StateCloseSignalled)) {
							bs.execCloseMarket(prcList, symbol)

						}
					}
				}
			}
			if openSignal != 0 {
				log.Info("达到条件延迟是:%d毫秒, symbol=%v", time.Now().UnixMilli()-signalCalBean.Ts0, symbol)
				if atomic.CompareAndSwapInt32(&bs.strategyState, 0, 1) {
					bs.execOpenMarket(openSignal, prcList, symbol)
					log.Info("执行后的延迟是:%d毫秒", time.Now().UnixMilli()-signalCalBean.Ts0)
				}
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

func (bs *backendServer) realDiff(prices []float64) (signal int, realDiffPct float64) {
	//从三个价格中判断是否可以 open position
	kuMax := prices[0]
	kuMin := prices[1]
	okeMax := prices[2]
	okeMin := prices[3]
	signal = 0
	realDiffPct = 0
	if kuMin > okeMax {
		realDiffPct = (kuMin/okeMax - 1) * 100
		if realDiffPct >= bs.config.StrategyOpenThreshold {
			signal = 1
		}
	} else if okeMin > kuMax {
		realDiffPct = (okeMin/kuMax - 1) * 100
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
			if execState.Side == consts.Sell && execState.PosSide == consts.Open && execState.CexName == cex.KUCOIN {
				//ku sell suc, 赶紧通知 ok place open buy market order
				bs.okOpenBuyMarketFunc()
			}

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
					feishu.Send("strategy all done!!!")
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

func (bs *backendServer) dbClient() {
	bs.db = db.New(&db.Config{
		DriverName: "mysql",
		Ip:         "localhost",
		Port:       3317,
		Usr:        "root",
		Pwd:        "",
		Schema:     "crypto",
	})
	_ = bs.db.Sync([]interface{}{models.Account{}, models.Orders{}, models.Config{}}...)

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

// 返回自我描述的 string
func (bs *backendServer) desc() string {
	result := "\n"
	for symbol, m := range bs.TickerDataMap {
		for cexName, data := range m {
			symbolInfo := fmt.Sprintf("%s的%s价格信息%v\n", cexName, symbol, *data)
			result = result + symbolInfo
		}
	}
	return result
}

func (bs *backendServer) QuantClose() error {
	// 准备关闭资源
	feishu.Send("程序准备退出, 准备重启")
	for _, s := range bs.cexServiceMap {
		s.Close()
	}
	// 通知main函数 退出
	bs.stopChan <- struct{}{}
	return nil
}

func (bs *backendServer) initMap() {
	// 初始化要监控的 ticker
	bs.TickerDataMap = make(map[string]map[string]*bean.Ticker, 0)
	for _, sym := range symb.GetAllSymb() {
		bs.TickerDataMap[sym] = make(map[string]*bean.Ticker)

		for _, cexName := range cex.GetAllCex() {
			bs.TickerDataMap[sym][cexName] = &bean.Ticker{}
		}
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

		for _, s := range bs.cexServiceMap {
			if s.GetOpenOrder() != nil {
				log.Info("openOrder:%v", s.GetOpenOrder())
				openOrders = append(openOrders, s.GetOpenOrder())
				bs.executingSymbol = strings.ToLower(strings.Split(s.GetOpenOrder().InstId, "-")[0])
			}
			if s.GetCloseOrder() != nil {
				log.Info("closeOrder:%v", s.GetCloseOrder())
				closeOrders = append(closeOrders, s.GetCloseOrder())
			}
		}
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

// 这种方式拿不到最好的价格
func (bs *backendServer) execOpenMarket(openSignal int, prcList []float64, symbol string) {
	feishu.Send(fmt.Sprintf("trigger&exec open market strategy, symb=%sA, sig=%v, prcs: %v, %v, %v, %v", symbol, openSignal, prcList[0], prcList[1], prcList[2], prcList[3]))
	bs.executingSymbol = symbol
	log.Info("signalOpen, strategyState=%v", bs.strategyState)
	if openSignal == 1 {
		// sell kucoin first, then buy oke upon filled signal
		for cexName, cexService := range bs.cexServiceMap {
			go func(cexName string, service cex.Service) {
				size := util.NumTrunc(bs.config.TradeAmt / prcList[0])
				if cexName == cex.KUCOIN {
					side := "sell"
					log.Info("ku准备开仓, side=%v, symbol=%v,  size=%v\n", side, symbol, size)
					msg := service.OpenPosMarket(symbol, size, side)
					log.Info("ku开仓结果是:" + msg)
				} else if cexName == cex.OKE {
					/**
					sz
					交易数量，表示要购买或者出售的数量。
					当币币/币币杠杆以限价买入和卖出时，指交易货币数量。
					* 当币币/币币杠杆以市价买入时，指计价货币的数量。*
					当币币/币币杠杆以市价卖出时，指交易货币的数量。
					*/
					// 以市价买入时，指计价货币的数量
					side := "buy"
					size = util.NumTrunc(bs.config.TradeAmt)
					log.Info("okeLog 准备一步开仓, side=%v, symbol=%v, size=%v\n", side, symbol, size)
					bs.okOpenBuyMarketFunc = func() {
						msg := service.OpenPosMarket(symbol, size, side)
						log.Info("okeLog 开仓结果是:" + msg)
					}
				}

			}(cexName, cexService)
		}
		return
	}

	//concurrent exec
	for cexName, cexService := range bs.cexServiceMap {
		go func(cexName string, service cex.Service) {
			size := util.NumTrunc(bs.config.TradeAmt / prcList[0])
			if cexName == cex.KUCOIN {
				side := "buy"
				if openSignal > 0 {
					side = "sell"
				}
				log.Info("ku准备开仓, side=%v, symbol=%v,  size=%v\n", side, symbol, size)
				msg := service.OpenPosMarket(symbol, size, side)
				log.Info("ku开仓结果是:" + msg)
			} else if cexName == cex.OKE {
				/**
				sz
				交易数量，表示要购买或者出售的数量。
				当币币/币币杠杆以限价买入和卖出时，指交易货币数量。
				* 当币币/币币杠杆以市价买入时，指计价货币的数量。*
				当币币/币币杠杆以市价卖出时，指交易货币的数量。
				*/

				side := "sell"
				if openSignal > 0 {
					// 以市价买入时，指计价货币的数量
					side = "buy"
					size = util.NumTrunc(bs.config.TradeAmt)
				}
				log.Info("okeLog 准备开仓, side=%v, symbol=%v, size=%v\n", side, symbol, size)
				msg := service.OpenPosMarket(symbol, size, side)
				log.Info("okeLog 开仓结果是:" + msg)
			}

		}(cexName, cexService)
	}
	feishu.Send("strategy open triggered")
}
func (bs *backendServer) execOpenLimit(openSignal int, prcList []float64, symbol string) {
	feishu.Send(fmt.Sprintf("trigger&exec open limit strategy, symb=%sA, sig=%v, prcs: %v, %v, %v, %v", symbol, openSignal, prcList[0], prcList[1], prcList[2], prcList[3]))
	bs.executingSymbol = symbol
	log.Info("signalOpen, strategyState=%v", bs.strategyState)

	for cexName, cexService := range bs.cexServiceMap {

		go func(cexName string, service cex.Service) {
			priceF := 0.0
			size := util.NumTrunc(bs.config.TradeAmt / prcList[0])
			if cexName == cex.KUCOIN {
				side := "buy"
				priceF = prcList[0]
				if openSignal > 0 {
					side = "sell"
					priceF = prcList[1]
				}
				price := util.AdjustPrice(priceF, side)
				log.Info("ku limit 准备开仓, side=%v, symbol=%v, price=%v, size=%v\n", side, symbol, price, size)
				msg := service.OpenPosLimit(symbol, price, size, side)
				log.Info("ku limit开仓结果是:" + msg)

			} else if cexName == cex.OKE {

				side := "sell"
				priceF = prcList[3]
				if openSignal > 0 {
					// 以市价买入时，指计价货币的数量
					side = "buy"
					priceF = prcList[2]
				}
				price := util.AdjustPrice(priceF, side)
				log.Info("准备开仓, side=%v, symbol=%v, price=%v, size=%v\n", side, symbol, price, size)
				msg := service.OpenPosLimit(symbol, price, size, side)
				log.Info("开仓结果是:" + msg)
			}

		}(cexName, cexService)
	}
	feishu.Send("strategy open triggered")
}

func (bs *backendServer) execCloseMarket(prcList []float64, symbol string) {
	feishu.Send(fmt.Sprintf("trigger&exec close market strategy, symb=%sA, prcs: %v, %v, %v,%v", symbol, prcList[0], prcList[1], prcList[2], prcList[3]))
	log.Info("signal close, strategyState=%v", bs.strategyState)
	for cexName, service_ := range bs.cexServiceMap {
		go func(cexName string, service cex.Service, prcList []float64) {
			if cexName == cex.KUCOIN {
				log.Info("kucoinLog 执行关仓， market")
				msg := service.ClosePosMarket(prcList[0], prcList[1])
				log.Info("kucoinLog 关仓结果是:" + msg)
			} else if cexName == cex.OKE {
				log.Info("okeLog 执行关仓， market")
				msg := service.ClosePosMarket(prcList[2], prcList[3])
				log.Info("okeLog 关仓结果是:" + msg)
			}

		}(cexName, service_, prcList)
	}
}
