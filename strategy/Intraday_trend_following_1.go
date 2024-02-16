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

	if currentTime.After(compareTime) {
		client.Logout()
		fmt.Printf("Session closed ")
		return
	}

}
func TrendFollowingStretgy(client *smartapigo.Client, db *sql.DB) {

	stockList := LoadStockList(db)
	TrackOrders(client, "DUMMY")
	for {
		for _, stock := range stockList {
			CloseSession(client)
			Execute(stock.Token, stock.Symbol, client)
		}
		time.Sleep(10 * time.Second)
	}
}

func Execute(symbol, stockToken string, client *smartapigo.Client) {
	data := GetStockTick(client, stockToken, "FIVE_MINUTE")
	if len(data) == 0 {
		return
	}
	PopulateIndicators(data, stockToken)
	order := TrendFollowingRsi(data, stockToken, symbol, client)
	if order.OrderType == "None" {
		return
	}
	if order.Quantity < 1 {

		return
	}
	orderParams := SetOrderParams(order, stockToken, symbol)
	fmt.Printf("\norder params:\n%v\n", orderParams)
	var orderRes smartapigo.OrderResponse
	orderRes, _ = client.PlaceOrder(orderParams)
	fmt.Printf("order response %v", orderRes)
	TrackOrders(client, symbol)

}

func TrendFollowingRsi(data []smartapigo.CandleResponse, token, symbol string, client *smartapigo.Client) ORDER {
	idx := len(data) - 1
	sma5 := sma[token+"5"][idx]
	sma8 := sma[token+"8"][idx]
	adx14 := adx[token]
	rsi := rsi[token]
	var order ORDER
	order.OrderType = "None"
	fmt.Printf("\nStock Name: %v\n", symbol)
	fmt.Printf("adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v ", adx14.Adx[idx], sma5, sma8, sma[token+"13"][idx], sma[token+"21"][idx], rsi[idx])
	if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma5 > sma8 && sma8 > sma[token+"13"][idx] && sma[token+"13"][idx] > sma[token+"21"][idx] && rsi[idx] < 70 && rsi[idx] > 55 && rsi[idx-2] < rsi[idx] {
		order = ORDER{
			Spot:      data[idx].High + 0.05,
			Sl:        int(data[idx].High * 0.01),
			Tp:        int(data[idx].High * 0.02),
			Quantity:  CalculatePosition(data[idx].High, data[idx].High-data[idx].High*0.01, client),
			OrderType: "BUY",
		}

	} else if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] < adx14.MinusDi[idx] && sma5 < sma8 && sma8 < sma[token+"13"][idx] && sma[token+"13"][idx] < sma[token+"21"][idx] && rsi[idx] < 40 && rsi[idx] > 30 && rsi[idx-2] > rsi[idx] {
		order = ORDER{
			Spot:      data[idx].Low - 0.05,
			Sl:        int(data[idx].Low * 0.01),
			Tp:        int(data[idx].Low * 0.02),
			Quantity:  CalculatePosition(data[idx].High, data[idx].High-data[idx].High*0.01, client),
			OrderType: "SELL",
		}

		fmt.Printf("order placed: %v\n", order)
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

func TrackOrders(client *smartapigo.Client, symbol string) {
	for {
		//orders, _ := client.GetOrderBook()
		time.Sleep(1 * time.Second)
		positions, _ := client.GetPositions()
		isAnyPostionOpen := false
		totalPL := 0.0
		fmt.Printf("\n*************** Positions ************** \n")
		isPrint := true
		for _, postion := range positions {
			if isPrint {
				fmt.Printf("\n%v\n", postion)
				isPrint = false
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