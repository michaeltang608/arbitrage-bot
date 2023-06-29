package models

import "time"

type Account struct {
	ID        uint32    `json:"id" xorm:"notnull pk autoincr int id comment('id')"`
	Body      string    `json:"body" xorm:"notnull default '' Varchar(300) comment('各个cex详情数据')"`
	Total     float64   `json:"total" xorm:"notnull default 0 Double comment('总额')"`
	Pct       float64   `json:"pct" xorm:"notnull default 0 Double comment('收益率')"`
	Type      string    `json:"type" xorm:"notnull default '0' Varchar(10) comment('收益率')"`
	CreatedAt time.Time `json:"created_at" xorm:"notnull default CURRENT_TIMESTAMP timestamp comment('创建时间')"`
}
