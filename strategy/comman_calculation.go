package strategy

import (
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
)

func CalculatePosition(buyPrice, sl float64, client *smartapigo.Client) int {
	Amount := math.Min(20000.0, GetAmount(client))
	if Amount/buyPrice <= 1 {
		return 0
	}
	return int(Amount/buyPrice) * 4
}

func CaluclateScore(data *DataWithIndicators, order ORDER) float64 {
	score := 0.0
	//score += calculateDirectionalStrength(data.Data, order.OrderType)
	//score += calculateROC(data.Data, order.OrderType)
	score += calculateVolumeSocre(data.Data, order.OrderType)
	//score += calculateLongerTimePeriodDirectionalScore(data.Data, order.OrderType)
	//score += calculateAtrScore(data.Data, order)
	return score
}

func calculateDirectionalStrength(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" || len(data) < 10 {
		return 0.0
	}

	var count = 0.0
	if orderType == "BUY" {
		for i := len(data) - 1; i >= len(data)-10; i-- {
			if data[i].Open > data[i].Close {
				count++
			}
		}

		return count
	}

	// for sell type
	for i := len(data) - 1; i >= len(data)-10; i-- {
		if data[i].Open < data[i].Close {
			count++
		}
	}

	return count
}

func calculateROC(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" || len(data) < 14 {
		return 0.0
	}

	currentIdx := len(data) - 1
	score := 0.0
	ROC := ((data[currentIdx].Close - data[currentIdx-5].Close) / data[currentIdx-5].Close) * 100
	if orderType == "BUY" {
		score = math.Max(0, ROC*2.0)
	}
	if orderType == "SELL" {
		score = math.Max(0, math.Abs(ROC*2.0))
	}
	return score
}

func calculateVolumeSocre(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" || len(data) < 2 {
		return 0.0
	}
	currentIdx := len(data) - 1
	score := 0.0
	if orderType == "BUY" {
		if data[currentIdx].Volume > data[currentIdx-1].Volume && data[currentIdx].Close > data[currentIdx-1].Close {
			score += 7
		}
		if data[currentIdx-1].Volume > data[currentIdx-2].Volume && data[currentIdx-1].Close > data[currentIdx-2].Close {
			score += 3
		}
	} else if orderType == "SELL" {

		if data[currentIdx].Volume > data[currentIdx-1].Volume && data[currentIdx].Close < data[currentIdx-1].Close {
			score += 7
		}
		if data[currentIdx-1].Volume > data[currentIdx-2].Volume && data[currentIdx-1].Close < data[currentIdx-2].Close {
			score += 3
		}
	}

	return score
}

func calculateLongerTimePeriodDirectionalScore(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" {
		return 0
	}
	closePrice := GetClosePriceArray(data)
	if len(closePrice) <= 80 {
		return 0
	}

	currentIdx := len(data) - 1
	score := 0.0
	ema50 := CalculateEma(closePrice, 50)
	ema80 := CalculateEma(closePrice, 80)
	ema30 := CalculateEma(closePrice, 30)
	if orderType == "BUY" {
		if ema30[currentIdx] > ema50[currentIdx] {
			score += 4
		}
		if ema50[currentIdx] > ema80[currentIdx] {
			score += 4
		}
		if closePrice[currentIdx-20] < closePrice[currentIdx] {
			score += 2
		}
	} else if orderType == "SELL" {
		if ema30[currentIdx] < ema50[currentIdx] {
			score += 4
		}
		if ema50[currentIdx] < ema80[currentIdx] {
			score += 4
		}
		if closePrice[currentIdx-20] > closePrice[currentIdx] {
			score += 2
		}
	}
	return score
}

func calculateAtrScore(data []smartapigo.CandleResponse, order ORDER) float64 {
	if order.OrderType == "None" {
		return 0
	}
	currentIdx := len(data) - 1
	atr := CalculateAtr(data, 14, "atr14")
	if len(atr) < 14 {
		return 0
	}
	if atr[currentIdx] >= float64(order.Tp) {
		return 12
	} else {
		return 0
	}
}

func getAvg(data []float64, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	sum := 0.0
	for i := len(data) - 1; i >= len(data)-period; i-- {
		sum += data[i]
	}
	return sum / float64(period)
}

func getAvgVol(data []smartapigo.CandleResponse, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	sum := 0.0
	for i := len(data) - 1; i >= len(data)-period; i-- {
		sum += float64(data[i].Volume)
	}
	return sum / float64(period)
}

func GetVwap(data []smartapigo.CandleResponse, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	cumTypical := 0.0
	cumVol := 0.0
	for i := len(data) - 1; i >= len(data)-period; i-- {
		cumVol += float64(data[i].Volume)
		cumTypical = cumTypical + (((data[i].High + data[i].Low + data[i].Close) / 3) * float64(data[i].Volume))
	}
	return cumTypical / cumVol
}

func CalculateDynamicSL(high, low float64) float64 {
	// Calculate dynamic stop-loss based on volatility or other criteria
	return math.Max(1, math.Abs(high-low)*0.5) // Example: 50% of the channel range
}

func CalculateDynamicTP(high, low float64) int {
	// Calculate dynamic take-profit based on volatility or other criteria
	return int(math.Abs(high-low) * 1.5) // Example: 150% of the channel range
}

func CalculateDynamicQuantity(close float64) int {
	// Calculate dynamic quantity based on risk management rules
	return int(100000 / close) // Example: allocate a fixed amount of capital per trade
}
