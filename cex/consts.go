package cex

const (
	BINAN = "binan"
	OKE   = "oke"
	Buy   = "buy"
	Sell  = "sell"
	Open  = "open"
	Close = "close"
)

func GetAllCex() []string {
	return []string{OKE, BINAN}
}
