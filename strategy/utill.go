package strategy

import (
	"bytes"
	"encoding/json"
	"fmt"
	smartapigo "github.com/TredingInGo/smartapi"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type params struct {
	key    string
	volume float64
}

type paramsSlice []params

func (s paramsSlice) Len() int {
	return len(s)
}

func (s paramsSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s paramsSlice) Less(i, j int) bool {
	return s[i].key < s[j].key
}

var tokenMap = make(map[string][]smartapigo.CandleResponse)

func GetClosePriceArray(data []smartapigo.CandleResponse) []float64 {
	var closePrice []float64
	for i := 0; i < len(data); i++ {
		closePrice = append(closePrice, data[i].Close)
	}
	return closePrice
}

func GetHighPriceArray(data []smartapigo.CandleResponse) []float64 {
	var highPrice []float64
	for i := 0; i < len(data); i++ {
		highPrice = append(highPrice, data[i].High)
	}
	return highPrice
}

func GetLowPriceArray(data []smartapigo.CandleResponse) []float64 {
	var lowPrice []float64
	for i := 0; i < len(data); i++ {
		lowPrice = append(lowPrice, data[i].Low)
	}
	return lowPrice
}

func GetVolumeArray(data []smartapigo.CandleResponse) []float64 {
	var volume []float64
	for i := 0; i < len(data); i++ {
		volume = append(volume, float64(data[i].Volume))
	}
	return volume
}

func (s strategy) FilterStocks(exchange string) []string {
	tokenList := GetAllToken(exchange)
	var filteredList []params
	for i := range tokenList {
		s.getPrevData(tokenList[i])
	}
	for key := range tokenMap {
		candles := tokenMap[key]
		lastCandle := candles[len(candles)-1]
		closingPrice := GetClosePriceArray(candles)
		volumes := GetVolumeArray(candles)
		last50 := closingPrice[len(closingPrice)-50:]
		last20Volume := volumes[len(volumes)-20:]
		CalculateSma(last20Volume, 9)
		emaArray := CalculateEma(last50, 44)
		smaArray := CalculateSma(last50, 9)
		if lastCandle.Low > emaArray[len(emaArray)-1] && lastCandle.High >= 200.00 && lastCandle.High <= 1000.00 {
			filteredList = append(filteredList, params{
				key:    key,
				volume: smaArray[len(smaArray)-1],
			})
		}
	}
	sort.Sort(paramsSlice(filteredList))
	var keys []string
	for i := len(filteredList) - 3; i < len(filteredList); i++ {
		keys = append(keys, filteredList[i].key)
	}
	return keys
}

func (s strategy) getPrevData(token string) {
	pastData := tokenMap[token]
	var max = 3

	for count := 0; count < max; count++ {
		t := time.Now()
		t = t.Add(time.Hour * 24 * time.Duration(-count))

		if t.Weekday() == 0 || t.Weekday() == 6 {
			max++
			continue
		}

		year := t.Year()
		month := int(t.Month())
		day := t.Day()
		fromDate := fmt.Sprintf("%d-%02d-%02d %v", year, month, day, startTime)
		toDate := fmt.Sprintf("%d-%02d-%02d %v", year, month, day, endTime)

		candles, err := s.history.GetCandle(smartapigo.CandleParams{
			Exchange:    "NSE",
			SymbolToken: token,
			Interval:    "FIVE_MINUTE",
			FromDate:    fromDate,
			ToDate:      toDate,
		})
		if err != nil {
			log.Println("error while getting history data", err)
			continue
		}

		pastData = append(candles, pastData...)
		tokenMap[token] = pastData
	}
}
func (s *strategy) fillPastData(symbol string, exhange string, max int) {
	// add last 10 days data

	for count := 0; count < max; count++ {
		t := time.Now()
		t = t.Add(time.Hour * 24 * time.Duration(-count))

		if t.Weekday() == 0 || t.Weekday() == 6 {
			max++
			continue
		}

		year := t.Year()
		month := int(t.Month())
		day := t.Day()
		fromDate := fmt.Sprintf("%d-%02d-%02d %v", year, month, day, startTime)
		toDate := fmt.Sprintf("%d-%02d-%02d %v", year, month, day, endTime)

		candles, err := s.history.GetCandle(smartapigo.CandleParams{
			Exchange:    exhange,
			SymbolToken: symbol,
			Interval:    "FIVE_MINUTE",
			FromDate:    fromDate,
			ToDate:      toDate,
		})
		if err != nil {
			log.Println("error while getting history data", err)
			continue
		}

		s.pastData = append(candles, s.pastData...)
	}
}

//func (s *strategy) makeCandle(ch <-chan *models.SnapQuote, duration int) {
//	for data := range ch {
//		lastCandleFormAt := s.pastData[len(s.pastData)-1].Timestamp
//		nextCandleFormAt := lastCandleFormAt.Add(time.Second * time.Duration(duration))
//		currentTime := time.Now()
//		if currentTime.After(nextCandleFormAt) {
//
//			s.pastData[len(s.pastData)-1].Timestamp = nextCandleFormAt
//			tempOhlc = CandleResponse{
//				Timestamp: nextCandleFormAt,
//				Open:      float64(data.LastTradedPrice) / 100,
//				High:      float64(data.LastTradedPrice) / 100,
//				Low:       float64(data.LastTradedPrice) / 100,
//				Close:     float64(data.LastTradedPrice) / 100,
//				Volume:    0,
//			}
//			s.pastData = append(s.pastData, smartapigo.CandleResponse{
//				Timestamp: tempOhlc.Timestamp,
//				Open:      tempOhlc.Open,
//				High:      tempOhlc.High,
//				Low:       tempOhlc.Low,
//				Close:     tempOhlc.Close,
//				Volume:    0,
//			})
//		} else {
//			if tempOhlc.Open == 0.0 {
//
//				tempOhlc = CandleResponse{
//					Timestamp: lastCandleFormAt,
//					Open:      float64(data.LastTradedPrice) / 100,
//					High:      float64(data.LastTradedPrice) / 100,
//					Low:       float64(data.LastTradedPrice) / 100,
//					Close:     float64(data.LastTradedPrice) / 100,
//					Volume:    0,
//				}
//				s.pastData = append(s.pastData, smartapigo.CandleResponse{
//					Timestamp: tempOhlc.Timestamp,
//					Open:      tempOhlc.Open,
//					High:      tempOhlc.High,
//					Low:       tempOhlc.Low,
//					Close:     tempOhlc.Close,
//					Volume:    0,
//				})
//			} else {
//				tempOhlc.Close = float64(data.LastTradedPrice) / 100
//				s.pastData[len(s.pastData)-1].Close = tempOhlc.Close
//				s.pastData[len(s.pastData)-1].High = math.Max(s.pastData[len(s.pastData)-1].High, tempOhlc.Close)
//				s.pastData[len(s.pastData)-1].Low = math.Min(s.pastData[len(s.pastData)-1].Low, tempOhlc.Close)
//			}
//
//		}
//	}
//
//}

func trainModel(pastData []smartapigo.CandleResponse, token string) {
	ohlcData := make([][]float64, 0)
	for _, dataPoint := range pastData {
		data := []float64{
			dataPoint.Open,
			dataPoint.High,
			dataPoint.Low,
			dataPoint.Close,
		}

		ohlcData = append(ohlcData, data)
	}

	// Define the payload
	payload := map[string]interface{}{
		"stock_name": token,
		"ohlc_data":  ohlcData,
	}

	// Convert the payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return
	}

	// Define the API endpoint URL
	trainingUrl := "http://127.0.0.1:6000/train_model" // Replace with your API URL

	// Send a POST request to the API
	resp, err := http.Post(trainingUrl, "application/json", bytes.NewBuffer(payloadJSON))
	if err != nil {
		log.Println("Error sending POST request:", err)
		return
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode == http.StatusOK {
		log.Println("API call successful")
	} else {
		log.Println("API call failed with status:", resp.Status)
	}
}

func GetNextPrice(stockName string, pastData []smartapigo.CandleResponse) (float64, error) {
	// Define the URL of the Python API
	var ohlcData []map[string]interface{}
	for _, candle := range pastData {
		data := map[string]interface{}{
			"Open":  candle.Open,
			"High":  candle.High,
			"Low":   candle.Low,
			"Close": candle.Close,
		}
		ohlcData = append(ohlcData, data)
	}
	apiURL := "http://localhost:5001/get_next_price"

	// Create a request payload
	requestData := map[string]interface{}{
		"stock_name": stockName,
		"ohlc_data":  ohlcData,
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return 0, err
	}

	// Send a POST request to the Python API
	response, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	// Read and parse the response from the Python API
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	// Parse the JSON response
	var responseData map[string]interface{}
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		return 0, err
	}

	// Check if the response contains an error
	if errorMessage, ok := responseData["error"]; ok {
		return 0, fmt.Errorf("Python API error: %s", errorMessage)
	}

	// Extract the predicted price from the response
	predictedPrice, ok := responseData["next_predicted_price"].(float64)
	if !ok {
		return 0, fmt.Errorf("Failed to get the next predicted price")
	}

	return predictedPrice, nil
}

