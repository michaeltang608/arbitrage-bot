package symb

import "strings"

var (
	marginList = []string{"HBAR", "XMR", "ADA", "OMG", "DYDX", "AAVE", "GRT", "ETC", "SAND", "OP", "IMX", "NEAR",
		"SUSHI", "AR", "ZEC", "ETH", "MASK", "ZIL", "ETH", "DASH", "NEO", "GMT", "MANA", "MATIC", "CRO", "SOL", "XLM",
		"BNB", "TRX", "AVAX", "ATOM", "SHIB", "SUI", "TRX", "BLUR", "ALGO", "UNI", "SNX", "APT", "FIL", "STX",
		"EOS", "WAVES", "LTC", "ARB", "WOO", "CFX", "FLOKI", "ICP", "DOT", "APE", "LINK", "XRP", "EGLD", "THETA",
		"DOGE", "USDC", "XRP", "CRV"}

	futureList = []string{"BTC", "ETH", "LTC", "XRP", "BCH", "SOL", "PEPE", "FIL", "ORDI", "1INCH", "AAVE", "ADA", "AGLD",
		"AIDOGE", "ALGO", "ALPHA", "ANT", "APE", "API3", "APT", "AR", "ARB", "ATOM", "AVAX", "AXS", "BADGER", "BAL",
		"BAND", "BAT", "BICO", "BLUR", "BNB", "BNT", "BSV", "CELO", "CEL", "CETUS", "CFX", "CHZ", "COMP", "CORE",
		"CRO", "CRV", "CSPR", "CVC", "DASH", "DOGE", "DORA", "DOT", "DYDX", "EGLD", "ENJ", "ENS", "EOS", "ETC", "ETHW",
		"FITFI", "FLM", "FLOKI", "FTM", "GALA", "GFT", "GMT", "GMX", "GODS", "GRT", "ICP", "IMX", "IOST", "IOTA", "JST",
		"KISHU", "KLAY", "KNC", "KSM", "LDO", "LINK", "LOOKS", "LPT", "LRC", "LUNA", "LUNC", "MAGIC", "MANA", "MASK",
		"MATIC", "MINA", "MKR", "NEAR", "NEO", "NFT", "OMG", "ONT", "OP", "PEOPLE", "PERP", "QTUM", "RDNT", "REN", "RSR",
		"RVN", "SAND", "SHIB", "SLP", "SNX", "STARL", "STORJ", "STX", "SUI", "SUSHI", "SWEAT", "THETA", "TON", "TRB",
		"TRX", "UMA", "UNI", "USDC", "USTC", "WAVES", "WOO", "XCH", "XLM", "XMR", "XTZ", "YFI", "YFII", "YGG", "ZEC",
		"ZEN", "ZIL", "ZRX"}

	marginFuturePairList = []string{"XMR", "ADA", "OMG", "DYDX", "AAVE", "GRT", "ETC", "SAND", "OP", "IMX", "NEAR",
		"SUSHI", "AR", "ZEC", "ETH", "MASK", "ZIL", "ETH", "DASH", "NEO", "GMT", "MANA", "MATIC", "CRO", "SOL", "XLM",
		"BNB", "TRX", "AVAX", "ATOM", "SHIB", "SUI", "TRX", "BLUR", "ALGO", "UNI", "SNX", "APT", "FIL", "STX", "EOS",
		"WAVES", "LTC", "ARB", "WOO", "CFX", "FLOKI", "ICP", "DOT", "APE", "LINK", "XRP", "EGLD", "THETA", "DOGE", "USDC",
		"XRP", "CRV"}

	resultMarginBinanOke = []string{"HBAR", "XMR", "ADA", "DYDX", "AAVE", "LUNA", "GRT", "ETC", "SAND", "OP", "IMX", "NEAR", "SUSHI", "AR", "ZEC", "ETH", "MASK", "ZIL", "ETH", "DASH", "NEO", "GMT", "MANA", "MATIC", "LDO", "SOL", "XLM", "TRX", "FTM", "AVAX", "ATOM", "SHIB", "SUI", "TRX", "ALGO", "UNI", "SNX", "APT", "FIL", "STX", "USTC", "EOS", "WAVES", "LTC", "ARB", "WOO", "PEPE", "CFX", "FLOKI", "ICP", "DOT", "APE", "LINK", "XRP", "EGLD", "MINA", "GMX", "THETA", "DOGE", "USDC", "LUNC", "BCH", "XRP", "CRV"}

	symbolMap = make(map[string]struct{})
)

func init() {
	for _, s := range resultMarginBinanOke {
		symbolMap[s] = struct{}{}
	}
}

func GetAllSymb() []string {
	return resultMarginBinanOke
}

func SymbolExist(symbol string) bool {
	_, exist := symbolMap[strings.ToUpper(symbol)]
	return exist
}

func GetAllOkFuture() []string {
	return marginFuturePairList
}
