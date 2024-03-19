package strategy

import "fmt"

func StocBuySignal(k float64, d float64, token string, idx int) bool {
	if idx > 2000 {
		fmt.Printf("over 2000")
	}
	stoValues := sto[token]
	return stoValues[idx-1].K < k && stoValues[idx].K > k
}

func StocSellSignal(k float64, d float64, token string, idx int) bool {
	stoValues := sto[token]
	return stoValues[idx-1].K > k && stoValues[idx].K < k
}

func AlligatorBuy(data []float64, idx int, token string, avgType string) bool {

	if avgType == "EMA" {
		//CalculateEma(data, 5, token+"5")
		//CalculateEma(data, 8, token+"8")
		//CalculateEma(data, 13, token+"13")
		return data[idx] > ema[token+"5"][idx] && ema[token+"5"][idx] > ema[token+"8"][idx] && ema[token+"8"][idx] > ema[token+"13"][idx]
	} else {
		CalculateSma(data, 5, token+"5")
		CalculateSma(data, 8, token+"8")
		CalculateSma(data, 13, token+"13")
		return data[idx] > sma[token+"5"][idx] && sma[token+"5"][idx] > sma[token+"8"][idx] && sma[token+"8"][idx] > sma[token+"13"][idx]
	}
}

func AlligatorSell(data []float64, idx int, token string, avgType string) bool {

	if avgType == "EMA" {
		//CalculateEma(data, 5, token+"5")
		//CalculateEma(data, 8, token+"8")
		//CalculateEma(data, 13, token+"13")
		return data[idx] < ema[token+"5"][idx] && ema[token+"5"][idx] < ema[token+"8"][idx] && ema[token+"8"][idx] < ema[token+"13"][idx]
	} else {
		CalculateSma(data, 5, token+"5")
		CalculateSma(data, 8, token+"8")
		CalculateSma(data, 13, token+"13")
		return data[idx] < sma[token+"5"][idx] && sma[token+"5"][idx] < sma[token+"8"][idx] && sma[token+"8"][idx] < sma[token+"13"][idx]
	}
}

func HeikinAshiReversalSignal(idx int, token string) string {
	if HeikinAshi[token][idx].Close > HeikinAshi[token][idx].Open && HeikinAshi[token][idx-1].Close < HeikinAshi[token][idx-1].Open {
		return "BUY"
	}
	if HeikinAshi[token][idx].Close < HeikinAshi[token][idx].Open && HeikinAshi[token][idx-1].Close > HeikinAshi[token][idx-1].Open {
		return "SELL"
	}
	return "NONE"
}
