package binan

import (
	"fmt"
)

func (s *service) queryMarginAccount() {
	// GET /sapi/v1/margin/pair
	url := "/sapi/v1/margin/asset"
	url = fmt.Sprintf("%s?asset=USDT", url)
	respBytes := http("GET", url)
	log.Info("查询到的余额是: %v\n", string(respBytes))
}
