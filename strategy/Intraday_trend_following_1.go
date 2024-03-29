package strategy

import (
	"database/sql"
	"fmt"
	smartapigo "github.com/TredingInGo/smartapi"
	"math"
	"sort"
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

func CloseSession(client *smartapigo.Client) bool {

	currentTime := time.Now()
	compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 0, 0, 0, currentTime.Location())
	userProfile, _ := client.GetUserProfile()
	if currentTime.After(compareTime) {
		client.Logout()
		fmt.Printf("Session closed  for %v", userProfile.UserName)
		return true
	}
	return false

}

func ForceCloseSession(client *smartapigo.Client) {
	client.Logout()
	userProfile, _ := client.GetUserProfile()
	fmt.Printf("Session closed  for %v", userProfile.UserName)
}

func TrendFollowingStretgy(client *smartapigo.Client, db *sql.DB) {

	stockList := LoadStockList(db)
	userProfile, _ := client.GetUserProfile()
	TrackOrders(client, "DUMMY", userProfile.UserName)

	for {
		//for _, stock := range stockList {
		//	CloseSession(client)
		//	Execute(stock.Token, stock.Symbol, client, userProfile.UserName)
		//}
		isClosed := CloseSession(client)
		if isClosed {
			fmt.Printf("Todays Session Closed")
			return
		}
		eligibleStocks := getEligibleStocks(stockList, client, userProfile.UserName)
		// get most eligible stock for trade

		sort.Slice(eligibleStocks, func(i, j int) bool {
			return eligibleStocks[i].Score > eligibleStocks[j].Score
		})

		for i := 0; i < len(eligibleStocks); i++ {
			fmt.Printf("stock %v : %v\n", i+1, *eligibleStocks[i])
		}

		if len(eligibleStocks) > 0 {
			orderParams := GetOrderParams(eligibleStocks[0])
			PlaceOrder(client, orderParams, userProfile.UserName, eligibleStocks[0].Symbol)
		}
	}
}

func getEligibleStocks(stocks []Symbols, client *smartapigo.Client, userName string) []*ORDER {
	filteredStocks := []*ORDER{}
	orders := []*ORDER{}
	start := time.Now()
	for _, stock := range stocks {
		param := EligibleStockParam{
			Symbols:  Symbols{Symbol: stock.Symbol, Token: stock.Token},
			UserName: userName,
		}

		order := Execute(param.Token, param.Symbol, client, param.UserName)
		if order != nil {
			filteredStocks = append(filteredStocks, order)
		}
	}

	fmt.Println("Time to filter stocks ", time.Since(start))

	start = time.Now()
	for _, stock := range filteredStocks {
		order := Execute(stock.Symbol, stock.Token, client, userName)
		if order != nil {
			orders = append(orders, order)
		}
	}

	fmt.Println("Time to get orders ", time.Since(start))

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
	orderRes, _ := client.PlaceOrder(orderParams)
	fmt.Printf("order response %v for %v", orderRes, userName)
	TrackOrders(client, symbol, userName)
}

func TrendFollowingRsi(data *DataWithIndicators, token, symbol, username string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	sma5 := data.Indicators["sma"+"5"][idx]
	sma8 := data.Indicators["sma"+"8"][idx]
	sma13 := data.Indicators["sma"+"13"][idx]
	sma21 := data.Indicators["sma"+"21"][idx]
	rsi := data.Indicators["rsi"+"14"]
	adx14 := data.Adx["Adx"+"14"]
	rsiAvg5 := getAvg(rsi, 5)
	rsiavg8 := getAvg(rsi, 8)
	adxAvg5 := getAvg(adx14.Adx, 5)
	adxAvg8 := getAvg(adx14.Adx, 8)
	var order ORDER
	order.OrderType = "None"
	fmt.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	//fmt.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if adxAvg5 > adxAvg8 && adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma5 > sma8 && sma8 > sma13 && sma21 > sma13 && rsi[idx] > 55 && rsi[idx] < 65 && rsiAvg5 > rsiavg8 {
		order = ORDER{
			Spot:      data.Data[idx].High + 0.05,
			Sl:        int(data.Data[idx].High * 0.01),
			Tp:        int(data.Data[idx].High * 0.02),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "BUY",
		}

	} else if adxAvg5 > adxAvg8 && adx14.Adx[idx] >= 20 && adx14.PlusDi[idx] < adx14.MinusDi[idx] && sma5 < sma8 && sma8 < sma13 && sma21 > sma13 && rsi[idx] < 40 && rsi[idx] > 30 && rsiAvg5 < rsiavg8 {
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
				ForceCloseSession(client)
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
	return int(Amount/buyPrice) * 4
}

func CaluclateScore(data *DataWithIndicators, order ORDER) float64 {
	score := 0.0
	//score += calculateDirectionalStrength(data.Data, order.OrderType)
	//score += calculateROC(data.Data, order.OrderType)
	score += calculateVolumeSocre(data.Data, order.OrderType)
	//score += calculateLongerTimePeriodDirectionalScore(data.Data, order.OrderType)
	score += calculateAtrScore(data.Data, order.OrderType)
	return score
}

func calculateDirectionalStrength(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" || len(data) < 10 {
		return 0.0
	}

	var count = 0.0
	if orderType == "BUY" {
		for i := len(data) - 1; i >= len(data)-10; i-- {
			if data[i].Open > data[i].Close {
				count++
			}
		}

		return count
	}

	// for sell type
	for i := len(data) - 1; i >= len(data)-10; i-- {
		if data[i].Open < data[i].Close {
			count++
		}
	}

	return count
}

func calculateROC(data []smartapigo.CandleResponse, orderType string) float64 {
	if orderType == "None" || len(data) < 14 {
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
	if orderType == "None" || len(data) < 2 {
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

func getAvg(data []float64, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	sum := 0.0
	for i := len(data) - 1; i >= len(data)-period; i-- {
		sum += data[i]
	}
	return sum / float64(period)
}
