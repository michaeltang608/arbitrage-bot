package cex

const (
	//BINAN = "binan"
	//HUO   = "huo"
	OKE    = "oke"
	KUCOIN = "kucoin"
	Buy    = "buy"
	Sell   = "sell"
	Open   = "open"
	Close  = "close"
)

func GetAllCex() []string {
	return []string{OKE, KUCOIN}
}
