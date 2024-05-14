package strategy

import (
	"database/sql"
	"github.com/TredingInGo/AutomationService/historyData"
	"github.com/TredingInGo/AutomationService/smartStream"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/TredingInGo/smartapi/models"
	"log"
	"math"
	"strconv"
	"strings"
)

type Strategy struct {
	history  historyData.History
	pastData []smartapigo.CandleResponse

	LiveData    chan *models.SnapQuote
	chForCandle chan *models.SnapQuote
	db          *sql.DB
}

type legInfo struct {
	price     float64
	strike    int
	orderType string
	token     string
	symbol    string
	quantity  int
	oi        uint64
}

type Legs struct {
	leg1 legInfo
	leg2 legInfo
	leg3 legInfo
	leg4 legInfo
}

type PriceInfo struct {
	price  float64
	token  string
	symbol string
	oi     uint64
}

func NEW() strategy {
	return strategy{

		LiveData:    make(chan *models.SnapQuote, 100),
		chForCandle: make(chan *models.SnapQuote, 100),
	}
}

func (s *strategy) Arbitrage(ltp smartStream.SmartStream, client *smartapigo.Client) {

	expiry := niftyExpairy
	index := "NIFTY"
	index = strings.ToUpper(index)
	expiry = strings.ToUpper(expiry)

	ATMstrike := getATMStrikePrice(client, index)
	if ATMstrike == 0 {
		return
	}

	spotDiff := 100
	limit := 600
	lotsize := 15
	if index == "NIFTY" {
		spotDiff = 50
		limit = 300
		lotsize = 25
	}

	ITM := int(ATMstrike) - limit
	OTM := int(ATMstrike) + limit

	var tokenMap = make(map[string]string)
	var priceMap = make(map[string]priceInfo)

	tokenInfo := createTokenMapAndBuildTokenModel(tokenMap, expiry, index, int(ATMstrike))
	totalToken := len(tokenInfo)
	go ltp.Connect(s.LiveData, models.SNAPQUOTE, tokenInfo)
	tokenCount := 0

	for data := range s.LiveData {
		tokenCount++
		priceMap[tokenMap[data.TokenInfo.Token]] = priceInfo{
			price:  float64(data.LastTradedPrice) / 100.0,
			token:  data.TokenInfo.Token,
			symbol: index + expiry + tokenMap[data.TokenInfo.Token],
			oi:     data.OpenInterest,
		}
		var leg Legs
		maxPL := -100000000.0
		if tokenCount%totalToken == 0 {
			for spot1 := ITM; spot1 <= OTM; spot1 += spotDiff {
				for spot2 := ITM; spot2 <= OTM; spot2 += spotDiff {
					if spot1 != spot2 {
						PLForCurrentPrice := CalculateNetPL(ATMstrike, float64(spot1), float64(spot2), priceMap) * float64(lotsize)
						if maxPL < PLForCurrentPrice {
							leg.leg1 = legInfo{
								price:     priceMap[strconv.Itoa(spot1)+call].price,
								strike:    spot1,
								orderType: "BUY",
								token:     priceMap[strconv.Itoa(spot1)+call].token,
								symbol:    priceMap[strconv.Itoa(spot1)+call].symbol,
								quantity:  lotsize,
								oi:        priceMap[strconv.Itoa(spot1)+call].oi,
							}
							leg.leg2 = legInfo{
								price:     priceMap[strconv.Itoa(spot1)+put].price,
								strike:    spot1,
								orderType: "SELL",
								token:     priceMap[strconv.Itoa(spot1)+put].token,
								symbol:    priceMap[strconv.Itoa(spot1)+put].symbol,
								quantity:  lotsize,
								oi:        priceMap[strconv.Itoa(spot1)+put].oi,
							}
							leg.leg3 = legInfo{
								price:     priceMap[strconv.Itoa(spot2)+call].price,
								strike:    spot2,
								orderType: "SELL",
								token:     priceMap[strconv.Itoa(spot2)+call].token,
								symbol:    priceMap[strconv.Itoa(spot2)+call].symbol,
								quantity:  lotsize,
								oi:        priceMap[strconv.Itoa(spot2)+call].oi,
							}
							leg.leg4 = legInfo{
								price:     priceMap[strconv.Itoa(spot2)+put].price,
								strike:    spot2,
								orderType: "BUY",
								token:     priceMap[strconv.Itoa(spot2)+put].token,
								symbol:    priceMap[strconv.Itoa(spot2)+put].symbol,
								quantity:  lotsize,
								oi:        priceMap[strconv.Itoa(spot2)+put].oi,
							}
							maxPL = PLForCurrentPrice
						}
					}
				}
			}
			placeFOArbitrageOrder(maxPL, leg, client, index)
		}

	}

}

