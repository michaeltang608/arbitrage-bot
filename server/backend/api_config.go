package backend

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"ws-quant/cex/models"
	"ws-quant/pkg/gintool"
	"ws-quant/pkg/mapper"
)

func (bs *backendServer) queryExecState(cxt *gin.Context) {
	data := make(map[string]interface{}, 0)
	data["executingSymbol"] = bs.executingSymbol
	data["triggerState"] = bs.triggerState
	data["orderStat"] = bs.execStates
	data["StrategyOpenThreshold"] = bs.config.StrategyOpenThreshold
	data["TradeAmt"] = bs.config.TradeAmt

	data["marginTrack"] = ""
	data["futureTrack"] = ""
	if bs.marginTrack != nil {
		data["marginTrack"] = fmt.Sprintf("%+v", bs.marginTrack)
	}
	if bs.futureTrack != nil {
		data["futureTrack"] = fmt.Sprintf("%+v", bs.futureTrack)
	}
	gintool.SucData(cxt, data)
}
func (bs *backendServer) getConfig(cxt *gin.Context) {
	gintool.SucData(cxt, bs.config)
}

func (bs *backendServer) changeConfig(cxt *gin.Context) {
	req := new(models.Config)
	err := cxt.ShouldBind(req)
	if err != nil {
		gintool.Error(cxt, err)
		return
	}
	bs.config = req
	//persist into db
	mapper.UpdateById(bs.db, 1, req)
	gintool.Suc(cxt)
	return
}
