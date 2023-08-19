package backend

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
	"ws-quant/cex/models"
	"ws-quant/cex/oke"
	"ws-quant/common/bean"
	"ws-quant/common/consts"
	"ws-quant/common/insttype"
	"ws-quant/common/orderstate"
	"ws-quant/common/symb"
	"ws-quant/pkg/db"
	"ws-quant/pkg/e"
	"ws-quant/pkg/feishu"
	logger "ws-quant/pkg/log"
	"ws-quant/pkg/mapper"
	"ws-quant/pkg/middleware"
	"ws-quant/pkg/router"
	"ws-quant/server"
	"xorm.io/xorm"
)

var log = logger.NewLog("backend")

type backendServer struct {
	config *models.Config

	tickerChan        chan bean.TickerBean //负责监听接收数据
	OkMarginFutureMap map[string]*MarginFutureTicker
	CalChan           chan SignalCalBean //负责分析数据
	okeService        *oke.Service
	stopChan          chan struct{}

	//curMax             float64
	maxDiffMarginFuture float64
	db                  *xorm.Engine
	engine              *gin.Engine
	executingSymbol     string //如eos

	strategyState int32 //0: 默认, 1 触发开仓策略，2 某cex完成open单，3 both cex完成open单；11 触发平仓；12 某cex完成close; 13 both cex 完成cex, 然后转0
	execStateChan chan bean.ExecState

	trackBeanChan chan bean.TrackBean
	marginTrack   *bean.TrackBean
	futureTrack   *bean.TrackBean
}

func New() server.Server {
	bs := &backendServer{}

	bs.initMap()
	return bs
}

func (bs *backendServer) QuantRun() error {
	// 连db
	bs.dbClient()
	bs.okeService = oke.New(bs.tickerChan, bs.execStateChan, bs.trackBeanChan, bs.db)
	go func() {
		defer e.Recover()()
		bs.okeService.Run()
	}()
	// listen ticker
	for i := 0; i < 100; i++ {
		go func() {
			defer e.Recover()()
			bs.listenAndExec()
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
	//feishu.Send("program start successfully")
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

// 0 不开，1 max ku sell, -1 min ku buy
func (bs *backendServer) shouldClose(t *MarginFutureTicker) bool {
	return t.AskFuture >= t.BidMargin && t.AskFuture <= t.AskMargin
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

func (bs *backendServer) dbClient() {
	bs.db = db.New(&db.Config{
		DriverName: "mysql",
		Ip:         "localhost",
		Port:       3317,
		Usr:        "root",
		Pwd:        "58c974081d67",
		Schema:     "crypto",
	})
	_ = bs.db.Sync([]interface{}{models.AccountOke{}, models.Orders{}, models.Config{}, models.Oppor{}}...)

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
	bs.trackBeanChan = make(chan bean.TrackBean, 100)
	bs.execStateChan = make(chan bean.ExecState)
}

func (bs *backendServer) PostInit() {
	go func() {
		defer e.Recover()()
		bs.marginTrack = nil
		time.Sleep(time.Second * 5)
		openMarginOrder := bs.okeService.GetOpenOrder(insttype.Margin)
		openFutureOrder := bs.okeService.GetOpenOrder(insttype.Future)

		closeMarginOrder := bs.okeService.GetCloseOrder(insttype.Margin)
		closeFutureOrder := bs.okeService.GetCloseOrder(insttype.Future)

		if openMarginOrder != nil {
			bs.marginTrack = &bean.TrackBean{}
			if closeMarginOrder != nil {
				bs.marginTrack.State = orderstate.Filled
			} else {
				bs.marginTrack.State = openMarginOrder.State
				bs.marginTrack.Side = openMarginOrder.Side
				bs.marginTrack.InstType = openMarginOrder.OrderType
			}
		}
		if openFutureOrder != nil {
			bs.futureTrack = &bean.TrackBean{}
			if closeFutureOrder != nil {
				bs.futureTrack.State = orderstate.Filled
			} else {
				bs.futureTrack.State = openFutureOrder.State
				bs.futureTrack.Side = openFutureOrder.Side
				bs.futureTrack.InstType = openFutureOrder.OrderType
			}
		}

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

func (bs *backendServer) calFutureSizeAndTradeAmt(symbol string, symbolPrc, numPerUnit float64) (actualTradeAmt float64, futureSize int) {
	//1 cal can buy sym num
	symNum := bs.config.TradeAmt / symbolPrc
	unitNum := symNum / numPerUnit
	futureSize = int(unitNum)
	if futureSize == 0 {
		futureSize = 1
		// 这个时候可能会超过 amtMax, 注意判断
		actualTradeAmt = float64(futureSize) * numPerUnit * symbolPrc
		if actualTradeAmt > bs.config.TradeAmtMax {
			msg := fmt.Sprintf("actual amt exceed max, %v, %v, %s", actualTradeAmt, bs.config.TradeAmtMax, symbol)
			feishu.Send(msg)
			log.Error(msg)
			return 0, 0
		}
	}
	//计算 actualAmtT
	actualSymNum := numPerUnit * float64(futureSize)
	actualTradeAmt = actualSymNum * symbolPrc
	return actualTradeAmt, futureSize
}
