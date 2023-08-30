package backend

import (
	"github.com/gin-gonic/gin"
	"ws-quant/common/bean"
	"ws-quant/pkg/gintool"
)

func (bs *backendServer) openLimit(cxt *gin.Context) {
	var req bean.OrderReq
	err := cxt.Bind(&req)
	if err != nil {
		gintool.Error(cxt, err)
		return
	}
	msg := bs.okeService.TradeLimit(req)
	gintool.SucMsg(cxt, msg)
	return
}

func (bs *backendServer) cancelOrder(cxt *gin.Context) {
	service := bs.okeService
	msg := service.CancelOrder(cxt.Query("instType"))
	gintool.SucMsg(cxt, msg)
	return
}

func (bs *backendServer) closeMarket(cxt *gin.Context) {
	gintool.SucMsg(cxt, bs.okeService.CloseOrder(cxt.Query("instType")))
	return
}
