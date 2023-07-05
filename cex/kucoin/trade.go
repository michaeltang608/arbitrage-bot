package kucoin

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/valyala/fastjson"
	"math"
	"net/http"
	"strconv"
	"strings"
	"ws-quant/cex"
	"ws-quant/core"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/util"
)

type OrderReq struct {
	ClientOid   string `json:"clientOid"`
	Side        string `json:"side"` //buy sell
	Symbol      string `json:"symbol"`
	AutoBorrow  bool   `json:"autoBorrow"`
	Price       string `json:"price"`
	Type        string `json:"type"`
	Size        string `json:"size"`
	TimeInForce string `json:"timeInForce"` //默认GTC
}

func (s *service) OpenPosLimit(symbol, price, size, side string) (msg string) {
	if s.openOrder != nil {
		return fmt.Sprintf("已开仓中，勿再重复开仓:%+v", *s.openOrder)
	}
	return s.TradeLimit(symbol, price, size, side, "open")
}

func (s *service) OpenPosMarket(symbol, size, side string) (msg string) {
	if s.openOrder != nil {
		return fmt.Sprintf("已开仓中，勿再重复开仓:%+v", *s.openOrder)
	}
	return s.TradeMarket(symbol, size, side, "open")
}

func (s *service) ClosePosMarket(askPrc float64, bidPrc float64) (msg string) {
	// 为开仓或者 开的仓位未成交
	if s.openOrder == nil || s.openOrder.State != string(core.FILLED) {
		return "无仓位需要平仓"
	}
	if s.closeOrder != nil {
		return "无需重复平仓"
	}
	side := cex.Buy
	if s.openOrder.Side == cex.Buy {
		side = cex.Sell
	}
	sizeFloat, _ := strconv.ParseFloat(s.openOrder.Size, 64)
	size := util.AdjustClosePosSize(sizeFloat, side, cex.KUCOIN)
	symbol := strings.Split(s.openOrder.InstId, "-")[0]
	return s.TradeMarket(symbol, size, side, cex.Close)
}
func (s *service) ClosePosLimit(price string) (msg string) {
	// 为开仓或者 开的仓位未成交
	if s.openOrder == nil || s.openOrder.State != string(core.FILLED) {
		return "无仓位需要平仓"
	}
	if s.closeOrder != nil {
		return "无需重复平仓"
	}
	side := cex.Buy
	if s.openOrder.Side == cex.Buy {
		side = cex.Sell
	}
	sizeFloat, _ := strconv.ParseFloat(s.openOrder.Size, 64)
	size := util.AdjustClosePosSize(sizeFloat, side, cex.KUCOIN)
	symbol := strings.Split(s.openOrder.InstId, "-")[0]
	return s.TradeLimit(symbol, price, size, side, cex.Close)
}

func (s *service) TradeLimit(symbol, price, size, side, posSide string) string {
	// kucoin的订单信息将会全部在推送中获取
	log.Info("准备执行trade")
	url := "/api/v1/margin/order"
	instId := fmt.Sprintf("%s-USDT", strings.ToUpper(symbol))

	//如果是平仓，无需借币；如果是买单也无需借币，只有开仓且卖单才需借币
	req := OrderReq{
		ClientOid:   uuid.New().String(),
		Side:        strings.ToLower(side),
		Symbol:      instId,
		AutoBorrow:  posSide == "open" && strings.ToLower(side) == "sell",
		Price:       price,
		Size:        size,
		TimeInForce: "GTC",
	}

	reqBytes, _ := json.Marshal(req)
	respBytes := authHttpRequest(url, http.MethodPost, string(reqBytes))
	result := string(respBytes)
	if fastjson.GetString(respBytes, "code") != "200000" {
		feishu.Send("ku 上链limit 失败," + result)
	}
	return result
}
func (s *service) TradeMarket(symbol, size, side, posSide string) string {
	// kucoin的订单信息将会全部在推送中获取
	log.Info("准备执行TradeMarket")
	url := "/api/v1/margin/order"
	instId := fmt.Sprintf("%s-USDT", strings.ToUpper(symbol))

	//如果是平仓，无需借币；如果是买单也无需借币，只有开仓且卖单才需借币
	req := OrderReq{
		ClientOid:  uuid.New().String(),
		Side:       strings.ToLower(side),
		Symbol:     instId,
		Type:       "market",
		AutoBorrow: posSide == "open" && strings.ToLower(side) == "sell",
		Size:       size,
	}

	reqBytes, _ := json.Marshal(req)
	respBytes := authHttpRequest(url, http.MethodPost, string(reqBytes))
	result := string(respBytes)
	if fastjson.GetString(respBytes, "code") != "200000" {
		log.Error("ku开仓市价单失败, 返回报文=%s", result)
		feishu.Send("ku上链 market 失败," + result)
	}
	return result
}

// 查询待还款，用于卖单后的平仓操作
func queryDebt() {
	url := "/api/v1/margin/borrow/outstanding?currency=DAO"
	respBytes := authHttpRequest(url, http.MethodGet, "")
	log.Info("resp of query in debt: " + string(respBytes))
	totalStr := fastjson.GetString(respBytes, "data", "items", "0", "liability")
	totalFloat, _ := strconv.ParseFloat(totalStr, 64)
	//保留到万分之一再向上取整
	totalFloat = math.Ceil(totalFloat*10000) / 10000
	log.Info("需要偿还的DAO金额是: %v\n", totalFloat)
}
