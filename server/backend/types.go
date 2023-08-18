package backend

type StrategyStateEnum int32

const (
	StateOpenSignalled   StrategyStateEnum = 1
	StateOpenFilledPart  StrategyStateEnum = 2
	StateOpenFilledAll   StrategyStateEnum = 3
	StateCloseSignalled  StrategyStateEnum = 11
	StateCloseFilledPart StrategyStateEnum = 12
	StateCloseFilledAll  StrategyStateEnum = 13
)

var (
	LogKuc = int64(1)
	LogOke = int64(2)

	LogDelay        = int64(3)
	LogMarginFuture = int64(4)
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

type MarginFutureTicker struct {
	Symbol    string
	AskMargin float64
	BidMargin float64
	AskFuture float64
	BidFuture float64
}

type TrackBean struct {
	State int
	Side  string
	SlPrc float64
	TpPrc float64
}
