package backend

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ws-quant/cex/models"
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

func (bs *backendServer) persistBalance(type_ string) {
	//todo
	//result := make(map[string]interface{})
	//
	//total := 0.0
	//for _, service := range bs.cexServiceMap {
	//	result[service.GetCexName()] = service.MarginBalance()
	//	total += service.MarginBalance()
	//}
	//marshal, _ := json.Marshal(result)
	//acc := &models.Account{
	//	ID:        0,
	//	Body:      string(marshal),
	//	Total:     total,
	//	Type:      type_,
	//	CreatedAt: time.Now(),
	//}
	//_ = mapper.Insert(bs.db, acc)
}

// 尝试 ok下单
//func (bs *backendServer) closePos(cxt *gin.Context) {
//	var req core.OrderReq
//	err := cxt.Bind(&req)
//	if err != nil {
//		gintool.Error(cxt, err)
//		return
//	}
//	service, ok := bs.cexServiceMap[req.Cex]
//	if !ok {
//		gintool.SucMsg(cxt, "cex不存在")
//		return
//	}
//	msg := service.ClosePosMarket()
//	gintool.SucMsg(cxt, msg)
//	return
//}
