package strategy

import (
	"database/sql"
	"fmt"
	smartapigo "github.com/TredingInGo/smartapi"
	"strconv"
	"time"
)

type OrderDetails struct {
	Spot      float64
	Tp        float64
	Sl        float64
	Quantity  int
	OrderType string
}

type Symbols struct {
	Symbol string `json:"symbol"`
	Token  string `json:"token"`
}

func CloseSession(client *smartapigo.Client) {

	currentTime := time.Now()
	compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 0, 0, 0, currentTime.Location())
	userProfile, _ := client.GetUserProfile()
	if currentTime.After(compareTime) {
		client.Logout()
		fmt.Printf("Session closed  for %v", userProfile.UserName)
		return
	}

}
func TrendFollowingStretgy(client *smartapigo.Client, db *sql.DB) {

	stockList := LoadStockList(db)
	userProfile, _ := client.GetUserProfile()
	TrackOrders(client, "DUMMY", userProfile.UserName)

	for {
		for _, stock := range stockList {
			CloseSession(client)
			Execute(stock.Token, stock.Symbol, client, userProfile.UserName)
		}
		time.Sleep(10 * time.Second)
	}
}

func Execute(symbol, stockToken string, client *smartapigo.Client, userName string) {
	data := GetStockTick(client, stockToken, "FIVE_MINUTE")
	dataToVarify := GetStockTick(client, stockToken, "FIFTEEN_MINUTE")

	if len(data) == 0 {
		return
	}
	PopulateIndicators(data, stockToken, userName)
	order := TrendFollowingRsi(data, dataToVarify, stockToken, symbol, userName, client)
	if order.OrderType == "None" {
		return
	}
	if order.Quantity < 1 {

		return
	}
	orderParams := SetOrderParams(order, stockToken, symbol)
	fmt.Printf("\norder params: for %v \n%v\n", userName, orderParams)
	var orderRes smartapigo.OrderResponse
	orderRes, _ = client.PlaceOrder(orderParams)
	fmt.Printf("order response %v for %v", orderRes, userName)
	TrackOrders(client, symbol, userName)

}

func TrendFollowingRsi(data, dataToVarify []smartapigo.CandleResponse, token, symbol, username string, client *smartapigo.Client) ORDER {
	tokenPlusUser := username + token
	var closePrice = GetClosePriceArray(dataToVarify)
	CalculateEma(closePrice, 7, token+"15M"+"7")
	CalculateEma(closePrice, 22, token+"15M"+"22")
	idx := len(data) - 1
	sma5 := sma[tokenPlusUser+"5"][idx]
	sma8 := sma[tokenPlusUser+"8"][idx]
	sma21 := sma[tokenPlusUser+"21"][idx]
	sma13 := sma[tokenPlusUser+"13"][idx]
	//sma3 := sma[tokenPlusUser+"3"][idx]
	adx14 := adx[tokenPlusUser]
	rsi := rsi[tokenPlusUser]
	//isEmaBuy := isEmaUpAlligator(data, token, symbol)
	//isEmaSell := isEmaDownAlligator(data, token, symbol)
	var order ORDER
	order.OrderType = "None"
	fmt.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	fmt.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if ema[token+"15M"+"7"][idx] > ema[token+"15M"+"22"][idx] && adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma5 > sma8 && sma8 > sma13 && sma13 > sma21 && rsi[idx] < 70 && rsi[idx] > 60 && rsi[idx-2] < rsi[idx] && rsi[idx-1] < rsi[idx] {
		order = ORDER{
			Spot:      data[idx].High + 0.05,
			Sl:        int(data[idx].High * 0.01),
			Tp:        int(data[idx].High * 0.02),
			Quantity:  CalculatePosition(data[idx].High, data[idx].High-data[idx].High*0.01, client),
			OrderType: "BUY",
		}

	} else if ema[token+"15M"+"7"][idx] < ema[token+"15M"+"22"][idx] && adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] < adx14.MinusDi[idx] && sma5 < sma8 && sma8 < sma13 && sma13 < sma21 && rsi[idx] < 40 && rsi[idx] > 30 && rsi[idx-2] > rsi[idx] && rsi[idx-1] > rsi[idx] {
		order = ORDER{
			Spot:      data[idx].Low - 0.05,
			Sl:        int(data[idx].Low * 0.01),
			Tp:        int(data[idx].Low * 0.02),
			Quantity:  CalculatePosition(data[idx].High, data[idx].High-data[idx].High*0.01, client),
			OrderType: "SELL",
		}

	}

	return order
}

func SetOrderParams(order ORDER, token, symbol string) smartapigo.OrderParams {

	orderParams := smartapigo.OrderParams{
		Variety:          "ROBO",
		TradingSymbol:    symbol + "-EQ",
		SymbolToken:      token,
		TransactionType:  order.OrderType,
		Exchange:         "NSE",
		OrderType:        "LIMIT",
		ProductType:      "BO",
		Duration:         "DAY",
		Price:            strconv.FormatFloat(order.Spot, 'f', 2, 64),
		SquareOff:        strconv.Itoa(order.Tp),
		StopLoss:         strconv.Itoa(order.Sl),
		Quantity:         strconv.Itoa(order.Quantity),
		TrailingStopLoss: strconv.Itoa(1),
	}
	return orderParams
}
func GetAmount(client *smartapigo.Client) float64 {
	RMS, _ := client.GetRMS()
	Amount, err := strconv.ParseFloat(RMS.AvailableCash, 64)
	amount := Amount
	if err != nil {
		fmt.Println(err)
	}
	return amount
}

func TrackOrders(client *smartapigo.Client, symbol, userName string) {
	isPrint := true
	for {
		//orders, _ := client.GetOrderBook()
		time.Sleep(1 * time.Second)
		positions, _ := client.GetPositions()
		isAnyPostionOpen := false
		totalPL := 0.0
		fmt.Printf("\n*************** Positions ************** \n")

		for _, postion := range positions {
			if isPrint {
				fmt.Printf("\nposition for %v is %v\n", postion, userName)

			}

			qty, _ := strconv.Atoi(postion.NetQty)
			if postion.SymbolName == symbol && qty != 0 {
				pl, _ := strconv.ParseFloat(postion.NetValue, 64)
				fmt.Printf("current P/L in %v symbol is %v", symbol, pl)
			}
			if qty != 0 {
				isAnyPostionOpen = true
			}
			val, _ := strconv.ParseFloat(postion.NetValue, 64)
			totalPL += val
		}
		isPrint = false
		if isAnyPostionOpen == false {
			if totalPL <= -1000.0 || totalPL >= 2000.0 {
				CloseSession(client)
			}
			fmt.Printf("total P/L  %v", totalPL)
			return
		}

	}

}

func CalculatePosition(buyPrice, sl float64, client *smartapigo.Client) int {
	Amount := GetAmount(client)
	if Amount/buyPrice <= 1 {
		return 0
	}
	return int(Amount/buyPrice) * 4
}

func isEmaUpAlligator(data []smartapigo.CandleResponse, token, symbol string) bool {
	idx := len(data) - 1
	if ema[token+"5"][idx] < ema[token+"21"][idx] {
		return true
	}
	return false
}

func isEmaDownAlligator(data []smartapigo.CandleResponse, token, symbol string) bool {
	idx := len(data) - 1
	if ema[token+"5"][idx] > ema[token+"21"][idx] {
		return true
	}
	return false
}
