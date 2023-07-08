package symb

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
)

func GetAllSymb() []string {
	return marginList
}

func GetAllOkFuture() []string {
	return marginFuturePairList
}
