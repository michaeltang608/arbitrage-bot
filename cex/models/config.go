package models

type Config struct {
	ID                     uint32  `json:"id" xorm:"notnull pk autoincr int id comment('id')"`
	TickerTimeout          int64   `json:"ticker_timeout" xorm:"default 300 BigInt comment('ticker超时时间秒')"`
	LogThreshold           float64 `json:"log_threshold" xorm:"default 1.0 Double comment('打印限制')"`
	LogTicker              int64   `json:"log_ticker" xorm:"default 300 BigInt comment('打印ticker')"`
	TradeAmt               float64 `json:"trade_amt" xorm:"default 20.0 Double comment('交易金额')"`
	TradeAmtMax            float64 `json:"trade_amt_max" xorm:"default 20.0 Double comment('交易金额')"`
	LogSymbol              string  `json:"log_symbol" xorm:"default '' Varchar(20) comment('打印品种')"`
	StrategyOpenThreshold  float64 `json:"strategy_open_threshold" xorm:"default 1.0 Double comment('开的条件')"`
	StrategyCloseThreshold float64 `json:"strategy_close_threshold" xorm:"default 0.1 Double comment('关的条件')"`
}
