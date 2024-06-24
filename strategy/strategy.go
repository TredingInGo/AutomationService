package strategy

import (
	"database/sql"
	"github.com/TredingInGo/AutomationService/historyData"
	"github.com/TredingInGo/AutomationService/smartStream"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/TredingInGo/smartapi/models"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	startTime        = "09:15"
	endTime          = "15:30"
	nifty            = 50
	bankNifty        = 100
	niftyToken       = "99926000"
	bankNiftyToken   = "99926009"
	call             = "CE"
	put              = "PE"
	nse              = "NSE"
	nfo              = "NFO"
	niftyLotSize     = 25
	bankNiftyLotSize = 15
	bankExpairy      = "26MAY24"
	niftyExpairy     = "27JUN24"
)

var (
	currTime = time.Now()
	baseTime = time.Date(currTime.Year(), currTime.Month(), currTime.Day(), 9, 0, 0, 0, time.Local)
)

type strategy struct {
	history  historyData.History
	pastData []smartapigo.CandleResponse

	LiveData    chan *models.SnapQuote
	chForCandle chan *models.SnapQuote
	db          *sql.DB
}

type LegInfo struct {
	price     float64
	strike    int
	orderType string
	token     string
	symbol    string
	quantity  int
}

type legs struct {
	leg1 LegInfo
	leg2 LegInfo
	leg3 LegInfo
	leg4 LegInfo
}

type priceInfo struct {
	price  float64
	token  string
	symbol string
	oi     uint64
}

