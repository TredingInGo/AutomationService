package BackTest

import (
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"sort"
	"time"
)

func populateStockData(stockList []strategy.Symbols, client *smartapigo.Client) {
	count := 1
	for _, stock := range stockList {
		populateStockTick(client, stock.Symbol, "TEN_MINUTE")
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

func populateStockTick(client *smartapigo.Client, symbolToken string, timeFrame string) {

	tempTime := time.Now()
	var candles []smartapigo.CandleResponse
	for i := 0; i < 10; i++ {
		toDate := tempTime.Format("2006-01-02 15:04")
		fromDate := tempTime.Add(time.Hour * 24 * -60).Format("2006-01-02 15:04")
		tempTime = tempTime.Add(time.Hour * 24 * -60)

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
