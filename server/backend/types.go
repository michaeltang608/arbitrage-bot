package backend

type StrategyStateEnum int32

var (
	LogOke = int64(2)
)

type RatioBean struct {
	UpperPct float64 //cex按照字母排序，最大的 pct
	LowerPct float64 //
}

// SignalCalBean 用于传递价格变动计算的 bean
type SignalCalBean struct {
	Symbol string
	Ts0    int64
	Ts1    int64
}

type Oppor struct {
	Symbol   string  //交易对，大写，如 EOS
	OpenDiff float64 // 如 1.0
	MaxDiff  float64 // 真实中最大的max diff
	MaxPrice float64
	MinPrice float64
	MaxCex   string
	MinCex   string
}

type OkBitTicker struct {
	Symbol   string
	AskOk    float64
	BidOk    float64
	AskBit   float64
	BidBit   float64
	LastTime int64
}