func PopulateIndicators(data *DataWithIndicators) {
	var closePrice = GetClosePriceArray(data.Data)
	if data.Indicators == nil {
		data.Indicators = make(map[string][]float64)
	}
	if data.StoArray == nil {
		data.StoArray = make(map[string][]StoField)
	}
	if data.Adx == nil {
		data.Adx = make(map[string]ADX)
	}
	for i := 2; i <= 50; i++ {
		ema := CalculateEma(closePrice, i)
		sma := CalculateSma(closePrice, i)
		rsi := CalculateRsi(closePrice, i)
		atr := CalculateAtr(data.Data, i, data.Token)
		sto := CalculateSto(data.Data, i, data.Token)
		macd := CalculateSignalLine(closePrice, i, 9, 26)
		adx := CalculateAdx(data.Data, i)
		data.Indicators["ema"+strconv.Itoa(i)] = ema
		data.Indicators["sma"+strconv.Itoa(i)] = sma
		data.Indicators["rsi"+strconv.Itoa(i)] = rsi
		data.Indicators["atr"+strconv.Itoa(i)] = atr
		data.Indicators["macd"+strconv.Itoa(i)] = macd
		data.StoArray["Sto"+strconv.Itoa(i)] = sto
		data.Adx["Adx"+strconv.Itoa(i)] = adx

	}

}

//func CalculatePositionSize(buyPrice, sl float64) int {
//	if Amount/buyPrice <= 1 {
//		return 0
//	}
//	maxRiskPercent := 0.05 // 2% maximum risk allowed
//	maxRiskAmount := Amount * maxRiskPercent
//	riskPerShare := math.Max(1, math.Abs(buyPrice-sl))
//	positionSize := maxRiskAmount / riskPerShare
//	return int(math.Min(Amount/buyPrice, positionSize))
//}
//func SetAmount(amount float64) {
//	Amount = amount
//}
