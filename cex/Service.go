package cex

import "ws-quant/cex/models"

type Service interface {
	Run()
	GetCexName() string
	ReloadOrders()
	GetOpenOrder() *models.Orders
	GetCloseOrder() *models.Orders
	ConnectAndSubscribe()
	ListenAndNotify()
	Close()

	SignalCloseLimit(price string) bool

	// MarginBalance for future ROI stats
	MarginBalance() float64

	ClosePosLimit(price string) (msg string)
	ClosePosMarket(askPrc float64, bidPrc float64) (msg string)
	OpenPosLimit(symbol, price, size, side string) (msg string)
	OpenPosMarket(symbol, size, side string) (msg string)
}
