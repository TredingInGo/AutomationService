package strategy

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/TredingInGo/AutomationService/smartStream"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/TredingInGo/smartapi/models"
	"log"
	"math"
	"strconv"
	"time"
)

type OrderDetails struct {
	Spot      float64
	Tp        int
	Sl        int
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

type ORDER struct {
	Spot      float64
	Sl        int
	Tp        int
	Quantity  int
	OrderType string
	Symbol    string
	Token     string
	Score     float64
}

func CloseSession(client *smartapigo.Client) bool {

	currentTime := time.Now()
	compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 0, 0, 0, currentTime.Location())
	userProfile, _ := client.GetUserProfile()
	if currentTime.After(compareTime) {
		client.Logout()
		log.Printf("Session closed  for %v", userProfile.UserName)
		return true
	}
	return false

}

func ForceCloseSession(client *smartapigo.Client) {
	client.Logout()
	userProfile, _ := client.GetUserProfile()
	log.Printf("Session closed  for %v", userProfile.UserName)
}

func (s *strategy) TrendFollowingStretgy(ltp smartStream.SmartStream, ctx context.Context, client *smartapigo.Client, db *sql.DB) {

	stockList := LoadStockList(db)
	userProfile, _ := client.GetUserProfile()
	//s.TrackOrders(ctx, client, "DUMMY", userProfile.UserName)

	for {
		//for _, stock := range stockList {
		//	CloseSession(client)
		//	Execute(stock.Token, stock.Symbol, client, userProfile.UserName)
		//}
		if IsEquityPostionOpen(client) {
			continue
		}
		isClosed := CloseSession(client)
		if isClosed {
			log.Printf("Todays Session Closed")
			return
		}

		select {
		case <-ctx.Done():
			log.Printf("context cancelled in TrendFollowingStretgy for user: %v\n", userProfile.UserName)
			return
		default:
		}

		s.getEligibleStocks(ltp, ctx, stockList, client, userProfile.UserName)

	}
}

func (s *strategy) getEligibleStocks(ltp smartStream.SmartStream, ctx context.Context, stocks []Symbols, client *smartapigo.Client, userName string) {

	log.Println("running getEligibleStocks for: ", userName, " at: ", time.Now().Format("2006-01-02 15:04:05"))
	for _, stock := range stocks {
		isClosed := CloseSession(client)
		if isClosed {
			log.Printf("Todays Session Closed")
			return
		}

		// check for context done
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled for user: %v\n", userName)
			return
		default:

		}

		param := EligibleStockParam{
			Symbols:  Symbols{Symbol: stock.Symbol, Token: stock.Token},
			UserName: userName,
		}

		order := Execute(param.Token, param.Symbol, client, param.UserName)
		if order != nil {
			orderParams := GetOrderParams(order)
			s.PlaceOrder(ltp, ctx, client, orderParams, userName, order.Symbol, order.Spot)
		}
	}

}

func Execute(symbol, stockToken string, client *smartapigo.Client, userName string) *ORDER {
	data := GetStockTick(client, stockToken, "FIVE_MINUTE", nse)
	if len(data) == 0 {
		return nil
	}

	dataWithIndicators := &DataWithIndicators{
		Data:     data,
		Token:    stockToken,
		UserName: userName,
	}

	PopulateIndicators(dataWithIndicators)
	order := DcForStocks(dataWithIndicators, stockToken, symbol, client)
	if order.OrderType == "None" || order.Quantity < 1 {
		return nil
	}

	return &order
}

func (s *strategy) PlaceOrder(ltp smartStream.SmartStream, ctx context.Context, client *smartapigo.Client, orderParams smartapigo.OrderParams, userName, symbol string, spot float64) bool {
	log.Printf("\norder params: for %v \n%v\n", userName, orderParams)
	orderRes, err := client.PlaceOrder(orderParams)
	if err != nil {
		log.Printf("error: %v", err)
		return false
	}
	log.Printf("\n order res %v", orderRes)
	orders, _ := client.GetOrderBook()
	currentOrder := GetOrderDetailsByOrderId(orderRes.OrderID, orders)
	for currentOrder.Status != "complete" {
		currentTick := GetStockTick(client, orderParams.SymbolToken, "FIVE_MINUTE", nfo)
		lastTradedPrice := currentTick[len(currentTick)-1].Close
		if lastTradedPrice > spot+(spot*0.05) {
			cancleOrderResponce, _ := client.CancelOrder(orderParams.Variety, orderRes.OrderID)
			fmt.Println("Order Cancelled", cancleOrderResponce)
			return false
		}
		orders, _ = client.GetOrderBook()
		currentOrder = GetCurrentOrder(orders, orderRes.OrderID)
	}

	log.Printf("order response %v for %v", orderRes, userName)
	s.TrackOrders(ltp, ctx, client, symbol, currentOrder.OrderID, orderParams)
	return true
}

