package backend

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ws-quant/cex/models"
	"ws-quant/cex/service"
	"ws-quant/common/bean"
	"ws-quant/common/insttype"
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

	tickerChan      chan bean.TickerBean //负责监听接收数据
	trackTickerChan chan bean.TickerBean //负责 sl tp 跟进

	orderStateChan chan bean.ExecState
	trackBeanChan  chan bean.TrackBean

	OkBitMap   map[string]*OkBitTicker
	okeService *service.Service
	stopChan   chan struct{}

	//curMax             float64
	maxDiffMarginFuture float64
	db                  *xorm.Engine
	engine              *gin.Engine
	execStates          []string //逐渐淘汰复杂的 strategy state
	executingSymbol     string   //如eos

	triggerState int32 //0 初始化, 1 trigger open pos, 2 trigger close,
	marginTrack  *bean.TrackBean
	futureTrack  *bean.TrackBean
}

func New() server.Server {
	bs := &backendServer{}

	bs.initMapAndChan()
	return bs
}

func (bs *backendServer) QuantRun() error {
	// 连db
	bs.dbClient()
	bs.okeService = service.New(bs.tickerChan, bs.orderStateChan, bs.trackBeanChan, bs.db)
	go func() {
		defer e.Recover()()
		bs.okeService.Run()
	}()
	// listen ticker
	for i := 0; i < 20; i++ {
		go func() {
			defer e.Recover()()
			bs.listenAndExec()
		}()
	}
	//todo 待将来打开
	//go func() {
	//	defer e.Recover()()
	//	bs.listenOrderState()
	//}()
	//
	//go func() {
	//	defer e.Recover()()
	//	bs.listenTrackBeanOpenFilledAndClose()
	//}()
	//
	//go func() {
	//	defer e.Recover()()
	//	bs.listenAndExecStTp()
	//}()
	// schedule 一些定时任务
	//bs.scheduleJobs()
	bs.PostInit()
	// router
	bs.router()
	feishu.Send("program start!")
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
func (bs *backendServer) shouldClose(t *OkBitTicker) bool {
	return t.AskBit >= t.BidOk && t.AskBit <= t.AskOk
}

func (bs *backendServer) realDiff(t *OkBitTicker) (signal int, realDiffPct float64) {
	// 从三个价格中判断是否可以 open position
	signal = 0
	realDiffPct = 0
	if t.BidOk > t.AskBit {
		realDiffPct = (t.BidOk/t.AskBit - 1) * 100
		if realDiffPct >= bs.config.StrategyOpenThreshold {
			signal = 1
		}
	} else if t.BidBit > t.AskOk {
		realDiffPct = (t.BidBit/t.AskOk - 1) * 100
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

func (bs *backendServer) getTrackBean(instType string) *bean.TrackBean {
	if instType == insttype.Margin {
		return bs.marginTrack
	} else if instType == insttype.Future {
		return bs.futureTrack
	}
	feishu.Send("invalid instType")
	return nil

}
