package strategy

//
//import (
//	"fmt"
//	smartapigo "github.com/TredingInGo/smartapi"
//	"math"
//)
//
//type ORDER struct {
//	Spot      float64
//	Tp        int
//	Sl        int
//	Quantity  int
//	OrderType string
//	Score     float64
//	Symbol    string
//	Token     string
//}
//
////func ReversalSystem1(data []smartapigo.CandleResponse, idx int, token string) ORDER {
////
////	return order
////}
//
//func TrendFollwoCrossSystemSMA(data []smartapigo.CandleResponse, idx int, token string, sma1, sma2 int) ORDER {
//	sma5 := sma[token+"5"][idx]
//	sma3 := sma[token+"3"][idx]
//	sma8 := sma[token+"8"][idx]
//	adx14 := adx[token]
//	rsi := rsi[token]
//	var order ORDER
//
//	if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma3 > sma5 && sma5 > sma8 && sma8 > sma[token+"13"][idx] && sma[token+"13"][idx] > sma[token+"21"][idx] && rsi[idx] < 75 && rsi[idx] > 60 && rsi[idx-2] < rsi[idx] {
//		log.Printf("order placed: trend following adx = %v \n", adx14.Adx[idx])
//		order = ORDER{
//			Spot:      data[idx].High + 0.05,
//			Sl:        int(math.Max(data[idx].High*0.005, 1.0)),
//			Tp:        int(data[idx].High * 0.01),
//			Quantity:  CalculatePositionSize(data[idx].High, data[idx].High-data[idx].High*0.01),
//			OrderType: "BUY",
//		}
//	} else if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] < adx14.MinusDi[idx] && sma3 < sma5 && sma5 < sma8 && sma8 < sma[token+"13"][idx] && sma[token+"13"][idx] < sma[token+"21"][idx] && rsi[idx] < 40 && rsi[idx] > 30 && rsi[idx-2] > rsi[idx] {
//		log.Printf("order placed: trend following %v\n", adx14.Adx[idx])
//		order = ORDER{
//			Spot:      data[idx].Low - 0.05,
//			Sl:        int(math.Max(data[idx].High*0.005, 1.0)),
//			Tp:        int(data[idx].Low * 0.01),
//			Quantity:  CalculatePositionSize(data[idx].High, data[idx].High-data[idx].High*0.01),
//			OrderType: "SELL",
//		}
//	}
//
//	return order
//}
//
//func RSIPlus44EMA(data []smartapigo.CandleResponse, idx int, token string) ORDER {
//
//	ema44High := ema[token+"High44"]
//	ema44Low := ema[token+"Low44"]
//	rsi := rsi[token]
//	var order ORDER
//	if ema44High[idx] < data[idx].Close && rsi[idx-1] < 60 && rsi[idx] > 60 {
//		order = ORDER{
//			Spot:      data[idx].High,
//			Sl:        int(ema44Low[idx]),
//			Tp:        int(data[idx].High + 2*(data[idx].High-ema44Low[idx])),
//			Quantity:  CalculatePositionSize(data[idx].High, data[idx].High+2*(data[idx].High-ema44Low[idx])),
//			OrderType: "BUY",
//		}
//	}
//	return order
//}
//
//func LstmPlusStochStratgy(candles []smartapigo.CandleResponse, k, d float64, atr float64, token string) {
//	predictions := GetDirections(candles, token+"-5LSTM")
//	//log.Printf("close: %v,  k: %v, d: %v, atr: %v, prediction: %v", candles[len(candles)-1].Close, k, d, atr, predictions[len(predictions)-1])
//	if predictions[len(predictions)-1] > 0.7 && k < 30 && d < 20 && atr > 2.5 {
//		KPI.trade++
//		price := candles[len(candles)-1].Close
//		tp := price + atr
//		sl := price - ((tp - price) / 2.0)
//		quantity := 10.0
//		if order.flag == false {
//			order.spot = price
//			order.tp = tp
//			order.sl = sl
//			order.qty = quantity
//			order.flag = true
//			order.orderType = "BUY"
//		}
//	}
//	if predictions[len(predictions)-1] < 0.3 && k > 85 && d > 80 && atr > 2.5 {
//		KPI.trade++
//		price := candles[len(candles)-1].Close
//		tp := price - atr
//		sl := price + ((price - tp) / 2.0)
//		quantity := 10.0
//		if order.flag == false {
//			order.spot = price
//			order.tp = tp
//			order.sl = sl
//			order.qty = quantity
//			order.flag = true
//			order.orderType = "SELL"
//		}
//	}
//
//}
//func orderSimulation(ltp float64) {
//	if order.flag == false {
//		return
//	}
//	if order.orderType == "BUY" {
//		if ltp >= order.tp {
//			KPI.profit += order.tp - order.spot
//			KPI.profitCount++
//			order.flag = false
//			count = 0.0
//			log.Println(KPI)
//			return
//		}
//		if ltp <= order.sl {
//			KPI.loss += order.sl - order.spot
//			KPI.lossCount++
//			order.flag = false
//			count++
//			KPI.maxContinousloss = math.Max(KPI.maxContinousloss, count)
//			log.Println(KPI)
//			return
//		}
//	}
//	if order.orderType == "SELL" {
//		if ltp <= order.tp {
//			KPI.profit += order.spot - order.tp
//			KPI.profitCount++
//			order.flag = false
//			count = 0.0
//			log.Println(KPI)
//			return
//		}
//		if ltp >= order.sl {
//			KPI.loss += order.spot - order.sl
//			KPI.lossCount++
//			order.flag = false
//			count++
//			KPI.maxContinousloss = math.Max(KPI.maxContinousloss, count)
//			log.Println(KPI)
//			return
//		}
//	}
//
//}
