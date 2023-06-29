package bean

type TickerBean struct {
	CexName      string
	SymbolName   string
	PriceBestAsk float64
	Price        float64
	PriceBestBid float64
	Ts0          int64
}

type Ticker struct {
	PriceBestAsk float64
	Price        float64
	PriceBestBid float64
	CurTime      int64 // 这一次同步的时间
	LastTime     int64 // 上一次同步的时间，用于计算两次同步的时间差，因而告警长时间没有同步的数据，如 30s
}

type ExecState struct {
	PosSide string //filled close
	CexName string
	Side    string //buy sell
}
