package strategy

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/TredingInGo/AutomationService/smartStream"
	smartapigo "github.com/TredingInGo/smartapi"
	"github.com/TredingInGo/smartapi/models"
	"log"
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
	Sl        float64
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
		for _, stock := range stockList {
			CloseSession(client)
			Execute(stock.Token, stock.Symbol, client, userProfile.UserName)
		}
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
	time.Sleep(1 * time.Second)
	data := GetStockTick(client, stockToken, "FIVE_MINUTE", nse)
	fmt.Println("Data length: ", len(data))
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

func (s *strategy) PlaceOrder(ltp smartStream.SmartStream, ctx context.Context, client *smartapigo.Client, orderParams smartapigo.OrderParams, userName, symbol string, spot float64) bool {
	log.Printf("\norder params: for %v \n%v\n", userName, orderParams)
	orderRes, err := client.PlaceOrder(orderParams)
	if err != nil {
		log.Printf("error: %v", err)
		return false
	}
	log.Printf("\n order res %v", orderRes)
	time.Sleep(1 * time.Second)
	orders, _ := client.GetOrderBook()
	currentOrder := GetOrderDetailsByOrderId(orderRes.OrderID, orders)
	for currentOrder.Status != "complete" {
		fmt.Println("CurrentOrderStatus = ", currentOrder.Status)
		if currentOrder.Status == "rejected" {
			return false
		}

		currentTick := GetStockTick(client, orderParams.SymbolToken, "FIVE_MINUTE", "NSE")
		if currentTick == nil || len(currentTick) == 0 {
			continue
		}
		lastTradedPrice := currentTick[len(currentTick)-1].Close
		if lastTradedPrice > spot+(spot*0.05) {
			cancleOrderResponce, _ := client.CancelOrder(orderParams.Variety, orderRes.OrderID)
			fmt.Println("Order Cancelled", cancleOrderResponce)
			return false
		}
		time.Sleep(1 * time.Second)
		orders, _ = client.GetOrderBook()
		currentOrder = GetCurrentOrder(orders, orderRes.OrderID)
		time.Sleep(10 * time.Second)
	}

	log.Printf("order response %v for %v", orderRes, userName)
	//s.TrackOrders(ltp, ctx, client, symbol, currentOrder.OrderID, orderParams)
	for IsEquityPostionOpen(client) {
		time.Sleep(5 * time.Second)
	}
	return true
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
		StopLoss:         float64(order.Sl),
		Quantity:         strconv.Itoa(order.Quantity),
		TrailingStopLoss: float64(2),
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
	StopLoss := order.StopLoss
	trailingStopLoss := price - StopLoss
	if order.TransactionType == "SELL" {
		trailingStopLoss = price + StopLoss
		squareoff = price - target
	}
	orderPrice := price
	time.Sleep(1 * time.Second)
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
		if (order.TransactionType == "BUY" && (LTP <= trailingStopLoss || LTP >= squareoff)) || (order.TransactionType == "SELL" && (LTP >= trailingStopLoss || LTP <= squareoff)) {
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
	}
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
	time.Sleep(5 * time.Second)
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
