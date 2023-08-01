package models

import "time"

type Orders struct {
	ID         uint32    `json:"id" xorm:"notnull pk autoincr int id comment('id')"`
	InstId     string    `json:"inst_id" xorm:"notnull default '' Varchar(30) comment('InstId')"`
	Cex        string    `json:"cex" xorm:"notnull default '' Varchar(30) comment('cex')"`
	Price      string    `json:"price" xorm:"notnull default '' Varchar(30) comment('limit 价格')"`
	Size       string    `json:"size" xorm:"notnull default '' Varchar(30) comment('开仓量')"`
	NumPerSize string    `json:"numPerSize" xorm:"notnull default '' Varchar(30) comment('单张大小')"`
	Side       string    `json:"side" xorm:"notnull default '' Varchar(10) comment('买卖')"`
	PosSide    string    `json:"pos_side" xorm:"notnull default '' Varchar(10) comment('开闭仓，open/close')"`
	State      string    `json:"state" xorm:"notnull default '' Varchar(20) comment('下单、成交、取消')"`
	OrderId    string    `json:"order_id" xorm:"notnull default '' Varchar(80) comment('订单Id')"`
	Closed     string    `json:"closed" xorm:"notnull default 'N' Varchar(10) comment('是否关闭，默认否N')"`
	OrderType  string    `json:"order_type" xorm:"notnull default 'limit' Varchar(10) comment('订单类型market/limit')"`
	Created    time.Time `json:"created" xorm:"notnull default CURRENT_TIMESTAMP timestamp comment('创建时间')"`
	FilledTime time.Time `json:"filled_time" xorm:"timestamp comment('filled time')"`
	Updated    time.Time `json:"updated" xorm:"notnull default CURRENT_TIMESTAMP timestamp comment('更新时间')"`
}

type Config struct {
	ID                     uint32  `json:"id" xorm:"notnull pk autoincr int id comment('id')"`
	TickerTimeout          int64   `json:"ticker_timeout" xorm:"default 300 BigInt comment('ticker超时时间秒')"`
	LogThreshold           float64 `json:"log_threshold" xorm:"default 1.0 Double comment('打印限制')"`
	LogTicker              int64   `json:"log_ticker" xorm:"default 300 BigInt comment('打印ticker')"`
	TradeAmt               float64 `json:"trade_amt" xorm:"default 20.0 Double comment('交易金额')"`
	LogSymbol              string  `json:"log_symbol" xorm:"default '' Varchar(20) comment('打印品种')"`
	StrategyOpenThreshold  float64 `json:"strategy_open_threshold" xorm:"default 1.0 Double comment('开的条件')"`
	StrategyCloseThreshold float64 `json:"strategy_close_threshold" xorm:"default 0.1 Double comment('关的条件')"`
}

type Oppor struct {
	ID      uint32    `json:"id" xorm:"notnull pk autoincr int id comment('id')"`
	InstId  string    `json:"inst_id" xorm:"notnull default '' Varchar(30) comment('InstId')"`
	Cex     string    `json:"cex" xorm:"notnull default '' Varchar(30) comment('cex')"`
	MaxDiff string    `json:"price" xorm:"notnull default '' Varchar(50) comment('limit 价格')"`
	Created time.Time `json:"created" xorm:"notnull default CURRENT_TIMESTAMP timestamp comment('创建时间')"`
}
