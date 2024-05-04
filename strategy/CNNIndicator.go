package strategy

import (
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"math/rand"
)

func trainArima(ohlcData []smartapigo.CandleResponse) *ARIMAModel {
	p := 2 // Autoregressive order
	d := 1 // Differencing order
	q := 2 // Moving average order
	model, err := FitARIMA(ohlcData, p, d, q)
	if err != nil {
		log.Fatal("ARIMA model fitting error:", err)
	}
	return model
}
func getPridectedData(ohlcData []smartapigo.CandleResponse, model *ARIMAModel, numPeriods int) []float64 {
	forecastedHigh, err := ForecastARIMA(model, ohlcData, numPeriods)
	if err != nil {
		log.Fatal("ARIMA forecasting error:", err)
	}
	log.Println("Forecasted High Prices for the Next", numPeriods, "Periods:")
	for i, price := range forecastedHigh {
		log.Printf("Period %d: %.2f\n", i+1, price)
	}
	return forecastedHigh
}
func ARIMA(candles []smartapigo.CandleResponse) []float64 {
	trainData := candles[:len(candles)-10]
	model := trainArima(trainData)
	forecastedHigh := getPridectedData(candles[len(candles)-10:], model, 3)

	return forecastedHigh
}

// FitARIMA fits an ARIMA model to the provided time series data.
func FitARIMA(data []smartapigo.CandleResponse, p, d, q int) (*ARIMAModel, error) {
	differencedData := difference(data, d)
	armaModel, err := EstimateARMA(differencedData, p, q)
	if err != nil {
		return nil, err
	}
	// Combine ARMA and differencing to create ARIMA model
	return &ARIMAModel{ARMA: armaModel, DifferencingOrder: d}, nil
}
func EstimateARMA(data []float64, p, q int) (*ARMAModel, error) {
	// Set the learning rate for gradient descent
	learningRate := 0.01

	// Initialize AR and MA coefficients
	arCoeff := make([]float64, p)
	maCoeff := make([]float64, q)

	// Number of iterations for gradient descent
	numIterations := 1000

	// Perform gradient descent for AR coefficients
	for iter := 0; iter < numIterations; iter++ {
		arGradient := make([]float64, p)

		for i := p; i < len(data); i++ {
			prediction := 0.0
			for j := 0; j < p; j++ {
				prediction += arCoeff[j] * data[i-j-1]
			}
			error := data[i] - prediction
			for j := 0; j < p; j++ {
				arGradient[j] += error * data[i-j-1]
			}
		}
		for j := 0; j < p; j++ {
			// Update AR coefficients using gradient descent
			arCoeff[j] += learningRate * arGradient[j]
		}
	}

	// Perform gradient descent for MA coefficients
	for iter := 0; iter < numIterations; iter++ {
		maGradient := make([]float64, q)

		for i := q; i < len(data); i++ {
			error := data[i]

			// Calculate the MA prediction using current MA coefficients
			prediction := 0.0
			for j := 0; j < q; j++ {
				prediction += maCoeff[j] * error
			}

			// Update MA coefficient gradients
			for j := 0; j < q; j++ {
				maGradient[j] += error * prediction
			}
		}
		for j := 0; j < q; j++ {
			// Update MA coefficients using gradient descent
			maCoeff[j] += learningRate * maGradient[j]
		}
	}

	return &ARMAModel{AR: arCoeff, MA: maCoeff}, nil
}

// ForecastARIMA forecasts future values using the ARIMA model.
func ForecastARIMA(model *ARIMAModel, data []smartapigo.CandleResponse, numPeriods int) ([]float64, error) {
	// Perform differencing for order d
	differencedData := difference(data, model.DifferencingOrder)

	// Forecast using ARMA model
	armaForecast, err := ForecastARMA(model.ARMA, differencedData, numPeriods)
	if err != nil {
		return nil, err
	}

	// Undifference the forecasts
	forecasts := undifference(data, armaForecast, model.DifferencingOrder)

	return forecasts, nil
}

// ForecastARMA forecasts future values using the ARMA model.
// ForecastARMA uses Box-Jenkins forecasting to make forecasts using ARMA model parameters.

func ForecastARMA(model *ARMAModel, data []float64, numPeriods int) ([]float64, error) {
	// In this simplified example, we use a random walk for forecasting
	// You should replace this with a more advanced forecasting method
	forecasts := make([]float64, numPeriods)
	for i := 0; i < numPeriods; i++ {
		// Calculate the forecast based on the AR and MA terms
		forecast := 0.0
		for j := 0; j < len(model.AR) && i-j >= 0; j++ {
			forecast += model.AR[j] * data[i-j]
		}
		for j := 0; j < len(model.MA) && i-j >= 0; j++ {
			forecast += model.MA[j] * forecasts[i-j]
		}
		// In a real implementation, you'd use a more advanced forecasting method
		// Here, we use a random noise term for simplicity
		forecast += rand.Float64()
		forecasts[i] = forecast
	}
	return forecasts, nil
}

// Perform differencing on the time series data
// difference applies differencing to the "Close" prices in the OHLC data.
func difference(data []smartapigo.CandleResponse, order int) []float64 {
	result := make([]float64, len(data)-order)
	for i := order; i < len(data); i++ {
		result[i-order] = data[i].Close - data[i-order].Close
	}
	return result
}

// undifference performs the inverse of differencing to obtain forecasts for "Close" prices.
func undifference(originalData []smartapigo.CandleResponse, differencedData []float64, order int) []float64 {
	result := make([]float64, len(differencedData)+order)

	// Initialize the result with the original "Close" prices
	for i := 0; i < order; i++ {
		result[i] = originalData[i].Close
	}

	// Perform the undifferencing operation
	for i := order; i < len(result); i++ {
		result[i] = differencedData[i-order] + result[i-1]
	}

	return result
}

type ARIMAModel struct {
	ARMA              *ARMAModel
	DifferencingOrder int
}

// ARMAModel represents the ARMA model.
type ARMAModel struct {
	AR []float64 // Autoregressive coefficients
	MA []float64 // Moving average coefficients
}
