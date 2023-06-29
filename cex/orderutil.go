package cex

import (
	"ws-quant/cex/models"
	"ws-quant/pkg/mapper"
	"xorm.io/xorm"
)

func QueryOpenCloseOrders(db *xorm.Engine, cexName string) (openOrder, closeOrder *models.Orders) {
	openOrder = reloadOpen(db, cexName)
	closeOrder = reloadClose(db, cexName)
	return openOrder, closeOrder
}

func reloadOpen(db *xorm.Engine, cexName string) *models.Orders {
	// check and load open order
	openOrder := new(models.Orders)
	has := mapper.GetByWhere(db, openOrder, "cex= ? and closed = ? and pos_side= ?",
		[]interface{}{cexName, "N", "open"}...)
	if has {
		return openOrder
	} else {
		return nil
	}

}
func reloadClose(db *xorm.Engine, cexName string) *models.Orders {
	// check and load close order
	closeOrder := new(models.Orders)
	has := mapper.GetByWhere(db, closeOrder, "cex= ? and closed = ? and pos_side= ?",
		[]interface{}{cexName, "N", "close"}...)
	if has {
		return closeOrder
	} else {
		return nil
	}
}
