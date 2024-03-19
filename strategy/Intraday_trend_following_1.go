package strategy

import (
	"database/sql"
	"fmt"
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	worker = 20
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

type EligibleStockParam struct {
	Symbols
	UserName string
}

type DataWithIndicators struct {
	Data       []smartapigo.CandleResponse
	Indicators map[string][]float64
	StoArray   map[string][]StoField
	Adx        map[string]ADX
	Token      string
	UserName   string
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
	//TrackOrders(client, "DUMMY", userProfile.UserName)

	for {
		//for _, stock := range stockList {
		//	CloseSession(client)
		//	Execute(stock.Token, stock.Symbol, client, userProfile.UserName)
		//}

		eligibleStocks := getEligibleStocks(stockList, client, userProfile.UserName)
		// get most eligible stock for trade

		sort.Slice(eligibleStocks, func(i, j int) bool {
			return eligibleStocks[i].Score > eligibleStocks[j].Score
		})
		for i := 0; i < len(eligibleStocks)-1; i++ {
			fmt.Printf("stock %v : %v\n", i+1, *eligibleStocks[i])
		}
		orderParams := GetOrderParams(eligibleStocks[0])
		// place order for the most eligible stock
		PlaceOrder(client, orderParams, userProfile.UserName, eligibleStocks[0].Symbol)

		time.Sleep(10 * time.Second)
	}
}

func getEligibleStocks(stocks []Symbols, client *smartapigo.Client, userName string) []*ORDER {
	inp := make(chan *EligibleStockParam, 1000)
	out := make(chan *ORDER, 1000)
	orders := []*ORDER{}

	wg := sync.WaitGroup{}

	go func() {
		for i := 0; i < worker; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for param := range inp {
					order := Execute(param.Symbol, param.Token, client, param.UserName)
					if order != nil {
						out <- order
					}
				}
			}()
		}
	}()

	go func() {
		for _, stock := range stocks {
			inp <- &EligibleStockParam{
				Symbols:  Symbols{Symbol: stock.Symbol, Token: stock.Token},
				UserName: userName,
			}
		}
		close(inp)
	}()

	// close output channel after all the workers are done
	go func() {
		wg.Wait()
		close(out)
	}()

	for order := range out {
		orders = append(orders, order)
	}

	return orders
}

func Execute(symbol, stockToken string, client *smartapigo.Client, userName string) *ORDER {
	data := GetStockTick(client, stockToken, "FIVE_MINUTE")
	if len(data) == 0 {
		return nil
	}

	dataWithIndicators := &DataWithIndicators{
		Data:     data,
		Token:    stockToken,
		UserName: userName,
	}

	PopulateIndicators(dataWithIndicators)
	order := TrendFollowingRsi(dataWithIndicators, stockToken, symbol, userName, client)
	if order.OrderType == "None" || order.Quantity < 1 {
		return nil
	}

	return &order
}

func PlaceOrder(client *smartapigo.Client, orderParams smartapigo.OrderParams, userName, symbol string) {
	fmt.Printf("\norder params: for %v \n%v\n", userName, orderParams)
	//orderRes, _ := client.PlaceOrder(orderParams)
	//fmt.Printf("order response %v for %v", orderRes, userName)
	//TrackOrders(client, symbol, userName)
}

func TrendFollowingRsi(data *DataWithIndicators, token, symbol, username string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	sma5 := data.Indicators["sma"+"5"][idx]
	sma8 := data.Indicators["sma"+"8"][idx]
	sma13 := data.Indicators["sma"+"13"][idx]
	sma21 := data.Indicators["sma"+"21"][idx]
	adx14 := data.Adx["adx"+"14"]
	rsi := data.Indicators["rsi"+"14"]
	var order ORDER
	order.OrderType = "None"
	fmt.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	fmt.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma5 > sma8 && sma8 > sma13 && sma13 > sma21 && rsi[idx] < 70 && rsi[idx] > 60 && rsi[idx-2] < rsi[idx] && rsi[idx-1] < rsi[idx] {
		order = ORDER{
			Spot:      data.Data[idx].High + 0.05,
			Sl:        int(data.Data[idx].High * 0.01),
			Tp:        int(data.Data[idx].High * 0.02),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "BUY",
		}

	} else if adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] < adx14.MinusDi[idx] && sma5 < sma8 && sma8 < sma13 && sma13 < sma21 && rsi[idx] < 40 && rsi[idx] > 30 && rsi[idx-2] > rsi[idx] && rsi[idx-1] > rsi[idx] {
		order = ORDER{
			Spot:      data.Data[idx].Low - 0.05,
			Sl:        int(data.Data[idx].Low * 0.01),
			Tp:        int(data.Data[idx].Low * 0.02),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "SELL",
		}

	}
	order.Score = CaluclateScore(data, order)
	order.Symbol = symbol
	order.Token = token

	return order
}

