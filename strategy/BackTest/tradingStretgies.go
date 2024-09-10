package BackTest

import (
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"strconv"
	"time"
)

func Execute(symbol, stockToken string, client *smartapigo.Client, userName string, idx *int, dcPeriod int, isEma bool) strategy.ORDER {
	var order strategy.ORDER
	order.OrderType = "None"
	order.Symbol = symbol
	order.Token = stockToken
	data := dataWithIndicatorsMap[stockToken]
	currentTime := data.Data[*idx].Timestamp
	compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 14, 00, 0, 0, currentTime.Location())
	if len(dataWithIndicatorsMap[stockToken].Data) == 0 || len(dataWithIndicatorsMap[stockToken].Data) <= *idx || currentTime.After(compareTime) {
		return order
	}
	_, low := GetORBRange(dataWithIndicatorsMap[stockToken], idx, 14)

	ma5 := data.Indicators["sma"+"5"][*idx]
	ma8 := data.Indicators["sma"+"8"][*idx]
	ma13 := data.Indicators["sma"+"13"][*idx]
	ma21 := data.Indicators["sma"+"21"][*idx]
	ema8 := data.Indicators["ema"+"14"][*idx]
	rsi := data.Indicators["rsi"+"20"]
	adx14 := data.Adx["Adx"+"20"]
	rsiAvg3 := getAvg(rsi, 3)
	rsiavg8 := getAvg(rsi, 8)

	if rsi[*idx] > 85 && rsi[*idx] < 25 && adx14.Adx[*idx] <= 35 {
		return order
	}

	obv := strategy.CalculateOBV(data)
	//log.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	//log.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	//if strategy.IsOBVIncreasing(obv) && data.Data[*idx].Low > ema8 && data.Data[*idx-1].Close >= high && data.Data[*idx].Close > strategy.GetVwap(data.Data, 14) && ma5 > ma8 && ma8 > ma13 && ma21 < ma13 && rsiAvg3 > rsiavg8 && rsi[*idx] > 65 {
	//	order = strategy.ORDER{
	//		Spot:      data.Data[*idx].Close + 0.05,
	//		Sl:        data.Data[*idx].Close * 0.01,
	//		Tp:        int(data.Data[*idx].High * 0.02),
	//		Quantity:  20,
	//		OrderType: "BUY",
	//	}
	if strategy.IsOBVDecreasing(obv) && data.Data[*idx].Low < ema8 && data.Data[*idx-1].Close <= low && data.Data[*idx].Close < strategy.GetVwap(data.Data, 14) && ma5 < ma8 && ma8 < ma13 && ma21 > ma13 && rsiAvg3 < rsiavg8 && rsi[*idx] < 50 {
		order = strategy.ORDER{
			Spot:      data.Data[*idx].Close - 0.05,
			Sl:        data.Data[*idx].Close * 0.01,
			Tp:        int(data.Data[*idx].Low * 0.03),
			Quantity:  200,
			OrderType: "SELL",
		}

	}
	order.Symbol = symbol
	order.Token = stockToken
	if order.OrderType != "None" {
		log.Printf("\n\n")
		log.Println("Order", order)
		log.Println("timeStamp", data.Data[*idx].Timestamp)
		log.Println("Data= ", data.Data[*idx].Low, "Low =", low, " previos Close = ", data.Data[*idx-1].Close, " Vawap = ", strategy.GetVwap(data.Data, 14), " Aligator = ", ma5, " ", ma8, " ", ma13, " ", ma21, " rsi = ", rsi[*idx])
	}
	return order
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

func ExecuteForIndex(symbol, stockToken string, client *smartapigo.Client, userName string, idx *int, dcPeriod int, isEma bool) strategy.ORDER {
	var order strategy.ORDER
	order.OrderType = "None"
	order.Symbol = symbol
	order.Token = stockToken
	data := dataWithIndicatorsMap[stockToken]
	currentTime := data.Data[*idx].Timestamp
	compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 14, 00, 0, 0, currentTime.Location())
	if len(dataWithIndicatorsMap[stockToken].Data) == 0 || len(dataWithIndicatorsMap[stockToken].Data) <= *idx || currentTime.After(compareTime) {
		return order
	}
	high, low := GetORBRange(dataWithIndicatorsMap[stockToken], idx, 10)

	//ma1 := data.Indicators["sma"+"1"][*idx]
	ma3 := data.Indicators["sma"+"3"][*idx]
	ma5 := data.Indicators["sma"+"5"][*idx]
	ma8 := data.Indicators["sma"+"8"][*idx]
	ma13 := data.Indicators["sma"+"13"][*idx]
	ma21 := data.Indicators["sma"+"21"][*idx]
	ema8 := data.Indicators["ema"+"14"][*idx]
	rsi := data.Indicators["rsi"+"20"]
	adx14 := data.Adx["Adx"+"20"]
	rsiAvg3 := getAvg(rsi, 3)
	rsiavg8 := getAvg(rsi, 8)

	if rsi[*idx] > 80 && rsi[*idx] < 30 && adx14.Adx[*idx] <= 35 {
		return order
	}

	//obv := strategy.CalculateOBV(data)
	//log.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	//log.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if data.Data[*idx].Low > ema8 && data.Data[*idx].Close >= high && ma3 > ma5 && ma5 > ma8 && ma8 > ma13 && ma21 < ma13 && rsiAvg3 > rsiavg8 && rsi[*idx] > 50 {
		order = strategy.ORDER{
			Spot:      data.Data[*idx].Close + 0.05,
			Sl:        05,
			Tp:        25,
			Quantity:  15,
			OrderType: "BUY",
		}
	} else if data.Data[*idx].High < ema8 && data.Data[*idx].Close <= low && ma3 < ma5 && ma5 < ma8 && ma8 < ma13 && ma21 > ma13 && rsiAvg3 < rsiavg8 && rsi[*idx] < 50 {
		order = strategy.ORDER{
			Spot:      data.Data[*idx].Close - 0.05,
			Sl:        05,
			Tp:        25,
			Quantity:  15,
			OrderType: "SELL",
		}

	}
	order.Symbol = symbol
	order.Token = stockToken
	if order.OrderType != "None" {
		log.Printf("\n\n")
		log.Println("Order", order)
		log.Println("timeStamp", data.Data[*idx].Timestamp)
		log.Println("Trade Type: ", order.OrderType, " Data= ", data.Data[*idx].Low, "Low =", low, " High = ", high, " previos Close = ", data.Data[*idx-1].Close, " Aligator = ", ma5, " ", ma8, " ", ma13, " ", ma21, " rsi = ", rsi[*idx])
	}
	return order
}
