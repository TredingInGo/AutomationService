package strategy

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Instrument struct {
	Token          string `json:"token"`
	Symbol         string `json:"symbol"`
	Name           string `json:"name"`
	Expiry         string `json:"expiry"`
	Strike         string `json:"strike"`
	LotSize        string `json:"lotsize"`
	InstrumentType string `json:"instrumenttype"`
	ExchSeg        string `json:"exch_seg"`
	TickSize       string `json:"tick_size"`
}

var apiUrl = "https://margincalculator.angelbroking.com/OpenAPI_File/files/OpenAPIScripMaster.json"
var client = &http.Client{}

type InstrumentList struct {
	Instruments []Instrument `json:"instruments"`
}

var InstrumentLists []Instrument

func tokenLookup(ticker string, exchange string) string {
	for _, instrument := range InstrumentLists {
		if instrument.Name == ticker && instrument.ExchSeg == exchange && getLastSymbolPart(instrument.Symbol) == "EQ" {
			return instrument.Token
		}
	}
	return "" // Return -1 if no matching token is found
}
func GetFOToken(ticker string, exchange string) string {
	for _, instrument := range InstrumentLists {
		if instrument.Symbol == ticker && instrument.ExchSeg == exchange {
			return instrument.Token
		}
	}
	return ""
}

func GetStockName(token string) string {
	for _, instrument := range InstrumentLists {
		if instrument.Token == token {
			return instrument.Name
		}
	}
	return "Not Found"
}

func GetAllToken(exchange string) []string {
	var tokenList []string
	for _, instrument := range InstrumentLists {
		if instrument.ExchSeg == exchange && getLastSymbolPart(instrument.Symbol) == "EQ" {
			tokenList = append(tokenList, instrument.Token)
		}
	}
	return tokenList
}

func getLastSymbolPart(symbol string) string {
	parts := strings.Split(symbol, "-")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func GetToken(ticker string, exchange string) string {
	token := tokenLookup(ticker, exchange)
	return token
}

func PopuletInstrumentsList() {

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		log.Println("Error creating the request:", err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending the request:", err)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading the response:", err)
		return
	}

	if err := json.Unmarshal([]byte(body), &InstrumentLists); err != nil {
		log.Println("Error unmarshaling JSON:", err)
		return
	}
}
