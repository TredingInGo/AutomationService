package strategy

import (
	"database/sql"
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"math"
	"sort"
	"strconv"
	"time"
)

type StockResponse struct {
	StockName string  `json:"stockName"`
	Token     string  `json:"token"`
	SpotPrice float64 `json:"spotPrice"`
	StopLoss  float64 `json:"stopLoss"`
	Target    float64 `json:"target"`
	TimeFrame string  `json:"timeFrame"`
	Score     float64 `json:"score"`
}

func SwingScreener(client *smartapigo.Client, db *sql.DB) []StockResponse {
	stockList := LoadStockListForSwing(db)
	log.Printf("*************** LIST FOR SWING TRADING *****************")
	var swingStocks []StockResponse
	var timeFrames = []string{"ONE_DAY"}
	for _, timeFrame := range timeFrames {
		for _, stock := range stockList {

			swingOrder := ExecuteScreener(stock.Token, stock.Symbol, client, timeFrame)
			if swingOrder != nil {
				swingStocks = append(swingStocks, *swingOrder)
			}

		}
	}
	log.Printf("Screener Completed")
	sort.Slice(swingStocks, func(i, j int) bool {
		return swingStocks[i].Score > swingStocks[j].Score
	})
	return swingStocks
}

func ExecuteScreener(symbol, stockToken string, client *smartapigo.Client, timeFrame string) *StockResponse {

	data := GetStockTickForSwing(client, stockToken, timeFrame)
	if len(data) <= 80 {
		return nil
	}

	dataWithIndicators := &DataWithIndicators{
		Data:     data,
		Token:    stockToken,
		UserName: "Dummy",
	}

	PopulateIndicators(dataWithIndicators)
	//order := TrendFollowingRsiForSwing(dataWithIndicators, stockToken, symbol, "dummy", client)
	order := DcScreener(*dataWithIndicators, stockToken, symbol, "dummy", client)
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
		Score:     order.Score,
	}

	orderParams := SetOrderParamsForSwing(order, stockToken, symbol)
	countStock := 1
	log.Printf("\n                   STOCK No: %v                        \n", countStock)
	log.Printf("\n=========================================================\n")
	log.Printf("\nSTOCK NAME -  %v\n", symbol)
	log.Printf("SPOT PRICE - %v\n", order.Spot)
	log.Printf("STOP LOSS -  %v\n", order.Sl)
	log.Printf("TARGET -      %v\n\n", order.Tp)
	log.Printf("Order Params -      %v\n\n", orderParams)
	log.Printf("\n=========================================================\n\n")
	countStock++

	return &response
}

func TrendFollowingRsiForSwing(data *DataWithIndicators, token, symbol, username string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	sma5 := data.Indicators["sma"+"5"][idx]
	sma8 := data.Indicators["sma"+"8"][idx]
	sma13 := data.Indicators["sma"+"13"][idx]
	sma21 := data.Indicators["sma"+"21"][idx]
	adx14 := data.Adx["Adx"+"14"]
	rsi := data.Indicators["rsi"+"14"]
	var order ORDER
	order.OrderType = "None"
	log.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	log.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma5 > sma8 && sma8 > sma13 && sma13 > sma21 && rsi[idx] < 70 && rsi[idx] > 60 && rsi[idx-2] < rsi[idx] && rsi[idx-1] < rsi[idx] {
		order = ORDER{
			Spot:      data.Data[idx].High + 0.05,
			Sl:        (data.Data[idx].High * 0.05),
			Tp:        int(data.Data[idx].High * 0.10),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "BUY",
		}

	}
	order.Score = CaluclateScore(data, order)
	order.Symbol = symbol
	order.Token = token

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
		StopLoss:         float64(order.Sl),
		Quantity:         strconv.Itoa(order.Quantity),
		TrailingStopLoss: float64(1),
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

func DcScreener(data DataWithIndicators, token, symbol, username string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	var order ORDER
	order.OrderType = "None"
	if idx <= 100 {
		return order
	}
	high, low := GetORBRange(data, idx, 100)
	obv := CalculateOBV(data)

	if high == 0.0 || low == 1000000.0 {
		return order
	}
	if data.Data[idx].Close > high && IsOBVIncreasing(obv) {
		order = ORDER{
			Spot:      data.Data[idx].Close + 0.05,
			Sl:        (high - low),
			Tp:        int((high - low) * 3),
			Quantity:  1,
			OrderType: "BUY",
			Token:     "1270",
		}
	}

	//if dataWithIndicatorsMap[stockToken].Data[*idx].Close < low && dataWithIndicatorsMap[stockToken].Indicators["rsi14"][*idx] < 30 && dataWithIndicatorsMap[stockToken].Indicators["rsi14"][*idx] > 10 && strategy.IsOBVDecreasing(obv) {
	//	order = strategy.ORDER{
	//		Spot:      dataWithIndicatorsMap[stockToken].Data[*idx].Close - 0.05,
	//		Sl:        int(dataWithIndicatorsMap[stockToken].Data[*idx].Close * 0.05),
	//		Tp:        int(dataWithIndicatorsMap[stockToken].Data[*idx].Close * 0.25),
	//		Quantity:  int(Amount) / int(dataWithIndicatorsMap[stockToken].Data[*idx].Close),
	//		OrderType: "SELL",
	//		Token:     "4963",
	//	}
	//}

	if order.OrderType == "None" || order.Quantity < 1 {
		return order
	}
	//fmt.Println("OrderPlaced ", order)
	return order
}

func GetORBRange(data DataWithIndicators, idx int, dcPeriod int) (float64, float64) {

	high := 0.0
	low := 1000000.0

	for i := idx - 1; i > idx-dcPeriod; i-- {
		high = math.Max(data.Data[i].High, high)
		low = math.Min(data.Data[i].Low, low)
	}
	return high, low
}
