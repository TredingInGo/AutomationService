package Simulation

import (
	"database/sql"
	smartapigo "github.com/TredingInGo/smartapi"
	_ "github.com/lib/pq"
	"log"
	"time"
)

type tradeResult struct {
	amount       float64
	profitCount  int64
	lossCount    int64
	totalTrade   int64
	maxAmount    float64
	minAmount    float64
	stockSymbol  int64
	timeDuration int64
	smaLow       int
	smaHigh      int
	rsiPeriod    int
}
type tradeReport struct {
	amount      float64
	profit      float64
	loss        float64
	stratgyName string
	entry       float64
	tp          float64
	sl          float64
}
type tradeRecord struct {
	entryTimestamp string
	entryPrice     float64 // New field to store entry price
	exitTimestamp  string
	exitPrice      float64 // New field to store exit price
	profit         bool
}

type OHLC struct {
	Timestamp        string  `json:"timestamp"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Close            float64 `json:"close"`
	Volume           int     `json:"volume"`
	ID               int64   `json:"id"`
	TimeFrameSeconds int64   `json:"timeFrameSeconds"`
}

var amount float64
var profitCount int64
var lossCount int64
var totalTrade int64
var maxAmount float64
var minAmount float64
var trades []tradeRecord
var trade []tradeReport

func DoBackTest(db *sql.DB) {
	//data := GetData(db, "13061")
	//dataToTest := getDataiInCandleResponseFormate(data)
	////backTestSystems(dataToTest, "13061")
	//RunStrategyRSI(dataToTest[len(dataToTest)-7500:])
	//RunStrategy(dataToTest)

}

func GetData(db *sql.DB, token string) []OHLC {
	rows, err := db.Query(`SELECT * FROM "History"."OHLCData" WHERE id = $1 AND timeframeinseconds = 300  order by timestamp`, token)
	if err != nil {
		log.Fatalf("Error querying data from the table: %v", err)
	}
	defer rows.Close()

	var ohlcDataArray []OHLC

	// Iterate through the rows and populate the array
	for rows.Next() {
		var ohlc OHLC
		err := rows.Scan(
			&ohlc.ID,
			&ohlc.TimeFrameSeconds,
			&ohlc.Open,
			&ohlc.High,
			&ohlc.Low,
			&ohlc.Close,
			&ohlc.Timestamp,
			&ohlc.Volume,
		)
		if err != nil {
			log.Fatal(err)
		}
		ohlcDataArray = append(ohlcDataArray, ohlc)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return ohlcDataArray

}

func SaveTradeReport(trades []tradeReport, db *sql.DB) {
	for i := 0; i < len(trades); i++ {
		insertSQL := `
		INSERT INTO "History"."TradeReport" (amount, profit, loss, strategyName, entry, tp, sl)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;`
		_, err := db.Exec(insertSQL,
			trades[i].amount,
			trades[i].profit,
			trades[i].loss,
			trades[i].stratgyName,
			trades[i].entry,
			trades[i].tp,
			trades[i].sl,
		)

		if err != nil {
			log.Fatal(err)
		}

		//log.Printf("Inserted row with ID: %d\n", insertedID)
	}
}
func getDataiInCandleResponseFormate(data []OHLC) []smartapigo.CandleResponse {
	dataSize := len(data)
	var dataToTest []smartapigo.CandleResponse
	layout := "2006-01-02 15:04:05-07:00" // This should match the format of your date string

	// Parse the date string into a time.Time variable
	for i := 0; i < dataSize; i++ {
		dateTime, err := time.Parse(layout, data[i].Timestamp)
		log.Print(err)
		dataToTest = append(dataToTest, smartapigo.CandleResponse{
			Timestamp: dateTime,
			Open:      data[i].Open,
			High:      data[i].High,
			Low:       data[i].Low,
			Close:     data[i].Close,
			Volume:    data[i].Volume,
		})
	}
	return dataToTest
}
