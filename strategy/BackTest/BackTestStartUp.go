package BackTest

import (
	"database/sql"
	"fmt"
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"math"
	"sort"
	"strconv"
	"time"
)

const (
	worker = 1
)

var stockData = make(map[string][]smartapigo.CandleResponse)
var Amount = 100000000.0

type kpi struct {
	trade            int
	profit           float64
	loss             float64
	maxContinousloss float64
	profitCount      float64
	lossCount        float64
	amount           float64
}

var dataWithIndicatorsMap = make(map[string]strategy.DataWithIndicators)

var tradeReport = initTrade()
var count = 0.0
var amountChange []float64
var trades []int

func BackTest(client *smartapigo.Client, db *sql.DB) {
	stock := strategy.Symbols{
		"99926000",
		"NIFTY",
	}
	var stockList []strategy.Symbols
	stockList = append(stockList, stock)
	populateStockData(stockList, client)
	//rsi, ema,
	maxProfit := initTrade()
	rsiVal := 5
	ema := false

	tradeReport = initTrade()
	Amount = 100000000
	executeBacktest(client, stockList, 14, false)
	printCurrentTradeReport()
	if tradeReport.profit > maxProfit.profit {
		maxProfit = tradeReport
		rsiVal = 14
		ema = false
	}

	log.Println("********************|| FINAL TRADE REPORT ||************************")
	tradeReport = maxProfit
	log.Println("Rsi: ", rsiVal, " isEma ", ema)
	printCurrentTradeReport()
	//plotGraph(amountChange, trades)
}

func populateStockData(stockList []strategy.Symbols, client *smartapigo.Client) {
	count := 1
	for _, stock := range stockList {
		populateStockTick(client, stock.Symbol, "FIVE_MINUTE")
		dataWithIndicators := strategy.DataWithIndicators{
			Data:     stockData[stock.Symbol],
			Token:    stock.Symbol,
			UserName: "BackTest",
		}
		strategy.PopulateIndicators(&dataWithIndicators)
		dataWithIndicatorsMap[stock.Symbol] = dataWithIndicators
		log.Println(count, "data Population done for ", stock.Token)
		count++

	}

}

// populate OHLC data of each stock of 1000 days.

func populateStockTick(client *smartapigo.Client, symbolToken string, timeFrame string) {

	tempTime := time.Now()
	var candles []smartapigo.CandleResponse
	for i := 0; i < 9; i++ {
		toDate := tempTime.Format("2006-01-02 15:04")
		fromDate := tempTime.Add(time.Hour * 24 * -100).Format("2006-01-02 15:04")
		tempTime = tempTime.Add(time.Hour * 24 * -100)

		// set timeout
		client.SetTimeout(5 * time.Second)

		tempHistoryData, _ := client.GetCandleData(smartapigo.CandleParams{
			Exchange:    "NSE",
			SymbolToken: symbolToken,
			Interval:    timeFrame,
			FromDate:    fromDate,
			ToDate:      toDate,
		})
		candles = append(candles, tempHistoryData...)
	}

	sort.Slice(candles, func(i, j int) bool {
		return candles[j].Timestamp.After(candles[i].Timestamp)
	})

	stockData[symbolToken] = candles
	return
}

func executeBacktest(client *smartapigo.Client, stockList []strategy.Symbols, rsiPeriod int, isEma bool) {
	idx := 100
	for idx < len(stockData[stockList[0].Symbol]) {
		eligibleStocks := getEligibleStocks(stockList, client, "BackTest", &idx, rsiPeriod, isEma)
		sort.Slice(eligibleStocks, func(i, j int) bool {
			return eligibleStocks[i].Score > eligibleStocks[j].Score
		})
		if len(eligibleStocks) == 0 {
			idx++
			continue
		}
		PlaceOrder(eligibleStocks[0], eligibleStocks[0].Token, &idx)
		//printCurrentTradeReport()
		idx++
	}
}

func getStockTick(stockToken string, idx *int) []smartapigo.CandleResponse {
	var empty = []smartapigo.CandleResponse{}
	if *idx > len(stockData[stockToken]) {
		return empty
	}
	return stockData[stockToken][0:*idx]
}

func getEligibleStocks(stocks []strategy.Symbols, client *smartapigo.Client, userName string, idx *int, rsiPeriod int, isEma bool) []*strategy.ORDER {

	filteredStocks := []*strategy.ORDER{}

	for _, stock := range stocks {
		param := strategy.EligibleStockParam{
			Symbols:  strategy.Symbols{Symbol: stock.Symbol, Token: stock.Token},
			UserName: userName,
		}

		order := Execute(param.Token, param.Symbol, client, param.UserName, idx, rsiPeriod, isEma)
		if order != nil {
			filteredStocks = append(filteredStocks, order)
		}
	}

	//log.Println("Time to get orders ", time.Since(start))

	return filteredStocks
}

