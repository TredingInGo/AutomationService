package BackTest

import (
	"github.com/TredingInGo/AutomationService/strategy"
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
)

func CalculateDynamicSL(high, low float64) float64 {
	// Calculate dynamic stop-loss based on volatility or other criteria
	return (math.Abs(high-low) * 0.5) // Example: 50% of the channel range
}

func CalculateDynamicTP(high, low float64) int {
	// Calculate dynamic take-profit based on volatility or other criteria
	return int(math.Abs(high-low) * 1.5) // Example: 150% of the channel range
}

func CalculateDynamicQuantity(close float64) int {
	// Calculate dynamic quantity based on risk management rules
	return int(100000 / close) // Example: allocate a fixed amount of capital per trade
}

func GetORBRange(data strategy.DataWithIndicators, idx *int, dcPeriod int) (float64, float64) {

	high := 0.0
	low := 1000000.0

	for i := *idx - 1; i > *idx-dcPeriod; i-- {
		high = math.Max(data.Data[i].High, high)
		low = math.Min(data.Data[i].Low, low)
	}
	return high, low
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func GetDC(data []smartapigo.CandleResponse, idx int) (float64, float64) {
	high, low := 0.0, 100000000.0
	for i := idx; i > idx-20; i-- {
		high = math.Max(high, data[i].High)
		low = math.Min(low, data[i].Low)
	}
	return high, low

}

func calculatePosition(price float64) int {
	//tempAmount := Amount
	//quantity := tempAmount / price
	//Amount = Amount - (quantity * price) - 200
	return 50
}
