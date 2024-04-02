package BackTest

import (
	"database/sql"
	"fmt"
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
	"sort"
	"strconv"
	"time"
)

const (
	worker = 1
)

var stockData = make(map[string][]smartapigo.CandleResponse)
var Amount = 100000.0

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
	stockList := strategy.LoadStockList(db)
	populateStockData(stockList, client)
	//rsi, ema,
	maxProfit := initTrade()
	rsiVal := 5
	ema := false
	for rsi := 5; rsi <= 20; rsi++ {
		tradeReport = initTrade()
		Amount = 100000
		executeBacktest(client, stockList, rsi, false)
		fmt.Printf("\n rsi = %v, isEma = false, \n", rsi)
		printCurrentTradeReport()
		if tradeReport.profit > maxProfit.profit {
			maxProfit = tradeReport
			rsiVal = rsi
			ema = false
		}
		Amount = 100000
		tradeReport = initTrade()
		fmt.Printf("\n rsi = %v, isEma = true, \n", rsi)
		executeBacktest(client, stockList, rsi, true)
		if tradeReport.profit > maxProfit.profit {
			maxProfit = tradeReport
			rsiVal = rsi
			ema = true
		}
		printCurrentTradeReport()
	}

	fmt.Println("********************|| FINAL TRADE REPORT ||************************")
	tradeReport = maxProfit
	fmt.Println("Rsi: ", rsiVal, " isEma ", ema)
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
		fmt.Println(count, "data Population done for ", stock.Token)
		count++

	}

}

// populate OHLC data of each stock of 1000 days.

func populateStockTick(client *smartapigo.Client, symbolToken string, timeFrame string) {

	tempTime := time.Now()
	var candles []smartapigo.CandleResponse
	for i := 0; i < 3; i++ {
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
	orders := []*strategy.ORDER{}
	filteredStocks := []*strategy.ORDER{}
	//inp := make(chan *strategy.EligibleStockParam, 1000)
	//out := make(chan *strategy.ORDER, 1000)
	//wg := sync.WaitGroup{}
	//
	//go func() {
	//	for i := 0; i < worker; i++ {
	//		wg.Add(1)
	//		go func() {
	//			defer wg.Done()
	//			for param := range inp {
	//				order := Execute(param.Symbol, param.Token, client, param.UserName, idx)
	//				if order != nil {
	//					out <- order
	//				}
	//			}
	//		}()
	//	}
	//}()
	//
	//go func() {
	//	for _, stock := range stocks {
	//		inp <- &strategy.EligibleStockParam{
	//			Symbols:  strategy.Symbols{Symbol: stock.Token, Token: stock.Symbol},
	//			UserName: userName,
	//		}
	//	}
	//	close(inp)
	//}()
	//
	//// close output channel after all the workers are done
	//go func() {
	//	wg.Wait()
	//	close(out)
	//}()
	//
	//for order := range out {
	//	orders = append(orders, order)
	//}

	//start := time.Now()
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

	//fmt.Println("Time to filter stocks ", time.Since(start))

	//start = time.Now()
	for _, stock := range filteredStocks {
		order := Execute(stock.Symbol, stock.Token, client, userName, idx, rsiPeriod, isEma)
		if order != nil {
			orders = append(orders, order)
		}
	}

	//fmt.Println("Time to get orders ", time.Since(start))

	return orders
}

func Execute(symbol, stockToken string, client *smartapigo.Client, userName string, idx *int, rsiPeriod int, isEma bool) *strategy.ORDER {
	if len(dataWithIndicatorsMap[stockToken].Data) == 0 || len(dataWithIndicatorsMap[stockToken].Data) <= *idx {
		return nil
	}

	order := TrendFollowingRsi(dataWithIndicatorsMap[stockToken], stockToken, symbol, userName, client, *idx, rsiPeriod, isEma)
	if order.OrderType == "None" || order.Quantity < 1 {
		return nil
	}

	return &order
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
	//fmt.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	rsiAvg5 := getAvg(rsi, 3)
	rsiavg8 := getAvg(rsi, 5)
	adxAvg5 := getAvg(adx20.Adx, 5)
	adxAvg8 := getAvg(adx20.Adx, 8)
	//atr14 := data.Indicators["atr"+"14"][idx]
	var tempOrder strategy.ORDER
	tempOrder.OrderType = "BUY"

	//high, low := GetDC(data.Data, idx-1)

	//fmt.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if data.Data[idx].Low > ma8 && adxAvg5 > adxAvg8 && adx20.Adx[idx] >= 25 && adx20.PlusDi[idx] > adx20.MinusDi[idx] && ma3 > ma5 && ma5 > ma8 && ma8 > ma13 && ma21 < ma13 && rsi[idx] > 55 && rsi[idx] < 65 && rsiAvg5 > rsiavg8 {
		order = strategy.ORDER{
			Spot:      data.Data[idx].High + 0.05,
			Sl:        int(data.Data[idx].High * 0.02),
			Tp:        int(data.Data[idx].High * 0.02),
			Quantity:  calculatePosition(data.Data[idx].High),
			OrderType: "BUY",
		}

	} else if data.Data[idx].High < ma8 && adxAvg5 > adxAvg8 && adx20.Adx[idx] >= 20 && adx20.PlusDi[idx] < adx20.MinusDi[idx] && ma3 < ma5 && ma5 < ma8 && ma8 < ma13 && ma21 > ma13 && rsi[idx] < 40 && rsi[idx] > 30 && rsiAvg5 < rsiavg8 {
		order = strategy.ORDER{
			Spot:      data.Data[idx].Low - 0.05,
			Sl:        int(data.Data[idx].High * 0.02),
			Tp:        int(data.Data[idx].High * 0.02),
			Quantity:  calculatePosition(data.Data[idx].High),
			OrderType: "SELL",
		}

	}
	order.Score = strategy.CaluclateScore(&data, order)
	order.Symbol = symbol
	order.Token = token

	return order
}

func calculatePosition(price float64) int {
	tempAmount := Amount - 500
	quantity := tempAmount / price
	//Amount = Amount - (quantity * price) - 200
	return int(quantity) * 5
}

func simulate(spot, sl, tp float64, data []smartapigo.CandleResponse, idx *int, orderType string) float64 {
	trailingStopLoss := sl
	for *idx < len(data)-1 {
		currentTime := data[*idx].Timestamp
		compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 20, 0, 0, currentTime.Location())
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
	fmt.Println("********************|| TRADE REPORT ||************************")
	fmt.Printf("Trade: %v\n", tradeReport.trade)
	fmt.Printf("Profit: %v\n", tradeReport.profit)
	fmt.Printf("Loss: %v\n", tradeReport.loss)
	fmt.Printf("MaxContinousLoss: %v\n", tradeReport.maxContinousloss)
	fmt.Printf("ProfitCount: %v\n", tradeReport.profitCount)
	fmt.Printf("LossCount: %v\n", tradeReport.lossCount)
	fmt.Printf("Amount: %v\n", tradeReport.amount)
	fmt.Printf("TotalAmount: %v\n", Amount)
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
