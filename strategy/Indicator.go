package strategy

import (
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
)

type StoField struct {
	K float64
	D float64
}

type HaikenOHLC struct {
	Open  float64
	High  float64
	Low   float64
	Close float64
}

type ADX struct {
	Adx     []float64
	PlusDi  []float64
	MinusDi []float64
}

var rsi = make(map[string][]float64)

var sma = make(map[string][]float64)
var ema = make(map[string][]float64)
var sto = make(map[string][]StoField)

var HeikinAshi = make(map[string][]HaikenOHLC)
var adx = make(map[string]ADX)

func roundTwo(number float64) float64 {
	return math.Round(number*100) / 100
}

func CalculateSma(data []float64, period int) []float64 {
	smaArray := make([]float64, 0)
	sum := 0.0
	for i := 0; i < period-1; i++ {
		smaArray = append(smaArray, -1.0)
		sum += data[i]
	}

	sum += data[period-1]
	smaArray = append(smaArray, roundTwo(sum/float64(period)))
	for i := period; i < len(data); i++ {
		sum += data[i] - data[i-period]
		Current := roundTwo(sum / float64(period))
		smaArray = append(smaArray, Current)
	}

	return smaArray
}

func CalculateEma(data []float64, period int) []float64 {
	var emaArray []float64
	multiplier := 2.0 / float64(period+1)
	sum := 0.0
	for i := 0; i < period-1; i++ {
		emaArray = append(emaArray, -1.0)
		sum += data[i]
	}
	sum += data[period-1]
	emaArray = append(emaArray, roundTwo(sum/float64(period)))
	for i := period; i < len(data); i++ {
		Current := roundTwo(((data[i] - emaArray[i-1]) * multiplier) + emaArray[i-1])
		emaArray = append(emaArray, Current)
	}

	return emaArray
}

func CalculateRsi(data []float64, period int) []float64 {
	var rsiArray []float64

	var changeArray []float64
	var gainArray []float64
	var lossArray []float64
	for i := 0; i < len(data); i++ {
		if i == 0 {
			changeArray = append(changeArray, roundTwo(data[i]))
		} else {
			changeArray = append(changeArray, roundTwo(data[i]-data[i-1]))
		}

	}
	for i := 0; i < len(changeArray); i++ {
		if changeArray[i] >= 0 {
			gainArray = append(gainArray, changeArray[i])
			lossArray = append(lossArray, 0)
		} else {
			gainArray = append(gainArray, 0)
			lossArray = append(lossArray, -1*changeArray[i])
		}
	}

	avgGainArray := CalculateEma(gainArray, period)
	avgLossArray := CalculateEma(lossArray, period)

	for i := 0; i < len(data); i++ {
		avgGain := avgGainArray[i]
		avgLoss := avgLossArray[i]
		rs := roundTwo(avgGain / avgLoss)
		rsiVal := 100 - (100 / (1 + rs))
		rsiArray = append(rsiArray, roundTwo(rsiVal))
	}

	return rsiArray
}

/*
The Current Period High minus (-) Current Period Low
The Absolute Value (abs) of the Current Period High minus (-) The Previous Period Close
The Absolute Value (abs) of the Current Period Low minus (-) The Previous Period Close
true range = max[(high - low), abs(high - previous close), abs (low - previous close)]

*/

func CalculateAtr(data []smartapigo.CandleResponse, period int, stockName string) []float64 {
	var trArray []float64
	trArray = append(trArray, data[0].High-data[0].Low)
	for i := 1; i < len(data); i++ {
		trArray = append(trArray, roundTwo(math.Max(roundTwo(data[i].High)-roundTwo(data[i].Low),
			math.Max(math.Abs(roundTwo(data[i].High)-roundTwo(data[i-1].Close)), math.Abs(roundTwo(data[i].Low)-roundTwo(data[i-1].Close))))))
	}

	atrArray := CalculateSma(trArray, period)

	return atrArray
}

// stochastic indicator.

func CalculateSto(data []smartapigo.CandleResponse, period int, stockName string) []StoField {
	k := 1
	d := 3
	var kArray []float64
	high := 0.0
	low := 1000000.0
	for i := 0; i < period; i++ {

		high = math.Max(high, data[i].High)
		low = math.Min(low, data[i].Low)
		kArray = append(kArray, 100.0*(data[i].Close-low)/(high-low))
	}
	for i := period; i < len(data)-2; i++ {
		high = 0.0
		low = 1000000.0
		for j := i; j > i-period; j-- {
			high = roundTwo(math.Max(high, data[j].High))
			low = roundTwo(math.Min(low, data[j].Low))
		}
		kArray = append(kArray, roundTwo(100.0*(data[i].Close-low)/(high-low)))
	}
	for i := len(data) - 2; i < len(data); i++ {
		high = 0.0
		low = 1000000.0
		for j := i; j > i-period; j-- {
			high = roundTwo(math.Max(high, data[j].High))
			low = roundTwo(math.Min(low, data[j].Low))
		}
		kArray = append(kArray, roundTwo(100.0*(data[i].Close-low)/(high-low)))
	}

	var stoArray []StoField
	kArray = CalculateSma(kArray, k)
	dArray := CalculateSma(kArray, d)

	for i := len(stoArray); i < len(data); i++ {
		stoArray = append(stoArray, StoField{kArray[i], dArray[i]})
	}

	return stoArray
}

