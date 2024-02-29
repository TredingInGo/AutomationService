package strategy

import (
	"database/sql"
	"fmt"
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
	"strconv"
)

type StockResponse struct {
	StockName string  `json:"stockName"`
	Token     string  `json:"token"`
	SpotPrice float64 `json:"spotPrice"`
	StopLoss  float64 `json:"stopLoss"`
	Target    float64 `json:"target"`
	TimeFrame string  `json:"timeFrame"`
}

func SwingScreener(client *smartapigo.Client, db *sql.DB) []StockResponse {
	stockList := LoadStockListForSwing(db)
	fmt.Printf("*************** LIST FOR SWING TRADING *****************")
	var swingStocks []StockResponse
	var timeFrames = []string{"ONE_HOUR", "ONE_DAY"}
	for _, timeFrame := range timeFrames {
		for _, stock := range stockList {

			swingOrder := ExecuteScreener(stock.Token, stock.Symbol, client, timeFrame)
			if swingOrder != nil {
				swingStocks = append(swingStocks, *swingOrder)
			}

		}
	}
	fmt.Printf("Screener Completed")
	return swingStocks
}

func ExecuteScreener(symbol, stockToken string, client *smartapigo.Client, timeFrame string) *StockResponse {

	data := GetStockTickForSwing(client, stockToken, timeFrame)
	//fmt.Printf("\nStock Name: %v, DataSize: %v\n", symbol, len(data))
	if len(data) <= 30 {
		return nil
	}
	PopulateIndicators(data, stockToken, "Swing")
	order := TrendFollowingRsiForSwing(data, stockToken, symbol)
	if order.OrderType == "None" {
		return nil
	}

	response := StockResponse{
		StockName: symbol,
		Token:     stockToken,
		SpotPrice: order.Spot,
		StopLoss:  float64(order.Sl),
		Target:    float64(order.Tp),
		TimeFrame: timeFrame,
	}

	orderParams := SetOrderParamsForSwing(order, stockToken, symbol)
	countStock := 1
	fmt.Printf("\n                   STOCK No: %v                        \n", countStock)
	fmt.Printf("\n=========================================================\n")
	fmt.Printf("\nSTOCK NAME -  %v\n", symbol)
	fmt.Printf("SPOT PRICE - %v\n", order.Spot)
	fmt.Printf("STOP LOSS -  %v\n", order.Sl)
	fmt.Printf("TARGET -      %v\n\n", order.Tp)
	fmt.Printf("Order Params -      %v\n\n", orderParams)
	fmt.Printf("\n=========================================================\n\n")
	countStock++

	return &response
}

func TrendFollowingRsiForSwing(data []smartapigo.CandleResponse, token, symbol string) ORDER {
	idx := len(data) - 1
	sma5 := sma["Swing"+token+"5"][idx]
	sma8 := sma["Swing"+token+"8"][idx]
	adx14 := adx["Swing"+token]
	rsi := rsi["Swing"+token]
	var order ORDER
	order.OrderType = "None"
	swingLow := GetSwingLow(data, 10)
	var _ = GetAvgVolume(data, 20)
	if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma5 > sma8 && sma8 > sma["Swing"+token+"13"][idx] && sma["Swing"+token+"13"][idx] > sma["Swing"+token+"21"][idx] && rsi[idx] < 70 && rsi[idx] > 60 && rsi[idx-4] < rsi[idx] && rsi[idx-2] > rsi[idx-4] {
		order = ORDER{
			Spot:      data[idx].High + 0.05,
			Sl:        int(swingLow),
			Tp:        int(data[idx].High + 2*(data[idx].High-swingLow)),
			Quantity:  1,
			OrderType: "BUY",
		}
	}

	return order
}

func SetOrderParamsForSwing(order ORDER, token, symbol string) smartapigo.OrderParams {

	orderParams := smartapigo.OrderParams{
		Variety:          "GTT",
		TradingSymbol:    symbol + "-EQ",
		SymbolToken:      token,
		TransactionType:  order.OrderType,
		Exchange:         "NSE",
		OrderType:        "LIMIT",
		ProductType:      "GTT",
		Duration:         "DELIVERY",
		Price:            strconv.FormatFloat(order.Spot, 'f', 2, 64),
		SquareOff:        strconv.Itoa(order.Tp),
		StopLoss:         strconv.Itoa(order.Sl),
		Quantity:         strconv.Itoa(order.Quantity),
		TrailingStopLoss: strconv.Itoa(1),
	}
	return orderParams
}

func GetSwingLow(data []smartapigo.CandleResponse, day int) float64 {
	length := len(data)
	low := 1000000.0
	for i := length - 1; i > length-day-1; i-- {
		low = math.Min(low, data[i].Low)
	}
	return low
}

func GetAvgVolume(data []smartapigo.CandleResponse, day int) float64 {
	length := len(data)
	sum := 0
	for i := length - 1; i > length-day-1; i-- {
		sum += data[i].Volume
	}
	return float64(sum) / float64(day)
}
