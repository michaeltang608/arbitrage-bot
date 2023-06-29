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
