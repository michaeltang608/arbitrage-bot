package binan

import (
	"github.com/gorilla/websocket"
	"sync"
	"ws-quant/cex"
	"ws-quant/cex/models"
	"ws-quant/core"
	"ws-quant/models/bean"
	"ws-quant/pkg/feishu"
	logger "ws-quant/pkg/log"
	"xorm.io/xorm"
)

var (
	log = logger.NewLog("binanLog")
)

var MsgChan = make(chan []byte)

type Ping struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type service struct {
	pubCon                *websocket.Conn
	prvCon                *websocket.Conn
	pubConLock            sync.Mutex
	prvConLock            sync.Mutex
	pubConLastConnectTime int
	prvConLastConnectTime uint64

	tickerChan    chan bean.TickerBean
	execStateChan chan bean.ExecState

	lastPong   int64   //最近收到的pong的时间戳，秒，用于监听是否链接过期了
	usdtBal    float64 //余额
	order      *core.Order
	db         *xorm.Engine
	openOrder  *models.Orders
	closeOrder *models.Orders
}

func New(tickerChan chan bean.TickerBean, execStateChan chan bean.ExecState,
	db_ *xorm.Engine) cex.Service {
	s := &service{
		tickerChan:    tickerChan,
		db:            db_,
		execStateChan: execStateChan,
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

func (s *service) MarginBalance() float64 {
	//TODO implement me
	panic("implement me")
}

func (s *service) ClosePosLimit(price string) (msg string) {
	//TODO implement me
	panic("implement me")
}

func (s *service) TradeLimit(symbol, price, size, side, posSide string) string {
	//TODO implement me
	panic("implement me")
}

func (s *service) ClosePosMarket(askPrc float64, bidPrc float64) (msg string) {
	//TODO implement me
	panic("implement me")
}

func (s *service) OpenPosLimit(symbol, price, size, side string) (msg string) {
	//TODO implement me
	panic("implement me")
}

func (s *service) OpenPosMarket(symbol, size, side string) (msg string) {
	//TODO implement me
	panic("implement me")
}

func (s *service) GetOpenOrder() *models.Orders {
	return s.openOrder
}
func (s *service) GetCloseOrder() *models.Orders {
	return s.closeOrder
}

func (s *service) SignalCloseLimit(price string) bool {
	//go func() {
	//	log.Info("执行关仓， price=" + price)
	//	msg := s.ClosePosLimit(price)
	//	log.Info("关仓结果是:" + msg)
	//}()
	return true
}

// ReloadOrders 重新从db加载开仓和关仓订单信息
func (s *service) ReloadOrders() {
	//openOrder, closeOrder := cex.QueryOpenCloseOrders(s.db, s.GetCexName())
	//s.openOrder = openOrder
	//s.closeOrder = closeOrder
	//TODO implement me
}

func (s *service) Run() {
	//defer e.Recover()()
	//go s.checkPong()
	//s.ConnectAndSubscribe()
	s.ListenAndNotify()
}

func (s *service) ConnectAndSubscribe() {
	//s.ConnectAndSubscribePublic()
	//s.ConnectAndSubscribePrivate()
}

func (s *service) ListenAndNotify() {
	//go func() {
	go s.ListenAndNotifyPublic()
	//}()
	//go func() {
	//	defer e.Recover()()
	//	s.ListenAndNotifyPrivate()
	//}()
}

func (s *service) GetCexName() string {
	return cex.BINAN
}

func (s *service) Close() {
	log.Info("准备关闭连接")
	//_ = s.conn.Close()
}
