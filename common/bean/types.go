package bean

type TickerBean struct {
	CexName      string  //暂时保留，以便拓展的复用
	SymbolName   string  //统一大写，如 EOS, BTC
	InstId       string  //以是否 SWAP结尾判断是否是永续
	PriceBestAsk float64 // 价格 上
	Price        float64 // 价格 中
	PriceBestBid float64 // 价格 下
	Ts0          int64   // 接受的时间
}

// Ticker 此 ticker结构体主要是追踪长期未更新的 instId
type Ticker struct {
	PriceBestAsk float64
	Price        float64
	PriceBestBid float64
	CurTime      int64 // 这一次同步的时间
	LastTime     int64 // 上一次同步的时间，用于计算两次同步的时间差，因而告警长时间没有同步的数据，如 30s
}

type ExecState struct {
	PosSide    string //open close
	Side       string //buy sell
	InstType   string //margin, future
	OrderState string //margin, future
}

type TrackBean struct {
	State     string //默认都是 Filled
	Symbol    string
	Side      string
	InstType  string
	MyOidOpen string
	OpenPrc   string  //actual prc
	SlPrc     float64 //移动止损价
}

type OrderReq struct {
	InstType string `json:"instType"`
	Symbol   string `json:"instId"`
	Price    string `json:"price"`
	Size     string `json:"size"`
	Side     string `json:"side"`
	PosSide  string `json:"posSide"` //open/close
}
