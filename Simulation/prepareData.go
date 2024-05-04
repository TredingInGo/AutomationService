package Simulation

import (
	"database/sql"
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"strings"
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
	if tempHistoryData[0].Close > 50 {
		query := `INSERT INTO "History"."Swing" (token, symbol)
			VALUES ($1, $2)`
		_, err := db.Exec(query, symbolToken, symbol)
		if err != nil {
			log.Println("Error executing INSERT query:", err)
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
	//		log.Println("Error executing INSERT query:", err)
	//		return
	//	}
	//}

}

func CollectData(db *sql.DB, client *smartapigo.Client) {
	strategy.PopuletInstrumentsList()
	stocks := strategy.InstrumentLists
	for i := range stocks {
		Symbols := strings.Split(stocks[i].Symbol, "-")
		token := strategy.GetToken(Symbols[0], "NSE")
		stockName := strategy.GetStockName(token)
		log.Printf("stock name = %v ", stockName)
		PrepareData(db, client, token, "ONE_DAY", stocks[i].Symbol)
	}
}
