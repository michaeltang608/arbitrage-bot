package orderstate

const (
	TRIGGER = "trigger"
	Failed  = "failed"
	Live    = "live"

	Filled    = "filled"
	Cancelled = "canceled"
	// 该position 已经被closed
	Closed = "closed"
)