func Execute(symbol, stockToken string, client *smartapigo.Client, userName string, idx *int, rsiPeriod int, isEma bool) *strategy.ORDER {
	if len(dataWithIndicatorsMap[stockToken].Data) == 0 || len(dataWithIndicatorsMap[stockToken].Data) <= *idx {
		return nil
	}
	high, low := GetORBRange(dataWithIndicatorsMap[stockToken], idx)
	var order strategy.ORDER
	order.OrderType = "None"
	if high == 0.0 || low == 1000000.0 {
		return &order
	}
	if dataWithIndicatorsMap[stockToken].Data[*idx].Close > high && dataWithIndicatorsMap[stockToken].Indicators["rsi20"][*idx] > 40 && dataWithIndicatorsMap[stockToken].Indicators["rsi14"][*idx] > 40 {
		order = strategy.ORDER{
			Spot:      high + 0.05,
			Sl:        5,
			Tp:        25,
			Quantity:  25,
			OrderType: "BUY",
			Token:     "99926000",
		}
	}

	if dataWithIndicatorsMap[stockToken].Data[*idx].Close < low && dataWithIndicatorsMap[stockToken].Indicators["rsi14"][*idx] < 30 && dataWithIndicatorsMap[stockToken].Indicators["rsi20"][*idx] < 30 {
		order = strategy.ORDER{
			Spot:      low - 0.05,
			Sl:        5,
			Tp:        25,
			Quantity:  25,
			OrderType: "SELL",
			Token:     "99926000",
		}
	}

	if order.OrderType == "None" || order.Quantity < 1 {
		return nil
	}
	fmt.Println("OrderPlaced ", order)
	return &order
}

func GetORBRange(data strategy.DataWithIndicators, idx *int) (float64, float64) {

	high := 0.0
	low := 1000000.0

	for i := *idx - 1; i > *idx-21; i-- {
		high = math.Max(data.Data[i].High, high)
		low = math.Min(data.Data[i].Low, low)
	}
	return high, low
}

func PlaceOrder(order *strategy.ORDER, symbol string, idx *int) {
	sl := float64(order.Sl)
	tp := float64(order.Tp)
	if order.OrderType == "BUY" {
		tp = order.Spot + tp
		sl = order.Spot - sl
	} else {
		tp = order.Spot - tp
		sl = order.Spot + sl
	}
	netPL := simulate(order.Spot, sl, tp, stockData[symbol], idx, order.OrderType)
	netPL = netPL * float64(order.Quantity)
	Amount += netPL

	if netPL > 0 {
		tradeReport.trade++
		tradeReport.profit += netPL
		tradeReport.profitCount++
		count = 0
		tradeReport.amount += netPL

	}
	if netPL < 0 {
		tradeReport.trade++
		tradeReport.loss += netPL
		tradeReport.lossCount++
		count++
		tradeReport.amount += netPL
		tradeReport.maxContinousloss = math.Max(tradeReport.maxContinousloss, count)
	}

	amountChange = append(amountChange, tradeReport.amount)
	trades = append(trades, tradeReport.trade)

}