func TrendFollowingRsi(data *DataWithIndicators, token, symbol, username string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	ma5 := data.Indicators["sma"+"5"][idx]
	ma8 := data.Indicators["sma"+"8"][idx]
	ma13 := data.Indicators["sma"+"13"][idx]
	ma21 := data.Indicators["sma"+"21"][idx]
	ema21 := data.Indicators["ema"+"21"][idx]
	rsi := data.Indicators["rsi"+"14"]
	adx14 := data.Adx["Adx"+"14"]
	rsiAvg3 := getAvg(rsi, 3)
	rsiavg8 := getAvg(rsi, 8)
	adxAvg3 := getAvg(adx14.Adx, 5)
	adxAvg8 := getAvg(adx14.Adx, 8)
	volAvg3 := getAvgVol(data.Data, 3)
	volAvg5 := getAvgVol(data.Data, 5)
	obv := CalculateOBV(*data)

	var order ORDER
	order.OrderType = "None"
	//log.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	//log.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if IsOBVIncreasing(obv) && data.Data[idx-1].Low > ema21 && data.Data[idx].Close > getVwap(data.Data, 14) && volAvg3 > volAvg5 && data.Data[idx].Volume > data.Data[idx-1].Volume && adxAvg3 > adxAvg8 && adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && ma5 > ma8 && ma8 > ma13 && ma21 < ma13 && rsi[idx] > 55 && rsi[idx] < 70 && rsiAvg3 > rsiavg8 {
		order = ORDER{
			Spot:      data.Data[idx].High + 0.05,
			Sl:        int(data.Data[idx].High * 0.01),
			Tp:        int(data.Data[idx].High * 0.02),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "BUY",
		}

	} else if !IsOBVIncreasing(obv) && data.Data[idx-1].High < ema21 && data.Data[idx].Close < getVwap(data.Data, 14) && volAvg3 > volAvg5 && data.Data[idx].Volume > data.Data[idx-1].Volume && adxAvg3 > adxAvg8 && adx14.Adx[idx] >= 20 && adx14.PlusDi[idx] < adx14.MinusDi[idx] && ma5 < ma8 && ma8 < ma13 && ma21 > ma13 && rsi[idx] < 35 && rsi[idx] > 70 && rsiAvg3 < rsiavg8 {
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

func DcForStocks(data *DataWithIndicators, token, symbol string, client *smartapigo.Client) ORDER {
	idx := len(data.Data) - 1
	rsi := data.Indicators["rsi"+"14"]
	adx := data.Adx["Adx20"].Adx
	ema21 := data.Indicators["ema"+"21"]
	var order ORDER
	order.OrderType = "None"

	high, low := GetDCRange(*data, idx)

	if high == 0.0 || low == 1000000.0 || rsi[idx] > 75 || rsi[idx] < 30 || adx[idx] < 25 {
		return order
	}

	if data.Data[idx].Close > high && ema21[idx] < data.Data[idx].Close {
		log.Println(" Buy Trade taken on Dc BreakOut:")
		order = ORDER{
			Spot:      data.Data[idx].Close + 0.05,
			Sl:        CalculateDynamicSL(high, low),
			Tp:        CalculateDynamicTP(high, low),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "BUY",
		}

	} else if data.Data[idx].Close < low && ema21[idx] > data.Data[idx].Close {
		log.Println(" SELL Trade taken on DC breakout ")
		order = ORDER{
			Spot:      data.Data[idx].Close - 0.05,
			Sl:        CalculateDynamicSL(high, low),
			Tp:        CalculateDynamicTP(high, low),
			Quantity:  CalculatePosition(data.Data[idx].High, data.Data[idx].High-data.Data[idx].High*0.01, client),
			OrderType: "SELL",
		}

	}
	order.Symbol = symbol
	order.Token = token

	return order
}

func CalculateDynamicSL(high, low float64) int {
	// Calculate dynamic stop-loss based on volatility or other criteria
	return int(math.Abs(high-low) * 0.5) // Example: 50% of the channel range
}

func CalculateDynamicTP(high, low float64) int {
	// Calculate dynamic take-profit based on volatility or other criteria
	return int(math.Abs(high-low) * 1.5) // Example: 150% of the channel range
}

func CalculateDynamicQuantity(close float64) int {
	// Calculate dynamic quantity based on risk management rules
	return int(100000 / close) // Example: allocate a fixed amount of capital per trade
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
		log.Println(err)
	}
	return amount
}

