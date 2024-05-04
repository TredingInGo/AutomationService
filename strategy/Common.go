package strategy

import (
	"database/sql"
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"math"
	"time"
)

func LoadStockList(db *sql.DB) []Symbols {
	rows, err := db.Query(`SELECT * FROM "History"."Intraday"`)
	if err != nil {
		log.Fatalf("Error querying data from the table: %v", err)
	}
	defer rows.Close()

	var stockList []Symbols

	// Iterate through the rows and populate the array
	for rows.Next() {
		var stock Symbols
		err := rows.Scan(
			&stock.Symbol,
			&stock.Token,
		)
		if err != nil {
			log.Fatal(err)
		}
		stockList = append(stockList, stock)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return stockList

}

func LoadStockListForSwing(db *sql.DB) []Symbols {
	rows, err := db.Query(`SELECT * FROM "History"."Swing"`)
	if err != nil {
		log.Fatalf("Error querying data from the table: %v", err)
	}
	defer rows.Close()

	var stockList []Symbols

	// Iterate through the rows and populate the array
	for rows.Next() {
		var stock Symbols
		err := rows.Scan(
			&stock.Symbol,
			&stock.Token,
		)
		if err != nil {
			log.Fatal(err)
		}
		stockList = append(stockList, stock)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return stockList

}

func GetStockTick(client *smartapigo.Client, symbolToken string, timeFrame string, exchange string) []smartapigo.CandleResponse {
	tempTime := time.Now()
	toDate := tempTime.Format("2006-01-02 15:04")
	fromDate := tempTime.Add(time.Hour * 24 * -20).Format("2006-01-02 15:04")
	tempTime = tempTime.Add(time.Hour * 24 * -20)

	// set timeout
	client.SetTimeout(5 * time.Second)

	tempHistoryData, err := client.GetCandleData(smartapigo.CandleParams{
		Exchange:    exchange,
		SymbolToken: symbolToken,
		Interval:    timeFrame,
		FromDate:    fromDate,
		ToDate:      toDate,
	})
	for i := 0; i < 3; i++ {
		tempHistoryData, err = client.GetCandleData(smartapigo.CandleParams{
			Exchange:    exchange,
			SymbolToken: symbolToken,
			Interval:    timeFrame,
			FromDate:    fromDate,
			ToDate:      toDate,
		})
		if err == nil {
			return tempHistoryData
		}

	}

	return tempHistoryData
}

func GetStockTickForSwing(client *smartapigo.Client, symbolToken string, timeFrame string) []smartapigo.CandleResponse {
	if timeFrame == "FOUR_HOUR" {
		return combine(GetHistoryData(client, symbolToken, "ONE_HOUR"))
	}
	return GetHistoryData(client, symbolToken, timeFrame)

}

func GetHistoryData(client *smartapigo.Client, symbolToken string, timeFrame string) []smartapigo.CandleResponse {
	tempTime := time.Now()
	toDate := tempTime.Format("2006-01-02 15:04")
	fromDate := tempTime.Add(time.Hour * 24 * -200).Format("2006-01-02 15:04")
	tempTime = tempTime.Add(time.Hour * 24 * -200)
	tempHistoryData, _ := client.GetCandleData(smartapigo.CandleParams{
		Exchange:    "NSE",
		SymbolToken: symbolToken,
		Interval:    timeFrame,
		FromDate:    fromDate,
		ToDate:      toDate,
	})
	return tempHistoryData
}

func newCandleResponse() smartapigo.CandleResponse {
	return smartapigo.CandleResponse{
		Timestamp: time.Now(),
		Open:      0.0,
		High:      0.0,
		Low:       0.0,
		Close:     0.0,
		Volume:    0,
	}
}

func updateCandle(candle smartapigo.CandleResponse, data smartapigo.CandleResponse) smartapigo.CandleResponse {
	return smartapigo.CandleResponse{
		Timestamp: data.Timestamp,
		Open:      0.0,
		High:      math.Max(data.High, candle.High),
		Low:       math.Min(data.Low, candle.Low),
		Close:     data.Close,
		Volume:    0,
	}
}

func combine(data []smartapigo.CandleResponse) []smartapigo.CandleResponse {
	var fourHourData []smartapigo.CandleResponse
	tempCandle := newCandleResponse()
	volume := 0
	for i := 0; i < len(data); i++ {
		volume += data[i].Volume
		if i%4 == 0 {
			tempCandle = data[i]

		} else if i%4 == 3 || i == len(data)-1 {
			tempCandle = updateCandle(tempCandle, data[i])
			tempCandle.Volume = volume
			volume = 0
			fourHourData = append(fourHourData, tempCandle)
			tempCandle = newCandleResponse()
		} else {
			tempCandle.High = math.Max(data[i].High, tempCandle.High)
			tempCandle.Low = math.Min(data[i].Low, tempCandle.Low)
		}

	}
	return fourHourData
}