func TrendFollowingRsi(data strategy.DataWithIndicators, token, symbol, username string, client *smartapigo.Client, idx int, rsiPeriod int, isEma bool) strategy.ORDER {
	var ma5, ma8, ma13, ma21, ma3 float64
	if isEma {
		ma5 = data.Indicators["ema"+"5"][idx]
		ma8 = data.Indicators["ema"+"8"][idx]
		ma13 = data.Indicators["ema"+"13"][idx]
		ma21 = data.Indicators["ema"+"21"][idx]
		ma3 = data.Indicators["ema"+"3"][idx]
	} else {
		ma5 = data.Indicators["sma"+"5"][idx]
		ma8 = data.Indicators["sma"+"8"][idx]
		ma13 = data.Indicators["sma"+"13"][idx]
		ma21 = data.Indicators["sma"+"21"][idx]
		ma3 = data.Indicators["sma"+"3"][idx]
	}

	adx20 := data.Adx["Adx"+"14"]
	rsi := data.Indicators["rsi"+strconv.Itoa(rsiPeriod)]
	var order strategy.ORDER
	order.OrderType = "None"
	//log.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	rsiAvg5 := getAvg(rsi, 3)
	rsiavg8 := getAvg(rsi, 5)
	adxAvg5 := getAvg(adx20.Adx, 5)
	adxAvg8 := getAvg(adx20.Adx, 8)
	//atr14 := data.Indicators["atr"+"14"][idx]
	var tempOrder strategy.ORDER
	tempOrder.OrderType = "BUY"

	//high, low := GetDC(data.Data, idx-1)

	//log.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if data.Data[idx].Low > ma8 && adxAvg5 > adxAvg8 && adx20.Adx[idx] >= 25 && adx20.PlusDi[idx] > adx20.MinusDi[idx] && ma3 > ma5 && ma5 > ma8 && ma8 > ma13 && ma21 < ma13 && rsi[idx] > 55 && rsi[idx] < 65 && rsiAvg5 > rsiavg8 {
		order = strategy.ORDER{
			Spot:      data.Data[idx].High + 0.05,
			Sl:        20,
			Tp:        60,
			Quantity:  calculatePosition(data.Data[idx].High),
			OrderType: "BUY",
			Token:     "99926000",
		}

		//} else if data.Data[idx].High < ma8 && adxAvg5 > adxAvg8 && adx20.Adx[idx] >= 20 && adx20.PlusDi[idx] < adx20.MinusDi[idx] && ma3 < ma5 && ma5 < ma8 && ma8 < ma13 && ma21 > ma13 && rsi[idx] < 40 && rsi[idx] > 30 && rsiAvg5 < rsiavg8 {
		//	order = strategy.ORDER{
		//		Spot:      data.Data[idx].Low - 0.05,
		//		Sl:        20,
		//		Tp:        60,
		//		Quantity:  calculatePosition(data.Data[idx].High),
		//		OrderType: "SELL",
		//	}

	}
	order.Score = strategy.CaluclateScore(&data, order)
	order.Symbol = symbol
	order.Token = token

	return order
}

func calculatePosition(price float64) int {
	//tempAmount := Amount
	//quantity := tempAmount / price
	//Amount = Amount - (quantity * price) - 200
	return 50
}

func simulate(spot, sl, tp float64, data []smartapigo.CandleResponse, idx *int, orderType string) float64 {
	trailingStopLoss := sl
	for *idx < len(data)-1 {
		currentTime := data[*idx].Timestamp
		compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 15, 0, 0, currentTime.Location())
		if currentTime.After(compareTime) {
			if orderType == "BUY" {
				return data[*idx].Close - spot
			} else {
				return spot - data[*idx].Close
			}
		}
		if orderType == "BUY" {

			if data[*idx].Close >= tp {
				return tp - spot
			}
			if data[*idx].Close <= sl {
				return sl - spot
			}
			if spot < data[*idx].Close {
				trailingStopLoss += float64(int((data[*idx].Close - spot) / 2))
				//sl = trailingStopLoss
			}
		}

		if orderType == "SELL" {
			if data[*idx].Close >= sl {
				return spot - sl
			}
			if data[*idx].Close <= tp {
				return spot - tp
			}
			if spot > data[*idx].Close {
				trailingStopLoss -= float64(int((spot - data[*idx].Close) / 2))
				//sl = trailingStopLoss
			}
		}
		*idx++

	}
	return 0
}

func initTrade() kpi {
	return kpi{
		0,
		0,
		0,
		0,
		0,
		0,
		Amount,
	}
}

func printCurrentTradeReport() {
	log.Println("********************|| TRADE REPORT ||************************")
	log.Printf("Trade: %v\n", tradeReport.trade)
	log.Printf("Profit: %v\n", tradeReport.profit)
	log.Printf("Loss: %v\n", tradeReport.loss)
	log.Printf("MaxContinousLoss: %v\n", tradeReport.maxContinousloss)
	log.Printf("ProfitCount: %v\n", tradeReport.profitCount)
	log.Printf("LossCount: %v\n", tradeReport.lossCount)
	log.Printf("Amount: %v\n", tradeReport.amount)
	log.Printf("TotalAmount: %v\n", Amount)
}

func getAvg(data []float64, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	sum := 0.0
	for i := len(data) - 1; i >= len(data)-period; i-- {
		sum += data[i]
	}
	return sum / float64(period)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func GetDC(data []smartapigo.CandleResponse, idx int) (float64, float64) {
	high, low := 0.0, 100000000.0
	for i := idx; i > idx-20; i-- {
		high = math.Max(high, data[i].High)
		low = math.Min(low, data[i].Low)
	}
	return high, low
}
