package strategy

import (
	"encoding/json"
	"fmt"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/valyala/fasthttp"
	"log"
)

type OHLC struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}
type PredictionResponse struct {
	Predictions [][]float64 `json:"predictions"`
}
type PredictionRequest struct {
	Data [][][]float64 `json:"data"`
}

func OHLCsToFloatSlices(ohlcs []smartapigo.CandleResponse) [][]float64 {
	result := make([][]float64, len(ohlcs))
	for i, ohlc := range ohlcs {
		result[i] = []float64{ohlc.Open, ohlc.High, ohlc.Low, ohlc.Close}
	}
	return result
}

func makePrediction(modelName string, data [][][]float64) ([][]float64, error) {
	// Prepare the request payload
	payload := PredictionRequest{Data: data}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Make the HTTP request to the Python API
	url := fmt.Sprintf("http://127.0.0.1:5050/predict/%s", modelName)
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBody(payloadBytes)

	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		return nil, err
	}

	// Release request and response objects back to pool
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	// Handle the response
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode())
	}

	//log.Println("Raw response:", string(resp.Body()))

	var response [][]float64
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Check if the response contains at least one prediction
	if len(response) == 0 || len(response[0]) == 0 {
		return nil, fmt.Errorf("no predictions found in the response")
	}

	return response, nil
}

func GetDirections(data []smartapigo.CandleResponse, stockName string) []float64 {
	// Convert dataArray to [][]float64 for prediction
	inputForPrediction := OHLCsToFloatSlices(data)

	// Ensure we have at least 10 data points
	if len(inputForPrediction) < 10 {
		log.Fatal("Not enough data for prediction")
	}

	// Use the last 10 data points for prediction

	// Send data as a batch (even if it's a batch of one)
	batchData := [][][]float64{inputForPrediction}

	// Make a prediction using the specified model
	predictions, err := makePrediction(stockName, batchData)
	if err != nil {
		log.Fatal(err)
	}

	//log.Println("Predictions:", predictions)
	var predictionArray []float64
	for i := 0; i < len(predictions); i++ {
		predictionArray = append(predictionArray, predictions[i][0])
	}
	return predictionArray
}