func New() strategy {
	return strategy{

		LiveData:    make(chan *models.SnapQuote, 100),
		chForCandle: make(chan *models.SnapQuote, 100),
	}
}
func (s *strategy) Algo(ltp smartStream.SmartStream, client *smartapigo.Client) {
	maxTrade := 3
	for {
		isClosed := CloseSession(client)
		if isClosed || maxTrade == 0 {
			log.Printf("Todays Session Closed")
			return
		}
		s.ExecuteAlgo(ltp, bankExpairy, "BANKNIFTY", client, &maxTrade)
		s.ExecuteAlgo(ltp, niftyExpairy, "NIFTY", client, &maxTrade)

	}
}
func (s *strategy) ExecuteAlgo(ltp smartStream.SmartStream, expiry, index string, client *smartapigo.Client, maxTrade *int) {

	index = strings.ToUpper(index)
	expiry = strings.ToUpper(expiry)
	userProfile, _ := client.GetUserProfile()
	var token = ""
	if index == "NIFTY" {
		token = niftyToken
	} else if index == "BANKNIFTY" {
		token = bankNiftyToken
	}
	candles, ATMstrike := getATMStrike(client, index)
	if len(candles) <= 100 || *maxTrade == 0 {
		return
	}
	var ITMStrike int
	if index == "NIFTY" {
		ITMStrike = nifty * 4
	} else if index == "BANKNIFTY" {
		ITMStrike = bankNifty * 4
	}
	callSpot := ATMstrike - float64(ITMStrike)
	callSymbol := index + expiry + strconv.Itoa(int(callSpot)) + call
	callToken := GetFOToken(callSymbol, nfo)
	callSideITMTick := GetStockTick(client, callToken, "FIVE_MINUTE", nfo)
	if len(callSideITMTick) <= 100 {
		return
	}

	putSpot := ATMstrike + float64(ITMStrike)
	putSymbol := index + expiry + strconv.Itoa(int(putSpot)) + put
	putToken := GetFOToken(putSymbol, nfo)
	putSideITMTick := GetStockTick(client, putToken, "FIVE_MINUTE", nfo)
	if len(putSideITMTick) <= 100 {
		return
	}

	indexDataWithIndicators := &DataWithIndicators{
		Data:     candles,
		Token:    token,
		UserName: userProfile.UserName,
	}
	callDataWithIndicators := &DataWithIndicators{
		Data:     callSideITMTick,
		Token:    callToken,
		UserName: userProfile.UserName,
	}
	putDataWithIndicators := &DataWithIndicators{
		Data:     putSideITMTick,
		Token:    putToken,
		UserName: userProfile.UserName,
	}

	PopulateIndicators(indexDataWithIndicators)
	PopulateIndicators(callDataWithIndicators)
	PopulateIndicators(putDataWithIndicators)
	order := TrendFollowingRsiForFO(indexDataWithIndicators, callDataWithIndicators, putDataWithIndicators, callToken, putToken, callSymbol, putSymbol, userProfile.UserName, client, int(callSpot), int(putSpot))
	log.Println(order)
	if order.orderType == "None" || order.quantity < 1 {
		return
	}

	var orderInfo ORDER
	if order.orderType == "BUY" {
		orderInfo = getFOOrderInfo(index, order)

	}
	orderParams := getFOOrderParams(orderInfo)
	slOrder, isOrderPlaced := placeFOOrder(client, orderParams)
	if isOrderPlaced == false {
		//return
	}
	*maxTrade--
	var tokenInfo []models.TokenInfo
	tokenInfo = append(tokenInfo, models.TokenInfo{models.NSEFO, order.token})

	price := orderInfo.Spot
	trailingStopLoss := price - float64(orderInfo.Sl)

	target := price + float64(orderInfo.Tp)
	log.Println("Trying to connect with token ", tokenInfo)
	go ltp.Connect(s.LiveData, models.SNAPQUOTE, tokenInfo)
	for data := range s.LiveData {
		LTP := float64(data.LastTradedPrice / 100)
		if LTP >= price+10.0 {
			trailingStopLoss += 10
			price += 10
			modifyOrderParams := getModifyOrderParams(trailingStopLoss, orderParams, slOrder.OrderID)
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
		if LTP <= trailingStopLoss || LTP >= target {
			ltp.STOP()
			log.Println("Ltp stopped trailingStopLoss", trailingStopLoss, " LTP= ", LTP)
			break
		}
	}
	log.Println("LTP Streaming stopped")

}

func getATMStrike(client *smartapigo.Client, index string) ([]smartapigo.CandleResponse, float64) {
	//var candles []smartapigo.CandleResponse
	if index == "NIFTY" {
		candles := GetStockTick(client, niftyToken, "ONE_MINUTE", nse)
		if len(candles) <= 100 {
			return []smartapigo.CandleResponse{}, 0.0
		}
		spot := candles[len(candles)-1].Close
		mf := int(spot) / int(nifty)
		atmStrike := float64(nifty * mf)
		return candles, atmStrike
	} else if index == "BANKNIFTY" {
		candles := GetStockTick(client, bankNiftyToken, "ONE_MINUTE", nse)
		if len(candles) <= 100 {
			return []smartapigo.CandleResponse{}, 0.0
		}
		spot := candles[len(candles)-1].Close
		mf := int(spot) / int(bankNifty)
		atmStrike := float64(bankNifty * mf)
		return candles, atmStrike
	}
	return nil, 0
}

func placeFOOrder(client *smartapigo.Client, order smartapigo.OrderParams) (smartapigo.Order, bool) {
	orderRes, err := client.PlaceOrder(order)
	if err != nil {
		log.Printf("error: %v", err)
		return smartapigo.Order{}, false
	}
	log.Printf("\n order res %v", orderRes)
	orders, _ := client.GetOrderBook()
	orderDetails := getOrderDetailsByOrderId(orderRes.OrderID, orders)
	if orderDetails.OrderStatus != "complete" {
		time.Sleep(5000)
		orders, _ = client.GetOrderBook()
		orderDetails = getOrderDetailsByOrderId(orderRes.OrderID, orders)
		if orderDetails.OrderStatus != "complete" || orderDetails.OrderID != orderRes.OrderID {
			orderRes, _ := client.CancelOrder(order.Variety, orderRes.OrderID)
			log.Printf("Order Cancelled %v", orderRes)
			return orderDetails, false
		}
	}
	log.Printf("order placed %v", order)
	time.Sleep(1000)
	orders, _ = client.GetOrderBook()
	if orders == nil {
		return orderDetails, false
	}
	slOrder := getSLOrder(orders, order, orderRes.OrderID)
	log.Println("sl order", slOrder)
	return slOrder, true

}
func getFOOrderParams(order ORDER) smartapigo.OrderParams {
	orderParams := smartapigo.OrderParams{
		Variety:          "ROBO",
		TradingSymbol:    order.Symbol,
		SymbolToken:      order.Token,
		TransactionType:  order.OrderType,
		Exchange:         nfo,
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

func getFOOrderInfo(index string, order LegInfo) ORDER {
	var sl int
	var tp int
	var lotSize int
	if index == "NIFTY" {
		sl = 20
		tp = 60
		lotSize = niftyLotSize
	} else if index == "BANKNIFTY" {
		sl = 35
		tp = 105
		lotSize = bankNiftyLotSize
	}
	log.Printf("\nlot size %v \n", lotSize)
	orderParam := ORDER{
		Spot:      order.price,
		Sl:        sl,
		Tp:        tp,
		Quantity:  order.quantity * lotSize,
		OrderType: "BUY",
		Symbol:    order.symbol,
		Token:     order.token,
	}

	return orderParam

}

func TrendFollowingRsiForFO(data, callData, putData *DataWithIndicators, callToken, putToken, callSymbol, putSymbol, username string, client *smartapigo.Client, callStrike, putStrike int) LegInfo {
	idx := len(data.Data) - 1
	callIdx := len(callData.Data) - 1
	putIdx := len(putData.Data) - 1
	ema10 := data.Indicators["ema"+"10"]
	ema30 := data.Indicators["ema"+"30"]
	callEma7 := callData.Indicators["ema"+"7"][callIdx]
	callEma22 := callData.Indicators["ema"+"22"][callIdx]
	putEma7 := putData.Indicators["ema"+"7"][putIdx]
	putEma22 := putData.Indicators["ema"+"22"][putIdx]
	//firstCandle := GetFirstCandleOfToday(data.Data)
	var order LegInfo
	order.orderType = "None"
	if ema10[idx] > ema30[idx] && ema10[idx-1] <= ema10[idx-1] && callEma22 < callEma7 {
		log.Println(" CALL Trade taken on crossover: ema 10 = ", ema10, " ema30 = ", ema30, " callEma22 = ", callEma22, " callEma7 = ", callEma7)
		return LegInfo{
			price:     callData.Data[callIdx].High + 0.5,
			strike:    callStrike,
			orderType: "BUY",
			token:     callToken,
			symbol:    callSymbol,
			quantity:  1,
		}

	} else if ema10[idx] < ema30[idx] && ema10[idx-1] >= ema10[idx-1] && putEma22 > putEma7 {
		log.Println(" PUT Trade taken on crossover: ema 10 = ", ema10, " ema30 = ", ema30, " putEma22 = ", putEma22, " putEma7 = ", putEma7)
		return LegInfo{
			price:     putData.Data[putIdx].High + 0.5,
			strike:    putStrike,
			orderType: "BUY",
			token:     putToken,
			symbol:    putSymbol,
			quantity:  1,
		}

	}

	return order
}
func getOrderDetailsByOrderId(orderId string, orders smartapigo.Orders) smartapigo.Order {
	for i := 0; i < len(orders); i++ {
		if orders[i].OrderID == orderId {
			return orders[i]
		}
	}
	return smartapigo.Order{}
}

func getSLOrder(orders smartapigo.Orders, orderParams smartapigo.OrderParams, orderId string) smartapigo.Order {
	for i := 0; i < len(orders); i++ {
		sl := orders[i].Price
		price, _ := strconv.ParseFloat(orderParams.Price, 64)
		if orders[i].SymbolToken == orderParams.SymbolToken && sl < price && orders[i].OrderID > orderId && orders[i].OrderStatus != "complete" {
			return orders[i]
		}
	}
	orders[0].OrderID = "NA"
	return orders[0]
}

func getModifyOrderParams(sl float64, order smartapigo.OrderParams, orderId string) smartapigo.ModifyOrderParams {
	return smartapigo.ModifyOrderParams{
		Variety:       order.Variety,
		OrderID:       orderId,
		OrderType:     "STOPLOSS_LIMIT",
		ProductType:   order.ProductType,
		Duration:      order.Duration,
		Price:         strconv.FormatFloat(sl, 'f', 2, 64),
		Quantity:      order.Quantity,
		TradingSymbol: order.TradingSymbol,
		SymbolToken:   order.SymbolToken,
		Exchange:      nfo,
		TriggerPrice:  strconv.FormatFloat(sl, 'f', 2, 64),
	}
}

func GetFirstCandleOfToday(candles []smartapigo.CandleResponse) *smartapigo.CandleResponse {
	location, _ := time.LoadLocation("Asia/Kolkata") // Adjust timezone as per the input format
	today := time.Now().In(location).Format("2006-01-02")
	var firstCandle *smartapigo.CandleResponse

	// Loop through the slice in reverse order
	for i := len(candles) - 1; i >= 0; i-- {
		candle := candles[i]
		candleDate := candle.Timestamp.Format("2006-01-02")
		if candleDate == today {
			if firstCandle == nil || candle.Timestamp.Before(firstCandle.Timestamp) {
				firstCandle = &candle
			}
		}
	}

	return firstCandle
}
