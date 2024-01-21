package Simulation

import (
	"database/sql"
	"fmt"
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"time"
)

var StockUniverse []string

func PrepareData(db *sql.DB, client *smartapigo.Client, token string, timeFrame, symbol string) {
	symbolToken := token
	tempTime := time.Now()
	toDate := tempTime.Format("2006-01-02 15:04")
	fromDate := tempTime.Add(time.Hour * 24 * -5).Format("2006-01-02 15:04")
	tempTime = tempTime.Add(time.Hour * 24 * -5)
	tempHistoryData, _ := client.GetCandleData(smartapigo.CandleParams{
		Exchange:    "NSE",
		SymbolToken: symbolToken,
		Interval:    timeFrame,
		FromDate:    fromDate,
		ToDate:      toDate,
	})
	if len(tempHistoryData) == 0 {
		return
	}
	if tempHistoryData[0].High < 700 && tempHistoryData[0].Close > 100 {
		query := `INSERT INTO "History"."Intraday" (token, symbol)
			VALUES ($1, $2)`
		_, err := db.Exec(query, symbolToken, symbol)
		if err != nil {
			fmt.Println("Error executing INSERT query:", err)
			return
		}
	}

	//for _, tempHistoryData := range tempHistoryData {
	//	// Prepare the INSERT statement
	//	insertQuery := `
	//            INSERT INTO "History"."OHLCData" (id, timeframeinseconds, open, high, low, close, timestamp, volume)
	//            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	//
	//	// Execute the INSERT statement
	//	_, err := db.Exec(insertQuery,
	//		symbolToken,
	//		300,
	//		tempHistoryData.Open,
	//		tempHistoryData.High,
	//		tempHistoryData.Low,
	//		tempHistoryData.Close,
	//		tempHistoryData.Timestamp,
	//		tempHistoryData.Volume,
	//	)
	//
	//	if err != nil {
	//		fmt.Println("Error executing INSERT query:", err)
	//		return
	//	}
	//}

}

func CollectData(db *sql.DB, client *smartapigo.Client) {
	stocks := GetStockSymbolList()
	for i := range stocks {
		token := strategy.GetToken(stocks[i].Symbol, "NSE")
		stockName := strategy.GetStockName(token)
		fmt.Printf("stock name = %v ", stockName)
		PrepareData(db, client, token, "FIVE_MINUTE", stocks[i].Symbol)
	}
}
