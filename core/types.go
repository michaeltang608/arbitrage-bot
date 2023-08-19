package core

type Side string

type OrderState string

func (s OrderState) State() string {
	return string(s)
}

const (
// TRIGGER OrderState = "trigger"
)

type OrderReq struct {
	InstType string `json:"instType"`
	Symbol   string `json:"instId"`
	Price    string `json:"price"`
	Size     string `json:"size"`
	Side     string `json:"side"`
	PosSide  string `json:"posSide"` //open/close
}

type Order struct {
	InstId string
	Price  string
	Size   string
	Side   string
	OrderState
	OrderId string
}
