package strategy

import (
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"time"
)

func TrendFollowingRsi(data *DataWithIndicators, token, symbol, username string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	ma5 := data.Indicators["sma"+"5"][idx]
	ma8 := data.Indicators["sma"+"8"][idx]
	ma13 := data.Indicators["sma"+"13"][idx]
	ma21 := data.Indicators["sma"+"21"][idx]
	ema21 := data.Indicators["ema"+"14"][idx]
	rsi := data.Indicators["rsi"+"20"]
	adx14 := data.Adx["Adx"+"20"]
	rsiAvg3 := getAvg(rsi, 3)
	rsiavg8 := getAvg(rsi, 8)
	high, low := GetCustomDCRange(*data, idx, 14)
	currentTime := data.Data[idx].Timestamp
	compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 14, 00, 0, 0, currentTime.Location())
	var order ORDER
	order.OrderType = "None"
	order.Symbol = symbol
	order.Token = token
	obv := CalculateOBV(*data)

	if rsi[idx] > 85 || rsi[idx] < 25 || adx14.Adx[idx] <= 35 || currentTime.After(compareTime) {
		return order
	}

	//log.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	log.Printf("currentTime:%v, currentData:%v, adx = %v, high = %v, low = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v \n", time.Now(), data.Data[idx], adx14.Adx[idx], high, low, ma5, ma8, ma13, ma21, rsi[idx], username)
	//if data.Data[idx-1].Close > high && data.Data[idx-1].Low > ema21 && data.Data[idx].Close > GetVwap(data.Data, 14) && adx14.PlusDi[idx] > adx14.MinusDi[idx] && ma5 > ma8 && ma8 > ma13 && ma21 < ma13 && rsiAvg3 > rsiavg8 {
	//	order = ORDER{
	//		Spot:      data.Data[idx].High + 0.05,
	//		Sl:        (data.Data[idx].High * 0.01),
	//		Tp:        int(data.Data[idx].High * 0.02),
	//		Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
	//		OrderType: "BUY",
	//	}
	//
	//} else
	if IsOBVDecreasing(obv) && data.Data[idx].Low < ema21 && data.Data[idx-1].Close <= low && data.Data[idx].Close < GetVwap(data.Data, 14) && ma5 < ma8 && ma8 < ma13 && ma21 > ma13 && rsiAvg3 < rsiavg8 && rsi[idx] < 50 {
		order = ORDER{
			Spot:      data.Data[idx].Low - 0.05,
			Sl:        (data.Data[idx].Low * 0.01),
			Tp:        int(data.Data[idx].Low * 0.025),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "SELL",
		}

	}
	order.Score = CaluclateScore(data, order)
	return order
}

func DcForStocks(data *DataWithIndicators, token, symbol string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	rsi := data.Indicators["rsi"+"14"]
	adx := data.Adx["Adx20"].Adx
	ema21 := data.Indicators["ema"+"15"]
	var order ORDER
	order.OrderType = "None"
	high, low := GetDCRange(*data, idx)

	if high == 0.0 || low == 1000000.0 || rsi[idx] > 75 || rsi[idx] < 30 || adx[idx] < 25 {
		return order
	}

	if data.Data[idx].Close > high && ema21[idx] > GetVwap(data.Data, 14) {
		log.Println(" Buy Trade taken on Dc BreakOut:")
		order = ORDER{
			Spot:      data.Data[idx].Close + 0.05,
			Sl:        data.Data[idx].Close * 0.01,
			Tp:        int(data.Data[idx].Close * 0.02),
			Quantity:  CalculatePosition(data.Data[idx].Close+0.05, 5, client),
			OrderType: "BUY",
		}

	} else if data.Data[idx].Close < low && ema21[idx] < GetVwap(data.Data, 14) {
		log.Println(" SELL Trade taken on DC breakout ")
		order = ORDER{
			Spot:      data.Data[idx].Close - 0.05,
			Sl:        data.Data[idx].Close * 0.01,
			Tp:        int(data.Data[idx].Close * 0.02),
			Quantity:  CalculatePosition(data.Data[idx].Close+0.05, 5, client),
			OrderType: "SELL",
		}

	}
	order.Symbol = symbol
	order.Token = token

	return order
}
