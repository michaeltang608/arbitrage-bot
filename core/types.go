package core

type Side string

const (
	BUY  Side = "buy"
	SELL Side = "sell"
)

type OrderState string

func (s OrderState) State() string {
	return string(s)
}

const (
	TRIGGER  OrderState = "trigger"
	PLACED   OrderState = "placed"
	FILLED   OrderState = "filled"
	LIVE     OrderState = "live"
	CANCELED OrderState = "canceled"
)

type OrderReq struct {
	Symbol  string `json:"symbol"`
	Cex     string `json:"cex"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Side    string `json:"side"`
	PosSide string `json:"posSide"` //open/close
}

type Order struct {
	InstId string
	Price  string
	Size   string
	Side   string
	OrderState
	OrderId string
}
