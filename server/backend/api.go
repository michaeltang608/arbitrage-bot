package backend

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"ws-quant/cex/models"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/gintool"
	"ws-quant/pkg/mapper"
)

func (bs *backendServer) t1(cxt *gin.Context) {

	cxt.JSON(http.StatusOK, gin.H{
		"suc": true,
	})
	return
}

func (bs *backendServer) refreshStrategy(cxt *gin.Context) {
	_ = mapper.UpdateByWhere(bs.db, &models.Orders{Closed: "Y"}, "id > ?", 1)
	bs.strategyState = 0
	bs.executingSymbol = ""

	bs.okeService.ReloadOrders()
	cxt.JSON(http.StatusOK, gin.H{
		"suc": true,
		"msg": "strategy reloaded",
	})
	feishu.Send("strategy refreshed")
	return
}

// 查询并存储 margin balance
func (bs *backendServer) marginBalances(cxt *gin.Context) {
	var err error
	if err != nil {
		gintool.Error(cxt, err)
		return
	}
	bs.persistBalance("api")
	gintool.Suc(cxt)
	return
}

func (bs *backendServer) persistBalance(reason string) {

	bs.okeService.MarginBalance()
	accountOke := &models.AccountOke{
		Total:     bs.okeService.MarginBalance(),
		Reason:    reason,
		CreatedAt: time.Now(),
	}
	_ = mapper.Insert(bs.db, accountOke)
}
