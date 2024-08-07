package strategy

import (
	"fmt"
	"github.com/TredingInGo/AutomationService/smartStream"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/TredingInGo/smartapi/models"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

func (s *strategy) DCAlgo(ltp smartStream.SmartStream, client *smartapigo.Client) {
	maxTrade := 5
	for {
		isClosed := CloseSession(client)
		if isClosed || maxTrade == 0 {
			log.Printf("Todays Session Closed")
			return
		}
		if IsOptionPostionOpen(client) {
			continue
		}
		s.ExecuteDCAlgo(ltp, niftyExpairy, "NIFTY", client, &maxTrade)

	}
}
func (s *strategy) ExecuteDCAlgo(ltp smartStream.SmartStream, expiry, index string, client *smartapigo.Client, maxTrade *int) {

	index = strings.ToUpper(index)
	expiry = strings.ToUpper(expiry)
	userProfile, _ := client.GetUserProfile()
	var token = ""
	if index == "NIFTY" {
		token = niftyToken
	} else if index == "BANKNIFTY" {
		token = bankNiftyToken
	}
	candles, ATMstrike := getATMStrikeFoDc(client, index)
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
	order := DcForFo(indexDataWithIndicators, callDataWithIndicators, putDataWithIndicators, callToken, putToken, callSymbol, putSymbol, userProfile.UserName, client, int(callSpot), int(putSpot))
	log.Println(order)
	if order.orderType == "None" || order.quantity < 1 {
		return
	}

	var orderInfo ORDER
	if order.orderType == "BUY" {
		orderInfo = getDcOrderInfo(index, order)

	}
	orderParams := getDcOrderParams(orderInfo)
	isTradePlaced := s.PlaceDcOrder(ltp, client, orderParams, order.price)
	if isTradePlaced {
		*maxTrade--
	}

}

func getATMStrikeFoDc(client *smartapigo.Client, index string) ([]smartapigo.CandleResponse, float64) {
	//var candles []smartapigo.CandleResponse
	if index == "NIFTY" {
		candles := GetStockTick(client, niftyToken, "FIVE_MINUTE", nse)
		if len(candles) <= 100 {
			return []smartapigo.CandleResponse{}, 0.0
		}
		spot := candles[len(candles)-1].Close
		mf := int(spot) / int(nifty)
		atmStrike := float64(nifty * mf)
		return candles, atmStrike
	} else if index == "BANKNIFTY" {
		candles := GetStockTick(client, bankNiftyToken, "FIVE_MINUTE", nse)
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

func (s *strategy) PlaceDcOrder(ltp smartStream.SmartStream, client *smartapigo.Client, order smartapigo.OrderParams, spot float64) bool {
	orderRes, err := client.PlaceOrder(order)
	if err != nil {
		log.Printf("error: %v", err)
		return false
	}
	log.Printf("\n order res %v", orderRes)

	// Cancel Order if order is not full filled

	ordersList, _ := client.GetOrderBook()
	currentOrder := GetCurrentOrder(ordersList, orderRes.OrderID)
	for currentOrder.Status != "complete" {
		currentTick := GetStockTick(client, order.SymbolToken, "FIVE_MINUTE", nfo)
		lastTradedPrice := currentTick[len(currentTick)-1].Close
		if lastTradedPrice > spot+5 {
			cancleOrderResponce, _ := client.CancelOrder(order.Variety, orderRes.OrderID)
			fmt.Println("Order Cancelled", cancleOrderResponce)
			return false
		}
		ordersList, _ = client.GetOrderBook()
		currentOrder = GetCurrentOrder(ordersList, orderRes.OrderID)
	}
	s.TrackOrdersFoDc(ltp, client, order.SymbolToken, "User", orderRes.OrderID, order)
	return true

}
func getDcOrderParams(order ORDER) smartapigo.OrderParams {
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
		StopLoss:         float64(order.Sl),
		Quantity:         strconv.Itoa(order.Quantity),
		TrailingStopLoss: float64(1),
	}

	return orderParams
}

func getDcOrderInfo(index string, order LegInfo) ORDER {
	var sl int
	var tp int
	var lotSize int
	if index == "NIFTY" {
		sl = 5
		tp = 25
		lotSize = niftyLotSize
	} else if index == "BANKNIFTY" {
		sl = 10
		tp = 60
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

func DcForFo(data, callData, putData *DataWithIndicators, callToken, putToken, callSymbol, putSymbol, username string, client *smartapigo.Client, callStrike, putStrike int) LegInfo {
	idx := len(data.Data) - 1
	callIdx := len(callData.Data) - 1
	putIdx := len(putData.Data) - 1

	rsi := data.Indicators["rsi"+"14"]
	var order LegInfo
	order.orderType = "None"
	high, low := GetDCRange(*data, idx)

	if high == 0.0 || low == 1000000.0 {
		return order
	}

	if data.Data[idx-1].Close > high && rsi[idx] > 55 && rsi[idx] < 75 {
		log.Println(" CALL Trade taken on Dc BreakOut:")
		return LegInfo{
			price:     callData.Data[callIdx].Close + 0.5,
			strike:    callStrike,
			orderType: "BUY",
			token:     callToken,
			symbol:    callSymbol,
			quantity:  2,
		}

	} else if data.Data[idx-1].Close < low && rsi[idx] > 30 && rsi[idx] < 40 {
		log.Println(" PUT Trade taken on DC breakout ")
		return LegInfo{
			price:     putData.Data[putIdx].Close + 0.5,
			strike:    putStrike,
			orderType: "BUY",
			token:     putToken,
			symbol:    putSymbol,
			quantity:  2,
		}

	}

	return order
}

func GetDCRange(data DataWithIndicators, idx int) (float64, float64) {

	high := 0.0
	low := 1000000.0

	for i := idx - 1; i > idx-21; i-- {
		high = math.Max(data.Data[i].High, high)
		low = math.Min(data.Data[i].Low, low)
	}
	return high, low
}

func CalculateOBV(data DataWithIndicators) []int {
	var obv []int
	obv = append(obv, 0.0)
	for i := 1; i < len(data.Data); i++ {
		if data.Data[i-1].Close <= data.Data[i].Close {
			obv = append(obv, obv[len(obv)-1]+data.Data[i].Volume)
		}
		if data.Data[i-1].Close > data.Data[i].Close {
			obv = append(obv, obv[len(obv)-1]-data.Data[i].Volume)
		}
	}
	return obv
}
func IsOBVIncreasing(obv []int) bool {
	ma3 := 0.0
	ma9 := 0.0
	for i := len(obv) - 1; i > len(obv)-4; i-- {
		ma3 += float64(obv[i])
	}
	for i := len(obv) - 1; i > len(obv)-10; i-- {
		ma9 += float64(obv[i])
	}
	return ma3/3 > ma9/9

}

func IsOBVDecreasing(obv []int) bool {
	ma3 := 0.0
	ma9 := 0.0
	for i := len(obv) - 1; i > len(obv)-4; i-- {
		ma3 += float64(obv[i])
	}
	for i := len(obv) - 1; i > len(obv)-10; i-- {
		ma9 += float64(obv[i])
	}
	return ma3/3 < ma9/9

}

func (s *strategy) TrackOrdersFoDc(ltp smartStream.SmartStream, client *smartapigo.Client, symbol, userName, orderId string, order smartapigo.OrderParams) {
	var tokenInfo []models.TokenInfo
	tokenInfo = append(tokenInfo, models.TokenInfo{models.NSEFO, order.SymbolToken})

	price, _ := strconv.ParseFloat(order.Price, 64)
	target, _ := strconv.ParseFloat(order.SquareOff, 64)
	squareoff := price + target
	StopLoss := order.StopLoss
	trailingStopLoss := price - StopLoss
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
		if LTP >= price+5.0 {
			trailingStopLoss += 5.0
			price += 5.0
			for _, slOrder := range slOrders {
				modifyOrderParams := getModifyOrderParams(trailingStopLoss, slOrder, slOrder.OrderID, order.TradingSymbol)
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

		}
		if LTP <= trailingStopLoss || LTP >= squareoff {
			ltp.STOP()
			log.Println("Ltp stopped trailingStopLoss", trailingStopLoss, " LTP= ", LTP)
			break
		}
	}
}
func IsOptionPostionOpen(client *smartapigo.Client) bool {
	time.Sleep(1 * time.Second)
	positions, error := client.GetPositions()
	isAnyPostionOpen := false
	if error != nil {
		return true
	}
	totalPL := 0.0
	for _, postion := range positions {

		if postion.InstrumentType != "OPTIDX" {
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

func GetCurrentOrder(orders []smartapigo.Order, orderId string) smartapigo.Order {
	for _, order := range orders {
		if order.OrderID == orderId {
			return order
		}
	}
	return smartapigo.Order{}
}
