package oke

import (
	"github.com/gorilla/websocket"
	"sync"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/models/bean"
	"ws-quant/pkg/e"
	"ws-quant/pkg/feishu"
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
	//strategyExecOrdersChan chan *models.Orders

	usdtBal    float64 //余额
	openOrder  *models.Orders
	closeOrder *models.Orders
	db         *xorm.Engine
	debtMsgMap map[string]string
}

func New(tickerChan chan bean.TickerBean, execStateChan chan bean.ExecState,
	db_ *xorm.Engine) *Service {
	s := &Service{
		tickerChan:    tickerChan,
		db:            db_,
		execStateChan: execStateChan,
		debtMsgMap:    make(map[string]string),
	}

	s.ReloadOrders()
	if s.openOrder != nil {
		log.Info("实例化成功，find openPos")
		feishu.Send("实例化成功，find openPos")
	}
	if s.closeOrder != nil {
		log.Info("实例化成功，find closePos")
		feishu.Send("实例化成功，find closePos")
	}
	return s
}

func (s *Service) GetOpenOrder() *models.Orders {
	return s.openOrder
}
func (s *Service) GetCloseOrder() *models.Orders {
	return s.closeOrder
}

func (s *Service) SignalCloseLimit(price string) bool {
	go func() {
		log.Info("执行关仓， price=" + price)
		msg := s.ClosePosLimit(price)
		log.Info("关仓结果是:" + msg)
	}()
	return true
}

func (s *Service) ReloadOrders() {
	openOrder, closeOrder := cex.QueryOpenCloseOrders(s.db, cex.OKE)
	s.openOrder = openOrder
	s.closeOrder = closeOrder
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

func (s *Service) uploadOrder(posSide, side string) {
	s.execStateChan <- bean.ExecState{
		PosSide: posSide,
		CexName: cex.OKE,
		Side:    side,
	}
}
