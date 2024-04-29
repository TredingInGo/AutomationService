package strategy

import (
	"database/sql"
	"fmt"
	"github.com/TredingInGo/AutomationService/historyData"
	"github.com/TredingInGo/AutomationService/smartStream"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/TredingInGo/smartapi/models"
	"strconv"
	"strings"
	"time"
)

const (
	startTime      = "09:15"
	endTime        = "15:30"
	nifty          = 50
	bankNifty      = 100
	niftyToken     = "99926000"
	bankNiftyToken = "99926009"
	call           = "CE"
	put            = "PE"
	nse            = "NSE"
	nfo            = "NFO"
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

func (s *strategy) Algo(ltp smartStream.SmartStream, expiry, index string, client *smartapigo.Client) {

	maxTrade := 2
	index = strings.ToUpper(index)
	expiry = strings.ToUpper(expiry)
	userProfile, _ := client.GetUserProfile()
	var token = ""
	if index == "NIFTY" {
		token = niftyToken
	} else if index == "BANKNIFTY" {
		token = bankNiftyToken
	}
	for {
		candles, ATMstrike := getATMStrike(client, index)
		if candles == nil || maxTrade == 0 {
			return
		}
		var ITMStrike int
		if index == "NIFTY" {
			ITMStrike = nifty * 2
		} else if index == "BANKNIFTY" {
			ITMStrike = nifty * 2
		}
		callSpot := ATMstrike - float64(ITMStrike)
		callSymbol := index + expiry + strconv.Itoa(int(callSpot)) + call
		callToken := GetFOToken(callSymbol, nfo)
		callSideITMTick := GetStockTick(client, callToken, "FIVE_MINUTE", nfo)

		putSpot := ATMstrike + float64(ITMStrike)
		putSymbol := index + expiry + strconv.Itoa(int(putSpot)) + put
		putToken := GetFOToken(putSymbol, nfo)
		putSideITMTick := GetStockTick(client, putToken, "FIVE_MINUTE", nfo)

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
		if order.orderType == "None" || order.quantity < 1 {
			continue
		}

		var orderInfo ORDER
		var placedToken string
		if order.orderType == "BUY" {
			orderInfo = getFOOrderInfo(index, order)
			placedToken = callToken
		}

		if order.orderType == "SELL" {
			orderInfo = getFOOrderInfo(index, order)
			placedToken = putToken
		}

		orderParams := getFOOrderParams(orderInfo)
		slOrder, isOrderPlaced := placeFOOrder(client, orderParams)
		if isOrderPlaced == false {
			continue
		}
		maxTrade--
		var tokenInfo []models.TokenInfo
		tokenInfo = append(tokenInfo, models.TokenInfo{models.NSEFO, placedToken})
		sl := orderInfo.Sl
		price := orderInfo.Spot
		go ltp.Connect(s.LiveData, models.SNAPQUOTE, tokenInfo)
		for data := range s.LiveData {
			LTP := float64(data.LastTradedPrice / 100)
			slPrice, err := strconv.ParseFloat(slOrder.Price, 64)
			if err != nil {
				continue
			}
			if LTP > price-float64(sl)+20 {

				sl = sl + 10
				stopLossPrice := slPrice + float64(sl)
				modifyOrderParams := getModifyOrderParams(stopLossPrice, orderParams, slOrder.OrderID)
				orderRes, _ := client.ModifyOrder(modifyOrderParams)
				fmt.Printf("SL Modified %v", orderRes)
			}
			if LTP <= slPrice {
				ltp.STOP()
			}
		}

	}

}

func getATMStrike(client *smartapigo.Client, index string) ([]smartapigo.CandleResponse, float64) {
	//var candles []smartapigo.CandleResponse
	if index == "NIFTY" {
		candles := GetStockTick(client, niftyToken, "FIVE_MINUTE", nse)
		spot := candles[len(candles)-1].Close
		mf := int(spot) / int(nifty)
		atmStrike := float64(nifty * mf)
		return candles, atmStrike
	} else if index == "BANKNIFTY" {
		candles := GetStockTick(client, bankNiftyToken, "FIVE_MINUTE", nse)
		spot := candles[len(candles)-1].Close
		mf := int(spot) / int(bankNifty)
		atmStrike := float64(nifty * mf)
		return candles, atmStrike
	}
	return nil, 0
}

func placeFOOrder(client *smartapigo.Client, order smartapigo.OrderParams) (smartapigo.Order, bool) {
	orderRes, err := client.PlaceOrder(order)
	if err != nil {
		fmt.Printf("error: %v", err)
		return smartapigo.Order{}, false
	}
	orders, _ := client.GetOrderBook()
	orderDetails := getOrderDetailsByOrderId(orderRes.OrderID, orders)
	if orderDetails.OrderStatus != "complete" || orderDetails.OrderID != orderRes.OrderID {
		time.Sleep(5000)
		orders, _ = client.GetOrderBook()
		orderDetails = getOrderDetailsByOrderId(orderRes.OrderID, orders)
		if orderDetails.OrderStatus != "complete" || orderDetails.OrderID != orderRes.OrderID {
			orderRes, _ := client.CancelOrder(order.Variety, orderRes.OrderID)
			fmt.Printf("Order Cancelle %v", orderRes)
			return orderDetails, false
		}
	}
	orders, _ = client.GetOrderBook()
	slOrder := getSLOrder(orders, order, orderRes.OrderID)
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
		tp = 80
		lotSize = nifty / 2
	} else if index == "BANKNIFTY" {
		sl = 30
		tp = 120
		lotSize = bankNifty / 2
	}
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
	sma5 := data.Indicators["sma"+"5"][idx]
	sma8 := data.Indicators["sma"+"8"][idx]
	sma13 := data.Indicators["sma"+"13"][idx]
	sma21 := data.Indicators["sma"+"21"][idx]
	ema8 := data.Indicators["ema"+"8"][idx]
	rsi := data.Indicators["rsi"+"14"]
	adx14 := data.Adx["Adx"+"14"]
	rsiAvg3 := getAvg(rsi, 3)
	rsiavg8 := getAvg(rsi, 8)
	adxAvg3 := getAvg(adx14.Adx, 5)
	adxAvg8 := getAvg(adx14.Adx, 8)
	callEma7 := callData.Indicators["ema"+"7"][callIdx]
	callEma22 := callData.Indicators["ema"+"22"][callIdx]
	putEma7 := putData.Indicators["ema"+"7"][putIdx]
	putEma22 := putData.Indicators["ema"+"22"][putIdx]
	callRsi := callData.Indicators["rsi"+"14"][callIdx]
	putRsi := putData.Indicators["rsi"+"14"][putIdx]

	var order LegInfo
	order.orderType = "None"
	//fmt.Printf("\nStock Name: %v UserName %v\n", symbol, username)
	//fmt.Printf("currentTime:%v, currentData:%v, adx = %v, sma5 = %v, sma8 = %v, sma13 = %v, sma21 = %v, rsi = %v,  name = %v ", time.Now(), data.Data[idx], adx14.Adx[idx], sma5, sma8, sma13, sma21, rsi[idx], username)
	if data.Data[idx-1].Low > ema8 && callData.Data[idx].Close > getVwap(callData.Data, 14) && callData.Data[idx].Volume > callData.Data[idx-1].Volume && adxAvg3 > adxAvg8 && adx14.Adx[idx] >= 25 && adx14.PlusDi[idx] > adx14.MinusDi[idx] && sma5 > sma8 && sma8 > sma13 && sma21 < sma13 && rsi[idx] > 55 && rsi[idx] < 70 && rsiAvg3 > rsiavg8 && callEma7 > callEma22 && callRsi > 55 && callRsi <= 70 {
		order = LegInfo{
			price:     callData.Data[idx].High + 0.5,
			strike:    callStrike,
			orderType: "BUY",
			token:     callToken,
			symbol:    callSymbol,
			quantity:  2,
		}

	} else if data.Data[idx-1].High < ema8 && putData.Data[idx].Close < getVwap(putData.Data, 14) && putData.Data[idx].Volume > putData.Data[idx-1].Volume && adxAvg3 > adxAvg8 && adx14.Adx[idx] >= 20 && adx14.PlusDi[idx] < adx14.MinusDi[idx] && sma5 < sma8 && sma8 < sma13 && sma21 > sma13 && rsi[idx] < 40 && rsi[idx] > 30 && rsiAvg3 < rsiavg8 && putEma7 > putEma22 && putRsi > 55 && putRsi <= 70 {
		order = LegInfo{
			price:     putData.Data[idx].High + 0.5,
			strike:    putStrike,
			orderType: "BUY",
			token:     putToken,
			symbol:    putSymbol,
			quantity:  2,
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
	return orders[0]
}

func getSLOrder(orders smartapigo.Orders, orderParams smartapigo.OrderParams, orderId string) smartapigo.Order {
	for i := 0; i < len(orders); i++ {
		sl, _ := strconv.Atoi(orders[i].Price)
		price, _ := strconv.Atoi(orderParams.Price)
		if orders[i].SymbolToken == orderParams.SymbolToken && sl < price && orders[i].OrderID != orderId {
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
		OrderType:     "SELL",
		ProductType:   order.ProductType,
		Duration:      order.Duration,
		Price:         strconv.FormatFloat(sl, 'f', 2, 64),
		Quantity:      order.Quantity,
		TradingSymbol: order.TradingSymbol,
		SymbolToken:   order.SymbolToken,
		Exchange:      nfo,
	}
}
