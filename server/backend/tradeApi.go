package backend

import (
	"github.com/gin-gonic/gin"
	"ws-quant/core"
	"ws-quant/pkg/gintool"
)

func (bs *backendServer) openLimit(cxt *gin.Context) {
	var req core.OrderReq
	err := cxt.Bind(&req)
	if err != nil {
		gintool.Error(cxt, err)
		return
	}
	msg := bs.okeService.TradeLimit(req.InstId, req.Price, req.Size, req.Side, "open")
	gintool.SucMsg(cxt, msg)
	return
}

func (bs *backendServer) closeMarket(cxt *gin.Context) {
	//orderType := cxt.DefaultQuery("orderType", consts.Margin)
	//if orderType == consts.Margin{
	//	bs.okeService.CloseMarginMarket()
	//}
}