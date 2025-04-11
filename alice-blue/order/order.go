package order

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const baseURL = "https://ant.aliceblueonline.com/rest/AliceBlueAPIService/api"

// Request Structures
type PlaceOrderRequest struct {
	DiscQty          string `json:"discqty"`
	TradingSymbol    string `json:"trading_symbol"`
	Exch             string `json:"exch"`
	TransType        string `json:"transtype"`
	Ret              string `json:"ret"`
	Prctyp           string `json:"prctyp"`
	Qty              string `json:"qty"`
	SymbolID         string `json:"symbol_id"`
	Price            string `json:"price"`
	TrigPrice        string `json:"trigPrice"`
	PCode            string `json:"pCode"`
	Complexty        string `json:"complexty"`
	OrderTag         string `json:"orderTag"`
	DeviceNumber     string `json:"deviceNumber"`
	Target           string `json:"target,omitempty"`
	StopLoss         string `json:"stopLoss,omitempty"`
	TrailingStopLoss string `json:"trailing_stop_loss,omitempty"`
}

type CancelOrderRequest struct {
	Exch          string `json:"exch"`
	NestOrderNo   string `json:"nestOrderNumber"`
	TradingSymbol string `json:"trading_symbol"`
	DeviceNumber  string `json:"deviceNumber"`
}

type SquareOffRequest struct {
	ExchSeg      string `json:"exchSeg"`
	PCode        string `json:"pCode"`
	NetQty       string `json:"netQty"`
	TokenNo      string `json:"tockenNo"`
	Symbol       string `json:"symbol"`
	DeviceNumber string `json:"deviceNumber"`
}

type ExitBracketRequest struct {
	NestOrderNumber string `json:"nestOrderNumber"`
	SymbolOrderID   string `json:"symbolOrderId"`
	Status          string `json:"status"`
}

type ModifyOrderRequest struct {
	TransType       string `json:"transtype"`
	DiscQty         string `json:"discqty"`
	Exch            string `json:"exch"`
	TradingSymbol   string `json:"trading_symbol"`
	NestOrderNumber string `json:"nestOrderNumber"`
	Prctyp          string `json:"prctyp"`
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	TrigPrice       string `json:"trigPrice"`
	FilledQty       string `json:"filledQuantity"`
	PCode           string `json:"pCode"`
	DeviceNumber    string `json:"deviceNumber"`
}

// Common Response Structure
type OrderResponse struct {
	Stat            string `json:"stat"`
	NestOrderNumber string `json:"nestOrderNumber,omitempty"`
	EMsg            string `json:"emsg,omitempty"`
}

type BasketMarginRequest struct {
	Exchange      string `json:"exchange"`
	TradingSymbol string `json:"tradingSymbol"`
	Price         string `json:"price"`
	Qty           string `json:"qty"`
	Product       string `json:"product"`
	PriceType     string `json:"priceType"`
	Token         string `json:"token"`
	TransType     string `json:"transType"`
}

type BasketMarginResponse struct {
	Stat            string `json:"stat"`
	MarginUsed      string `json:"marginUsed"`
	MarginUsedTrade string `json:"marginUsedTrade"`
	ErrorMessage    string `json:"emsg,omitempty"`
}

// API Methods
func PlaceOrder(sessionToken string, payload PlaceOrderRequest) (*OrderResponse, error) {
	url := baseURL + "/placeOrder/executePlaceOrder"
	return doPost(sessionToken, payload, url)
}

func CancelOrder(sessionToken string, payload CancelOrderRequest) (*OrderResponse, error) {
	url := baseURL + "/placeOrder/cancelOrder"
	return doPost(sessionToken, payload, url)
}

func SquareOff(sessionToken string, payload SquareOffRequest) (*OrderResponse, error) {
	url := baseURL + "/positionAndHoldings/sqrOofPosition"
	return doPost(sessionToken, payload, url)
}

func ExitBracketOrder(sessionToken string, payload ExitBracketRequest) (*OrderResponse, error) {
	url := baseURL + "/placeOrder/exitBracketOrder"
	return doPost(sessionToken, payload, url)
}

func ModifyOrder(sessionToken string, payload ModifyOrderRequest) (*OrderResponse, error) {
	url := baseURL + "/placeOrder/modifyOrder"
	return doPost(sessionToken, payload, url)
}

func FetchOrderBook(sessionToken string) ([]map[string]interface{}, error) {
	url := baseURL + "/placeOrder/fetchOrderBook"
	return doGetList(sessionToken, url)
}

func FetchTradeBook(sessionToken string) ([]map[string]interface{}, error) {
	url := baseURL + "/placeOrder/fetchTradeBook"
	return doGetList(sessionToken, url)
}

// Shared Utility Functions
func doPost(sessionToken string, payload interface{}, url string) (*OrderResponse, error) {
	reqBody, _ := json.Marshal([]interface{}{payload})
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

	var result OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Stat != "Ok" {
		return &result, fmt.Errorf(result.EMsg)
	}
	return &result, nil
}

func doGetList(sessionToken, url string) ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result["stat"] != "Ok" {
		return nil, fmt.Errorf("API error: %v", result["emsg"])
	}

	data := result["orderBook"]
	if data == nil {
		data = result["tradeBook"]
	}

	return data.([]map[string]interface{}), nil
}

func CheckBasketMargin(sessionToken string, basket []BasketMarginRequest) (*BasketMarginResponse, error) {
	url := baseURL + "/basket/getMargin"
	reqBody, _ := json.Marshal(basket)

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

	var result BasketMarginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Stat != "Ok" {
		return nil, fmt.Errorf("Margin check failed: %s", result.ErrorMessage)
	}
	return &result, nil
}