func GetOrderParams(order *ORDER) smartapigo.OrderParams {

	orderParams := smartapigo.OrderParams{
		Variety:          "ROBO",
		TradingSymbol:    order.Symbol + "-EQ",
		SymbolToken:      order.Token,
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

			qty, err := strconv.Atoi(postion.NetQty)
			if err != nil {
				isAnyPostionOpen = true
				continue
			}
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
	Amount -= 500
	if Amount/buyPrice <= 1 {
		return 0
	}
	return int(Amount/buyPrice) * 5
}

func CaluclateScore(data *DataWithIndicators, order ORDER) float64 {
	score := 0.0
	score += calculateDirectionalStrength(data.Data, order.OrderType)
	score += calculateROC(data.Data, order.OrderType)
	score += calculateVolumeSocre(data.Data, order.OrderType)
	score += calculateLongerTimePeriodDirectionalScore(data.Data, order.OrderType)
	score += calculateAtrScore(data.Data, order.OrderType)
	return score
}

func calculateDirectionalStrength(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" {
		return 0.0
	}
	var count = 0.0
	if orderType == "BUY" {
		for i := len(data) - 1; i >= len(data)-10; i-- {
			if data[i].Open > data[i].Close {
				count++
			}
		}
	} else if orderType == "SELL" {
		for i := len(data) - 1; i >= len(data)-10; i-- {
			if data[i].Open < data[i].Close {
				count++
			}
		}
	}

	return count
}

func calculateROC(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" {
		return 0.0
	}
	currentIdx := len(data) - 1
	score := 0.0
	ROC := ((data[currentIdx].Close - data[currentIdx-13].Close) / data[currentIdx-13].Close) * 100
	if orderType == "BUY" {
		score = math.Max(0, ROC*2.0)
	}
	if order.orderType == "SELL" {
		score = math.Max(0, math.Abs(ROC*2.0))
	}
	return score
}

func calculateVolumeSocre(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" {
		return 0.0
	}
	currentIdx := len(data) - 1
	score := 0.0
	if orderType == "BUY" {
		if data[currentIdx].Volume > data[currentIdx-1].Volume && data[currentIdx].Close > data[currentIdx-1].Close {
			score += 7
		}
		if data[currentIdx-1].Volume > data[currentIdx-2].Volume && data[currentIdx-1].Close > data[currentIdx-2].Close {
			score += 3
		}
	} else if orderType == "SELL" {

		if data[currentIdx].Volume > data[currentIdx-1].Volume && data[currentIdx].Close < data[currentIdx-1].Close {
			score += 7
		}
		if data[currentIdx-1].Volume > data[currentIdx-2].Volume && data[currentIdx-1].Close < data[currentIdx-2].Close {
			score += 3
		}
	}

	return score
}

func calculateLongerTimePeriodDirectionalScore(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" {
		return 0
	}
	closePrice := GetClosePriceArray(data)
	if len(closePrice) <= 80 {
		return 0
	}

	currentIdx := len(data) - 1
	score := 0.0
	ema50 := CalculateEma(closePrice, 50)
	ema80 := CalculateEma(closePrice, 80)
	ema30 := CalculateEma(closePrice, 30)
	if orderType == "BUY" {
		if ema30[currentIdx] > ema50[currentIdx] {
			score += 4
		}
		if ema50[currentIdx] > ema80[currentIdx] {
			score += 4
		}
		if closePrice[currentIdx-20] < closePrice[currentIdx] {
			score += 2
		}
	} else if orderType == "SELL" {
		if ema30[currentIdx] < ema50[currentIdx] {
			score += 4
		}
		if ema50[currentIdx] < ema80[currentIdx] {
			score += 4
		}
		if closePrice[currentIdx-20] > closePrice[currentIdx] {
			score += 2
		}
	}
	return score
}

func calculateAtrScore(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" {
		return 0
	}
	currentIdx := len(data) - 1
	atr := CalculateAtr(data, 14, "atr14")
	if len(atr) < 14 {
		return 0
	}
	if atr[currentIdx] >= order.tp {
		return 12
	} else {
		return 0
	}
}
