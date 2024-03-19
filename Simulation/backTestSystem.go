package Simulation

import (
	"fmt"
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
)

type kpi struct {
	trade            int
	profit           float64
	loss             float64
	maxContinousloss float64
	profitCount      float64
	lossCount        float64
	amount           float64
}
type bestParams struct {
	sma1 int
	sma2 int
}

var best bestParams

func backTestSystems(data []smartapigo.CandleResponse, token string) {
	//strategy.PopulateIndicators(data, token)
	strategy.SetAmount(100000)
	values := strategy.HeikinAshi[token][len(strategy.HeikinAshi[token])-50:]
	fmt.Printf("%v", values)
	high := strategy.GetHighPriceArray(data)
	low := strategy.GetLowPriceArray(data)
	strategy.CalculateEma(high, 44)
	strategy.CalculateEma(low, 44)
	maxProfit := initTrade()
	for sma1 := 3; sma1 < 12; sma1++ {
		for sma2 := 12; sma2 < 30; sma2++ {
			count := 0.0
			tradeReport := initTrade()
			for i := 150; i < len(data); i++ {
				var price float64
				var idx int
				trade := strategy.TrendFollwoCrossSystemSMA(data, i, token, sma1, sma2)
				//trade := strategy.RSIPlus44EMA(data, i, token)
				if trade.OrderType == "BUY" {
					tradeReport.trade++
					price, idx = simulateBuyorder(data, trade.Quantity, i+1, trade.Spot, trade.Spot-float64(trade.Sl), trade.Spot+float64(trade.Tp))
					i = idx
				} else if trade.OrderType == "SELL" {
					tradeReport.trade++
					price, idx = simulateSellorder(data, trade.Quantity, i+1, trade.Spot, trade.Spot+float64(trade.Sl), trade.Spot-float64(trade.Tp))
					i = idx
				}
				if price > 0 {
					tradeReport.profit += price
					tradeReport.profitCount++
					count = 0
					tradeReport.amount += price

				}
				if price < 0 {
					tradeReport.loss += price
					tradeReport.lossCount++
					count++
					tradeReport.amount += price
					tradeReport.maxContinousloss = math.Max(tradeReport.maxContinousloss, count)
				}

			}
			if tradeReport.profitCount > maxProfit.profitCount {
				maxProfit = tradeReport
				best.sma1 = sma1
				best.sma2 = sma2

			}
			fmt.Printf("SMA1 %v SMA2 %v ---- \n KPI %v\n", sma1, sma2, tradeReport)
		}
	}
	fmt.Printf("SMA1 %v SMA2 %v ---- \n KPI %v\n", best.sma1, best.sma2, maxProfit)

}
func initTrade() kpi {
	return kpi{
		0,
		0,
		0,
		0,
		0,
		0,
		strategy.Amount,
	}
}
