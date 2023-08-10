package cex

import (
	"ws-quant/cex/models"
	"ws-quant/common/consts"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/mapper"
	"xorm.io/xorm"
)

func QueryOpenCloseOrders(db *xorm.Engine) (openOrders, closeOrders []*models.Orders) {

	orders := make([]*models.Orders, 0)
	openOrders = make([]*models.Orders, 0)
	closeOrders = make([]*models.Orders, 0)
	mapper.Find(db, &orders, &models.Orders{Closed: "N"})
	if len(orders) > 4 {
		feishu.Send("orders > 4, plz check")
		return
	}
	for _, o := range orders {
		if o.PosSide == consts.Open {
			openOrders = append(openOrders, o)
		}
		if o.PosSide == consts.Close {
			closeOrders = append(closeOrders, o)
		}
	}

	return openOrders, closeOrders
}
