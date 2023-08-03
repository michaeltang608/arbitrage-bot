package models

import "time"

type AccountOke struct {
	ID        uint32    `json:"id" xorm:"notnull pk autoincr int id comment('id')"`
	Total     float64   `json:"total" xorm:"notnull default 0 Double comment('总额')"`
	Reason    string    `json:"reason" xorm:"notnull default '' Varchar(30) comment('原因')"`
	CreatedAt time.Time `json:"created_at" xorm:"notnull default CURRENT_TIMESTAMP timestamp comment('创建时间')"`
}
