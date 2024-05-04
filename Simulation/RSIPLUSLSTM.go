package Simulation

//
//import (
//	"fmt"
//	"github.com/TredingInGo/AutomationService/strategy"
//	smartapigo "github.com/TredingInGo/smartapi"
//	"math"
//)
//
//var rsi []float64
//var kpiFor15083 kpi
//
//func RunStrategyRSI(data []smartapigo.CandleResponse) {
//	for i := 0; i < 13; i++ {
//		lstmValue = append(lstmValue, 0.5)
//	}
//	predictedValues := strategy.GetDirections(data, token+"-5LSTM")
//	lstmValue = append(lstmValue, predictedValues...)
//	priceArray := strategy.GetClosePriceArray(data)
//	strategy.CalculateRsi(priceArray, 14, token)
//	rsi = strategy.GetRsi(token)
//	strategy.CalculateAtr(data, 14, token)
//	atr = strategy.GetAtrArray(token)
//	bestProfitFactor := -math.MaxFloat64
//	var bestParams Params
//	var bestResult Result
//
//	// Define ranges and steps for each parameter
//	rsiThresholds := []float64{20, 30, 35}
//	atrThresholds := []float64{2.0, 2.5, 3.0, 4.0}
//	buyThresholds := []float64{0.6, 0.7, 0.8, 0.5}
//	sellThresholds := []float64{0.2, 0.3, 0.4, 0.1}
//
//	// Iterate over all combinations of parameters
//	for _, k := range rsiThresholds {
//		for _, atr := range atrThresholds {
//			for _, buy := range buyThresholds {
//				for _, sell := range sellThresholds {
//					params := Params{
//						KThreshold:    k,
//						ATRThreshold:  atr,
//						BuyThreshold:  buy,
//						SellThreshold: sell,
//					}
//
//					result := Lstm(data, params)
//					if result.ProfitFactor > bestProfitFactor {
//						bestProfitFactor = result.ProfitFactor
//						bestParams = params
//						bestResult = result
//					}
//
//					log.Printf("Params: %+v, Result: %+v\n", params, result)
//				}
//			}
//		}
//	}
//
//	log.Println("Best Parameters:")
//	log.Printf("Params: %+v, Result: %+v\n Kpi %v", bestParams, bestResult, kpiFor15083)
//}
//
//func Lstm(data []smartapigo.CandleResponse, params Params) Result {
//
//	kpiFor15083 = kpi{
//		0,
//		0.0,
//		0.0,
//		0.0,
//		0,
//		0,
//	}
//
//	for i := 13; i < len(data); {
//		//tempWindow := data[i-13 : i+1]
//		signal := "None"
//		direction := lstmValue[i]
//
//		if direction > params.BuyThreshold {
//			signal = "BUY"
//		}
//		if direction < params.SellThreshold {
//			signal = "SELL"
//		}
//
//		trade, idx := startTestingRsi(data, atr[i], rsi[i], signal, i+1, params)
//		if trade < 0 {
//			kpiFor15083.loss += -trade
//			kpiFor15083.trade++
//			kpiFor15083.lossCount++
//		}
//		if trade > 0 {
//			kpiFor15083.profit += trade
//			kpiFor15083.trade++
//			kpiFor15083.profitCount++
//		}
//		i = idx
//	}
//
//	result := Result{
//		Params:       params,
//		TradeCount:   kpiFor15083.trade,
//		Profit:       kpiFor15083.profit,
//		Loss:         kpiFor15083.loss,
//		ProfitFactor: kpiFor15083.profit / kpiFor15083.loss,
//	}
//	return result
//}
//
//func startTestingRsi(data []smartapigo.CandleResponse, atr float64, rsi float64, signal string, idx int, params Params) (float64, int) {
//	if rsi < params.KThreshold && signal == "BUY" {
//		if atr > params.ATRThreshold {
//			price := data[idx].Close
//			tp := price + price*0.04
//			sl := price - price*0.02
//			quantity := 100
//			return simulateBuyorder(data, quantity, idx, price, sl, tp), idx + 1
//		}
//	}
//	return 0.0, idx
//}
