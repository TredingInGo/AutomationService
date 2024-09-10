package BackTest

import (
	"database/sql"
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"math"
	"sort"
)

const (
	worker = 1
)

var stockData = make(map[string][]smartapigo.CandleResponse)
var Amount = 1000000.0

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

	var stockList []strategy.Symbols
	for i := 0; i < 1; i++ {
		stockList = append(stockList, strategy.Symbols{"99926009", "BANKNIFTY"})

	}
	populateStockData(stockList, client)

	//rsi, ema,
	maxProfit := initTrade()
	Dc := 0

	for i := 20; i <= 20; i++ {
		Amount = 1000000
		tradeReport = initTrade()
		executeBacktest(client, stockList, i, false)
		//fmt.Println("Current Dc = ", i)
		//printCurrentTradeReport()
		if tradeReport.profit > maxProfit.profit {
			maxProfit = tradeReport
			Dc = i
		}
	}

	log.Println("********************|| FINAL TRADE REPORT ||************************")
	tradeReport = maxProfit
	log.Println("DC: ", Dc)
	printCurrentTradeReport()
	//plotGraph(amountChange, trades)
}

// populate OHLC data of each stock of 1000 days.

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
		//PlaceOrder(eligibleStocks[0], eligibleStocks[0].Token, &idx)
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

func getEligibleStocks(stocks []strategy.Symbols, client *smartapigo.Client, userName string, idx *int, rsiPeriod int, isEma bool) []strategy.ORDER {

	filteredStocks := []strategy.ORDER{}

	for _, stock := range stocks {
		param := strategy.EligibleStockParam{
			Symbols:  strategy.Symbols{Symbol: stock.Symbol, Token: stock.Token},
			UserName: userName,
		}
		order := ExecuteForIndex(param.Token, param.Symbol, client, param.UserName, idx, rsiPeriod, isEma)
		if order.OrderType != "None" {
			PlaceOrder(order, order.Token, idx)
		}
	}

	return filteredStocks
}

func PlaceOrder(order strategy.ORDER, symbol string, idx *int) {
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
		log.Println("Profit ")
		tradeReport.trade++
		tradeReport.profit += netPL
		tradeReport.profitCount++
		count = 0
		tradeReport.amount += netPL

	}
	if netPL < 0 {
		log.Println("Loss")
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
	log.Printf("Amount: %v\n", int64(tradeReport.amount))
	log.Printf("TotalAmount: %v\n", Amount)
}