func CalculateMACD(data []float64, fastPeriod, slowPeriod int) []float64 {
	var macdArray []float64
	slowEma := CalculateEma(data, fastPeriod)
	fastEma := CalculateEma(data, slowPeriod)

	for i := 0; i < fastPeriod; i++ {
		macdArray = append(macdArray, -1.0)
	}
	for i := fastPeriod; i < len(data); i++ {
		macdArray = append(macdArray, fastEma[i]-slowEma[i])
	}

	return macdArray
}

func CalculateSignalLine(data []float64, period, fastPeriod, slowPeriod int) []float64 {
	var signalArray []float64
	macd := CalculateMACD(data, fastPeriod, slowPeriod)
	macdAvg := CalculateEma(macd, 9)

	for i := 0; i < period+fastPeriod; i++ {
		signalArray = append(signalArray, -1.0)
	}
	for i := period + fastPeriod; i < len(data); i++ {
		signalArray = append(signalArray, macdAvg[i])
	}

	return signalArray
}

func CalculateHeikinAshi(ohlc_data []smartapigo.CandleResponse) []HaikenOHLC {

	var heiken_ashi_data []HaikenOHLC
	if len(heiken_ashi_data) == 0 {
		heiken_ashi_data = append(heiken_ashi_data, HaikenOHLC{
			(ohlc_data[0].Open + ohlc_data[0].Close) / 2,
			ohlc_data[0].High,
			ohlc_data[0].Low,
			(ohlc_data[0].Open + ohlc_data[0].High + ohlc_data[0].Low + ohlc_data[0].Close) / 4.0,
		})
	}

	for i := len(heiken_ashi_data); i < len(ohlc_data); i++ {
		heiken_ashi_data = append(heiken_ashi_data, HaikenOHLC{
			Open:  (heiken_ashi_data[i-1].Open + heiken_ashi_data[i-1].Close) / 2,
			Close: (ohlc_data[i].Open + ohlc_data[i].High + ohlc_data[i].Low + ohlc_data[i].Close) / 4.0,
			High:  math.Max(ohlc_data[i].High, math.Max((heiken_ashi_data[i-1].Open+heiken_ashi_data[i-1].Close)/2, (ohlc_data[i].Open+ohlc_data[i].High+ohlc_data[i].Low+ohlc_data[i].Close)/4.0)),
			Low:   math.Min(ohlc_data[i].Low, math.Max((heiken_ashi_data[i-1].Open+heiken_ashi_data[i-1].Close)/2, (ohlc_data[i].Open+ohlc_data[i].High+ohlc_data[i].Low+ohlc_data[i].Close)/4.0)),
		})
	}
	return heiken_ashi_data
}

func calculateDMandTR(current, prev smartapigo.CandleResponse) (float64, float64, float64) {
	plusDM := current.High - prev.High
	minusDM := prev.Low - current.Low
	if plusDM < 0 {
		plusDM = 0
	}
	if minusDM < 0 {
		minusDM = 0
	}
	if plusDM < minusDM {
		plusDM = 0
	} else if minusDM < plusDM {
		minusDM = 0
	}

	tr := math.Max(current.High-current.Low, math.Max(math.Abs(current.High-prev.Close), math.Abs(current.Low-prev.Close)))

	return plusDM, minusDM, tr
}

func smooth(data []float64, period int) []float64 {
	smoothed := make([]float64, len(data))
	smoothed[0] = data[0] // First value is just the first value
	for i := 1; i < len(data); i++ {
		smoothed[i] = smoothed[i-1] - (smoothed[i-1] / float64(period)) + data[i]
	}
	return smoothed
}

func calculateADXDI(data []smartapigo.CandleResponse, period int) ([]float64, []float64, []float64) {
	// Initialize slices with -1
	adxs := make([]float64, len(data))
	plusDIs := make([]float64, len(data))
	minusDIs := make([]float64, len(data))
	for i := range adxs {
		adxs[i] = -1
		plusDIs[i] = -1
		minusDIs[i] = -1
	}

	for startIndex := 0; startIndex <= len(data)-period; startIndex++ {
		endIndex := startIndex + period

		var plusDMs, minusDMs, TRs []float64
		for i := startIndex + 1; i < endIndex; i++ {
			plusDM, minusDM, tr := calculateDMandTR(data[i], data[i-1])
			plusDMs = append(plusDMs, plusDM)
			minusDMs = append(minusDMs, minusDM)
			TRs = append(TRs, tr)
		}

		smoothedPlusDMs := smooth(plusDMs, period)
		smoothedMinusDMs := smooth(minusDMs, period)
		smoothedTRs := smooth(TRs, period)

		var DXs []float64
		for i := 0; i < len(smoothedTRs); i++ {
			plusDI := 100 * smoothedPlusDMs[i] / smoothedTRs[i]
			minusDI := 100 * smoothedMinusDMs[i] / smoothedTRs[i]
			diSum := plusDI + minusDI
			if diSum == 0 {
				diSum = 1
			}

			dx := 100 * math.Abs(plusDI-minusDI) / diSum

			DXs = append(DXs, dx)
			plusDIs[startIndex+period-1] = plusDI
			minusDIs[startIndex+period-1] = minusDI
		}

		firstADX := 0.0
		for _, dx := range DXs {
			firstADX += dx
		}
		firstADX /= float64(len(DXs))

		adxs[startIndex+period-1] = firstADX // Update ADX at the end of the period
	}

	return adxs, plusDIs, minusDIs // Return the slices of ADX, +DI, -DI values
}

func CalculateAdx(data []smartapigo.CandleResponse, period int) ADX {
	Adx, pdi, mdi := calculateADXDI(data, period)
	var adx ADX
	adx.Adx = Adx
	adx.PlusDi = pdi
	adx.MinusDi = mdi
	return adx
}
