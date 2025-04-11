package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const baseURL = "https://ant.aliceblueonline.com/rest/AliceBlueAPIService/api"

// Request and Response Structures for Historical Data
type HistoryRequest struct {
	Token      string `json:"token"`
	Resolution string `json:"resolution"`
	From       string `json:"from"`
	To         string `json:"to"`
	Exchange   string `json:"exchange"`
}

type OHLC struct {
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
	Time   string  `json:"time"`
}

type HistoryResponse struct {
	Stat   string `json:"stat"`
	Result []OHLC `json:"result"`
	EMsg   string `json:"emsg,omitempty"`
}

// FetchHistoricalData calls the /chart/history endpoint
func FetchHistoricalData(sessionToken string, payload HistoryRequest) (*HistoryResponse, error) {
	url := baseURL + "/chart/history"

	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result HistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Stat != "Ok" {
		return &result, fmt.Errorf(result.EMsg)
	}
	return &result, nil
}
