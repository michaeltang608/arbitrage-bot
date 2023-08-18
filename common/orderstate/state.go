package orderstate

const (
	Nil = iota
	Failed
	Live
	Filled
	Closed
	Cancelled
)