func getATMStrikePrice(client *smartapigo.Client, index string) float64 {
	var candles []smartapigo.CandleResponse
	if index == "NIFTY" {
		candles = GetStockTick(client, niftyToken, "ONE_DAY", nse)
		spot := candles[len(candles)-1].Close
		mf := int(spot) / int(nifty)
		return float64(nifty * mf)
	} else if index == "BANKNIFTY" {
		candles = GetStockTick(client, bankNiftyToken, "ONE_DAY", nse)
		spot := candles[len(candles)-1].Close
		mf := int(spot) / int(bankNifty)
		return float64(bankNifty * mf)
	}
	return 0

}

func createTokenMapAndBuildTokenModel(tokenMap map[string]string, expiry, index string, ATM int) []models.TokenInfo {
	spotDiff := 100
	limit := 600
	if index == "NIFTY" {
		spotDiff = 50
		limit = 350
	}

	ITM := ATM - limit
	OTM := ATM + limit
	var tokenInfo []models.TokenInfo
	for spot := ITM; spot <= OTM; spot += spotDiff {
		callSymbol := index + expiry + strconv.Itoa(spot) + call
		callToken := GetFOToken(callSymbol, nfo)
		putSymbol := index + expiry + strconv.Itoa(spot) + put
		putToken := GetFOToken(putSymbol, nfo)
		tokenMap[callToken] = strconv.Itoa(spot) + call
		tokenMap[putToken] = strconv.Itoa(spot) + put
		tokenInfo = append(tokenInfo, models.TokenInfo{models.NSEFO, callToken})
		tokenInfo = append(tokenInfo, models.TokenInfo{models.NSEFO, putToken})
	}
	return tokenInfo

}

func CalculateNetPL(ATMStrike, strike1, strike2 float64, priceMap map[string]priceInfo) float64 {
	stirke1CallIV := math.Max(0, ATMStrike-strike1)
	strike2CallIV := math.Max(0, ATMStrike-strike2)
	strike1PutIV := math.Max(0, strike1-ATMStrike)
	strike2PutIV := math.Max(0, strike2-ATMStrike)
	strike1CallPL := stirke1CallIV - priceMap[strconv.Itoa(int(strike1))+call].price
	strike2CallPL := priceMap[strconv.Itoa(int(strike2))+call].price - strike2CallIV
	strike1PutPL := priceMap[strconv.Itoa(int(strike1))+put].price - strike1PutIV
	strike2PutPL := strike2PutIV - priceMap[strconv.Itoa(int(strike2))+put].price
	return strike1CallPL + strike2CallPL + strike1PutPL + strike2PutPL

}

func placeFOArbitrageOrder(maxPL float64, leg Legs, client *smartapigo.Client, index string) {
	log.Println("MaxProfit: ", maxPL)
	log.Println(leg)
	//qty := leg.leg1.quantity / 15.0
	//if index == "NIFTY" {
	//	qty = leg.leg1.quantity / 25.0
	//}
	if maxPL > 350 {
		if isAllLiquid(leg) {
			order1 := getFOOrderParamsForArbitrage(leg.leg1)
			order2 := getFOOrderParamsForArbitrage(leg.leg2)
			order3 := getFOOrderParamsForArbitrage(leg.leg3)
			order4 := getFOOrderParamsForArbitrage(leg.leg4)
			orderRes1, err1 := client.PlaceOrder(order1)
			orderRes2, err2 := client.PlaceOrder(order2)
			orderRes3, err3 := client.PlaceOrder(order3)
			orderRes4, err4 := client.PlaceOrder(order4)
			log.Println(err1, err2, err3, err4)
			log.Println("orderID 1: ", orderRes1)
			log.Println("orderID 2: ", orderRes2)
			log.Println("orderID 3: ", orderRes3)
			log.Println("orderID 4: ", orderRes4)
		}

	}

}

func getFOOrderParamsForArbitrage(order legInfo) smartapigo.OrderParams {
	orderParams := smartapigo.OrderParams{
		Variety:          "NORMAL",
		TradingSymbol:    order.symbol,
		SymbolToken:      order.token,
		TransactionType:  order.orderType,
		Exchange:         nfo,
		OrderType:        "LIMIT",
		ProductType:      "CARRYFORWARD",
		Duration:         "DAY",
		Price:            strconv.FormatFloat(order.price, 'f', 2, 64),
		Quantity:         strconv.Itoa(order.quantity),
		SquareOff:        strconv.FormatFloat(0, 'f', 2, 64),
		TrailingStopLoss: strconv.FormatFloat(0, 'f', 2, 64),
		StopLoss:         strconv.FormatFloat(0, 'f', 2, 64),
	}

	return orderParams
}

func isAllLiquid(leg Legs) bool {
	if leg.leg1.oi < 1000 {
		return false
	}
	if leg.leg2.oi < 1000 {
		return false
	}
	if leg.leg3.oi < 1000 {
		return false
	}
	if leg.leg4.oi < 1000 {
		return false
	}
	return true
}