func (s *strategy) TrackOrders(ltp smartStream.SmartStream, ctx context.Context, client *smartapigo.Client, symbol, orderId string, order smartapigo.OrderParams) {

	var tokenInfo []models.TokenInfo
	tokenInfo = append(tokenInfo, models.TokenInfo{models.NSECM, order.SymbolToken})

	price, _ := strconv.ParseFloat(order.Price, 64)
	target, _ := strconv.ParseFloat(order.SquareOff, 64)
	squareoff := price + target
	StopLoss, _ := strconv.ParseFloat(order.StopLoss, 64)
	trailingStopLoss := price - StopLoss
	if order.OrderType == "SELL" {
		trailingStopLoss = price + StopLoss
		squareoff = price - target
	}
	orderPrice := price

	ordersList, _ := client.GetOrderBook()
	if ordersList == nil {
		return
	}

	slOrders := GetSLOrders(ordersList, order, orderId)
	go ltp.Connect(s.LiveData, models.SNAPQUOTE, tokenInfo)
	for data := range s.LiveData {
		LTP := float64(data.LastTradedPrice / 100.0)
		fmt.Println("P/L: - ", LTP-orderPrice)
		if LTP >= price+2.0 && order.OrderType == "BUY" {
			trailingStopLoss += 2.0
			price += 2.0
			for _, slOrder := range slOrders {
				modifyOrderParams := getModifyOrderParams(trailingStopLoss, slOrder, slOrder.OrderID, order.TradingSymbol)
				ModifyOrderWithRetry(modifyOrderParams, client)
			}

		}
		if (order.OrderType == "BUY" && (LTP <= trailingStopLoss || LTP >= squareoff)) || (order.OrderType == "SELL" && (LTP >= trailingStopLoss || LTP <= squareoff)) {
			ltp.STOP()
			log.Println("Ltp stopped trailingStopLoss", trailingStopLoss, " LTP= ", LTP)
			break
		} else if LTP <= price-2.0 && order.OrderType == "SELL" {
			trailingStopLoss -= 2.0
			price -= 2.0
			for _, slOrder := range slOrders {
				modifyOrderParams := getModifyOrderParams(trailingStopLoss, slOrder, slOrder.OrderID, order.TradingSymbol)
				ModifyOrderWithRetry(modifyOrderParams, client)
			}

		}
		if LTP >= trailingStopLoss || LTP <= squareoff {
			ltp.STOP()
			log.Println("Ltp stopped trailingStopLoss", trailingStopLoss, " LTP= ", LTP)
			break
		}
	}
}

func CalculatePosition(buyPrice, sl float64, client *smartapigo.Client) int {
	Amount := math.Min(20000.0, GetAmount(client))
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
	//score += calculateAtrScore(data.Data, order)
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
	ROC := ((data[currentIdx].Close - data[currentIdx-5].Close) / data[currentIdx-5].Close) * 100
	if orderType == "BUY" {
		score = math.Max(0, ROC*2.0)
	}
	if orderType == "SELL" {
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

func calculateAtrScore(data []smartapigo.CandleResponse, order ORDER) float64 {
	if order.OrderType == "None" {
		return 0
	}
	currentIdx := len(data) - 1
	atr := CalculateAtr(data, 14, "atr14")
	if len(atr) < 14 {
		return 0
	}
	if atr[currentIdx] >= float64(order.Tp) {
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

func getAvgVol(data []smartapigo.CandleResponse, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	sum := 0.0
	for i := len(data) - 1; i >= len(data)-period; i-- {
		sum += float64(data[i].Volume)
	}
	return sum / float64(period)
}

func getVwap(data []smartapigo.CandleResponse, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	cumTypical := 0.0
	cumVol := 0.0
	for i := len(data) - 1; i >= len(data)-period; i-- {
		cumVol += float64(data[i].Volume)
		cumTypical = cumTypical + (((data[i].High + data[i].Low + data[i].Close) / 3) * float64(data[i].Volume))
	}
	return cumTypical / cumVol
}
func ModifyOrderWithRetry(modifyOrderParams smartapigo.ModifyOrderParams, client *smartapigo.Client) {
	orderRes, err1 := client.ModifyOrder(modifyOrderParams)
	for i := 0; i < 3; i++ {
		if err1 == nil {
			break
		}
		log.Printf("\n Error in modifying SL: %v  retry -> %v \n", err1, i+1)
		orderRes, err1 = client.ModifyOrder(modifyOrderParams)
	}
	if err1 != nil {
		log.Printf("\n Error in modifying SL: %v \n", err1)
	} else {
		log.Printf("SL Modified %v", orderRes)
	}
}

func IsEquityPostionOpen(client *smartapigo.Client) bool {
	time.Sleep(1 * time.Second)
	positions, error := client.GetPositions()
	isAnyPostionOpen := false
	if error != nil {
		return true
	}
	totalPL := 0.0
	for _, postion := range positions {

		if postion.InstrumentType == "OPTIDX" {
			continue
		}
		qty, err := strconv.Atoi(postion.NetQty)
		if err != nil {
			isAnyPostionOpen = true
			continue
		}
		if qty != 0 {
			isAnyPostionOpen = true
		}
		val, err2 := strconv.ParseFloat(postion.NetValue, 64)
		if err2 != nil {
			isAnyPostionOpen = true
			continue
		}
		totalPL += val
	}

	return isAnyPostionOpen

}
