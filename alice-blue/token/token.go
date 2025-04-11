package token

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type TokenEntry struct {
	Token          string `json:"token"`
	TradingSymbol  string `json:"trading_symbol"`
	Exchange       string `json:"exch"`
	InstrumentType string `json:"instrument_type"`
	Expiry         int64  `json:"expiry_date"` // UNIX timestamp in ms
	StrikePrice    string `json:"strike_price"`
	OptionType     string `json:"option_type"`
	Symbol         string `json:"symbol"`
}

var allTokens []TokenEntry

func LoadAllTokens() error {
	url := "https://v2api.aliceblueonline.com/restpy/contract_master?exch=NFO"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading contract master: %v", err)
	}
	defer resp.Body.Close()

	var result map[string][]TokenEntry
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("error parsing json response: %v", err)
	}

	// Assuming the key is "NFO"
	tokens, ok := result["NFO"]
	if !ok {
		return fmt.Errorf("no NFO token data found in response")
	}

	allTokens = tokens
	return nil
}
func GetStockToken(tradingSymbol, exchange, instrumentType string) (string, bool) {
	for _, token := range allTokens {
		if token.TradingSymbol == tradingSymbol && token.Exchange == exchange && token.InstrumentType == instrumentType {
			return token.Token, true
		}
	}
	return "", false
}

func GetOptionToken(symbol, expiry, strikePrice, optionType, exchange string) (string, bool) {
	for _, token := range allTokens {
		if token.Symbol == symbol &&
			token.Exchange == exchange &&
			token.InstrumentType == "OPTIDX" &&
			fmt.Sprintf("%d", token.Expiry) == expiry &&
			token.StrikePrice == strikePrice &&
			token.OptionType == optionType {
			return token.Token, true
		}
	}
	return "", false
}

func GetFuturesToken(symbol, expiry, exchange string) (string, bool) {
	for _, token := range allTokens {
		if token.Symbol == symbol &&
			token.Exchange == exchange &&
			token.InstrumentType == "FUTIDX" &&
			fmt.Sprintf("%d", token.Expiry) == expiry {
			return token.Token, true
		}
	}
	return "", false
}
