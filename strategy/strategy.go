package strategy

import (
	"database/sql"
	"fmt"
	"github.com/TredingInGo/AutomationService/historyData"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/TredingInGo/smartapi/models"
	"math"
	"time"
)

type trade struct {
	spot      float64
	sl        float64
	tp        float64
	qty       float64
	orderType string
	flag      bool
}

type CandleResponse struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int
}
type OHLCData struct {
	Open  float64
	High  float64
	Low   float64
	Close float64
}

var tempOhlc = CandleResponse{
	Timestamp: time.Now(),
	Open:      0.0,
	High:      0.0,
	Low:       0.0,
	Close:     0.0,
	Volume:    0,
}

type kpi struct {
	trade            int
	profit           float64
	loss             float64
	maxContinousloss float64
	profitCount      float64
	lossCount        float64
}

var Amount float64
var t trade
var KPI kpi
var count float64

const (
	startTime = "09:15"
	endTime   = "15:30"
)

var (
	currTime = time.Now()
	baseTime = time.Date(currTime.Year(), currTime.Month(), currTime.Day(), 9, 0, 0, 0, time.Local)
)

type strategy struct {
	history  historyData.History
	pastData []smartapigo.CandleResponse

	LiveData    chan *models.SnapQuote
	chForCandle chan *models.SnapQuote
	db          *sql.DB
}

func New(history historyData.History, db *sql.DB) strategy {
	return strategy{
		history:     history,
		LiveData:    make(chan *models.SnapQuote, 100),
		chForCandle: make(chan *models.SnapQuote, 100),
		db:          db,
	}
}

var order trade

func (s *strategy) Algo(token string) {
	//fmt.Printf("Stock-- %v", GetStockName(token))
	go s.makeCandle(s.chForCandle, 300)
	//for data := range s.LiveData {
	//	if len(s.pastData) == 0 {
	//		s.fillPastData(data.TokenInfo.Token, "NSE", 15)
	//	}
	//
	//	s.chForCandle <- data
	//
	//	// some algo....
	//	candles := s.pastData
	//	PopulateIndicators(candles, token, "Dummy")
	//	fmt.Printf("candels %v data %v\n", candles[len(candles)-1], float64(data.LastTradedPrice)/100.0)
	//	//atr := GetAtrArray(token)
	//	sto := GetStoArray(token)
	//	LstmPlusStochStratgy(candles, sto[len(sto)-1].K, sto[len(sto)-1].D, atr[len(atr)-1], token)
	//	orderSimulation(float64(data.LastTradedPrice) / 100.0)
	//
	//	//s.oneRsStrategy(data)
	//	//s.Order(data)
	//	//fmt.Printf(" LiveData: ", float64(data.LastTradedPrice)/100.0)
	//}
}

func LstmPlusStochStratgy(candles []smartapigo.CandleResponse, k, d float64, atr float64, token string) {
	predictions := GetDirections(candles, token+"-5LSTM")
	//fmt.Printf("close: %v,  k: %v, d: %v, atr: %v, prediction: %v", candles[len(candles)-1].Close, k, d, atr, predictions[len(predictions)-1])
	if predictions[len(predictions)-1] > 0.7 && k < 30 && d < 20 && atr > 2.5 {
		KPI.trade++
		price := candles[len(candles)-1].Close
		tp := price + atr
		sl := price - ((tp - price) / 2.0)
		quantity := 10.0
		if order.flag == false {
			order.spot = price
			order.tp = tp
			order.sl = sl
			order.qty = quantity
			order.flag = true
			order.orderType = "BUY"
		}
	}
	if predictions[len(predictions)-1] < 0.3 && k > 85 && d > 80 && atr > 2.5 {
		KPI.trade++
		price := candles[len(candles)-1].Close
		tp := price - atr
		sl := price + ((price - tp) / 2.0)
		quantity := 10.0
		if order.flag == false {
			order.spot = price
			order.tp = tp
			order.sl = sl
			order.qty = quantity
			order.flag = true
			order.orderType = "SELL"
		}
	}

}
func orderSimulation(ltp float64) {
	if order.flag == false {
		return
	}
	if order.orderType == "BUY" {
		if ltp >= order.tp {
			KPI.profit += order.tp - order.spot
			KPI.profitCount++
			order.flag = false
			count = 0.0
			fmt.Println(KPI)
			return
		}
		if ltp <= order.sl {
			KPI.loss += order.sl - order.spot
			KPI.lossCount++
			order.flag = false
			count++
			KPI.maxContinousloss = math.Max(KPI.maxContinousloss, count)
			fmt.Println(KPI)
			return
		}
	}
	if order.orderType == "SELL" {
		if ltp <= order.tp {
			KPI.profit += order.spot - order.tp
			KPI.profitCount++
			order.flag = false
			count = 0.0
			fmt.Println(KPI)
			return
		}
		if ltp >= order.sl {
			KPI.loss += order.spot - order.sl
			KPI.lossCount++
			order.flag = false
			count++
			KPI.maxContinousloss = math.Max(KPI.maxContinousloss, count)
			fmt.Println(KPI)
			return
		}
	}

}
