package oke

import (
	"fmt"
	"github.com/gorilla/websocket"
	"sync"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/common/consts"
	"ws-quant/models/bean"
	"ws-quant/pkg/e"
	logger "ws-quant/pkg/log"
	"xorm.io/xorm"
)

var (
	log = logger.NewLog("okeLog")
)

type Service struct {
	pubCon                *websocket.Conn
	prvCon                *websocket.Conn
	pubConLock            sync.Mutex
	prvConLock            sync.Mutex
	pubConLastConnectTime int
	prvConLastConnectTime uint64

	tickerChan    chan bean.TickerBean
	execStateChan chan bean.ExecState
	trackBeanChan chan bean.TrackBean
	//strategyExecOrdersChan chan *models.Orders

	usdtBal          float64 //余额
	openMarginOrder  *models.Orders
	openFutureOrder  *models.Orders
	closeMarginOrder *models.Orders
	closeFutureOrder *models.Orders
	db               *xorm.Engine
	debtMsgMap       map[string]string
}

func New(
	tickerChan chan bean.TickerBean,
	execStateChan chan bean.ExecState,
	trackBeanChan chan bean.TrackBean,
	db_ *xorm.Engine) *Service {
	s := &Service{
		tickerChan:    tickerChan,
		execStateChan: execStateChan,
		trackBeanChan: trackBeanChan,
		db:            db_,
		debtMsgMap:    make(map[string]string),
	}
	s.ReloadOrders()
	return s
}

func (s *Service) GetOpenOrder(orderType string) *models.Orders {
	if orderType == consts.Margin {
		return s.openMarginOrder
	}
	return s.openFutureOrder
}

func (s *Service) GetOrderStat() string {
	one, two, three, four := "nil", "nil", "nil", "nil"
	if s.openMarginOrder != nil {
		one = "live"
		if s.openMarginOrder.State == consts.Filled {
			one = "filled"
		}
	}
	if s.openFutureOrder != nil {
		two = "live"
		if s.openFutureOrder.State == consts.Filled {
			two = "filled"
		}
	}
	if s.closeMarginOrder != nil {
		three = "live"
		if s.closeMarginOrder.State == consts.Filled {
			three = "filled"
		}
	}
	if s.closeFutureOrder != nil {
		four = "live"
		if s.closeFutureOrder.State == consts.Filled {
			four = "filled"
		}
	}
	return fmt.Sprintf("%s-%s-%s-%s", one, two, three, four)
}

func (s *Service) GetCloseOrder(orderType string) *models.Orders {
	if orderType == consts.Margin {
		return s.closeMarginOrder
	}
	return s.closeFutureOrder
}

func (s *Service) ReloadOrders() {
	s.openMarginOrder = nil
	s.openFutureOrder = nil

	s.closeMarginOrder = nil
	s.closeFutureOrder = nil
	openOrders, closeOrders := cex.QueryOpenCloseOrders(s.db)
	for _, o := range openOrders {
		if o.OrderType == consts.Margin {
			s.openMarginOrder = o
		}
		if o.OrderType == consts.Future {
			s.openFutureOrder = o
		}
	}

	for _, o := range closeOrders {
		if o.OrderType == consts.Margin {
			s.closeMarginOrder = o
		}
		if o.OrderType == consts.Future {
			s.closeFutureOrder = o
		}
	}
}

func (s *Service) Run() {
	defer e.Recover()()
	s.ConnectAndSubscribe()
	s.ListenAndNotify()
}
func (s *Service) ConnectAndSubscribe() {
	s.connectAndLoginPrivate()
	s.connectAndSubscribePublic()
}

// ListenAndNotify 处理数据接收
func (s *Service) ListenAndNotify() {
	go func() {
		defer e.Recover()
		s.listenAndNotifyPublic()
	}()
	go func() {
		defer e.Recover()
		s.listenAndNotifyPrivate()
	}()
}

func (s *Service) Close() {
	log.Info("准备关闭连接")
	if s.pubCon != nil {
		_ = s.pubCon.Close()
	}
	if s.prvCon != nil {
		_ = s.prvCon.Close()
	}
}
