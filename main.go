package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TredingInGo/AutomationService/Simulation"
	"github.com/TredingInGo/AutomationService/historyData"
	intra_day "github.com/TredingInGo/AutomationService/http/intra-day"
	"github.com/TredingInGo/AutomationService/http/session"
	"github.com/TredingInGo/AutomationService/http/start"
	"github.com/TredingInGo/AutomationService/http/stop"
	"github.com/TredingInGo/AutomationService/smartStream"
	"github.com/TredingInGo/AutomationService/strategy"
	"github.com/TredingInGo/AutomationService/strategy/BackTest"
	"github.com/TredingInGo/AutomationService/user"
	"github.com/TredingInGo/AutomationService/utils"
	smartapi "github.com/TredingInGo/smartapi"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

var (

	//apiClient                            *smartapi.Client
	//session                              smartapi.UserSession
	//err                                  error
	userSessions = make(map[string]*clientSession)
)

type clientSession struct {
	apiClient  *smartapi.Client
	session    smartapi.UserSession
	cancelFunc context.CancelFunc
}

func main() {
	fmt.Println("Starting the server, time: ", time.Now())
	mutex := sync.Mutex{}

	defer func() {
		recover()
	}()

	go utils.SendPing()
	strategy.PopuletInstrumentsList()
	r := mux.NewRouter()

	activeUsers := user.New()
	startHandler := start.New(activeUsers)
	sessionHandler := session.New(activeUsers)
	intraDayHandler := intra_day.New(activeUsers)
	stopHandler := stop.New(activeUsers)

	r.HandleFunc("/start", startHandler.Start).Methods(http.MethodPost)
	r.HandleFunc("/stop", stopHandler.Stop).Methods(http.MethodPost)
	r.HandleFunc("/session", sessionHandler.Session).Methods(http.MethodPost)
	r.HandleFunc("/intra-day", intraDayHandler.IntraDay).Methods(http.MethodPost)

	r.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		fmt.Println("Ping Received")
	}).Methods(http.MethodGet)

	r.HandleFunc("/movie", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(Simulation.GetMovies())
	}).Methods(http.MethodGet)

	r.HandleFunc("/candle", func(writer http.ResponseWriter, request *http.Request) {
		params := request.URL.Query()
		clientCode := params.Get("clientCode")
		if clientCode == "" {
			writer.Write([]byte("clientCode is required"))
			writer.WriteHeader(400)
			return
		}
		mutex.Lock()
		userSession, ok := userSessions[clientCode]
		mutex.Unlock()

		if !ok {
			writer.Write([]byte("clientCode not found"))
			writer.WriteHeader(400)
			return
		}

		history := historyData.New(userSession.apiClient)

		data, err := history.GetCandle(smartapi.CandleParams{
			Exchange:    params.Get("exchange"),
			SymbolToken: params.Get("symbolToken"),
			Interval:    params.Get("interval"),
			FromDate:    params.Get("fromDate"),
			ToDate:      params.Get("toDate"),
		})
		if err != nil {

			fmt.Println(err.Error())
		}

		fmt.Println(data)

		b, _ := json.Marshal(data)
		writer.Write(b)
		writer.WriteHeader(200)
	}).Methods(http.MethodGet)

	r.HandleFunc("/swing", func(writer http.ResponseWriter, request *http.Request) {
		body, _ := ioutil.ReadAll(request.Body)
		var param = make(map[string]string)
		json.Unmarshal(body, &param)
		clientCode := param["clientCode"]
		if clientCode == "" {
			writer.Write([]byte("clientCode is required"))
			writer.WriteHeader(400)
			return
		}
		mutex.Lock()
		userSession, ok := userSessions[clientCode]
		mutex.Unlock()
		if !ok {
			writer.Write([]byte("clientCode not found"))
			writer.WriteHeader(400)
			return
		}
		if userSession.session.FeedToken == "" {
			fmt.Println("feed token not set")
			return
		}
		db := Simulation.Connect()
		stockResponses := strategy.SwingScreener(userSession.apiClient, db)
		jsonResponse, err := json.Marshal(stockResponses)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte("Error marshalling response"))
			fmt.Println("Error marshalling response:", err)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK) // Explicitly setting status to 200 OK
		writer.Write(jsonResponse)
	}).Methods(http.MethodPost)

	r.HandleFunc("/backTest", func(writer http.ResponseWriter, request *http.Request) {
		body, _ := ioutil.ReadAll(request.Body)
		var param = make(map[string]string)
		err := json.Unmarshal(body, &param)
		clientCode := param["clientCode"]
		if clientCode == "" {
			writer.Write([]byte("clientCode is required"))
			writer.WriteHeader(400)
			return
		}
		mutex.Lock()
		userSession, ok := userSessions[clientCode]
		mutex.Unlock()
		if !ok {
			writer.Write([]byte("clientCode not found"))
			writer.WriteHeader(400)
			return
		}
		if userSession.session.FeedToken == "" {
			fmt.Println("feed token not set")
			return
		}
		db := Simulation.Connect()
		BackTest.BackTest(userSession.apiClient, db)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte("Error marshalling response"))
			fmt.Println("Error marshalling response:", err)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK) // Explicitly setting status to 200 OK

	}).Methods(http.MethodPost)

	r.HandleFunc("/renew", func(writer http.ResponseWriter, request *http.Request) {
		params := request.URL.Query()
		clientCode := params.Get("clientCode")
		if clientCode == "" {
			writer.Write([]byte("clientCode is required"))
			writer.WriteHeader(400)
			return
		}
		mutex.Lock()
		userSession, ok := userSessions[clientCode]
		mutex.Unlock()

		if !ok {
			writer.Write([]byte("clientCode not found"))
			writer.WriteHeader(400)
			return
		}

		apiClient := userSession.apiClient
		session := userSession.session

		var err error

		//Renew User Tokens using refresh token
		session.UserSessionTokens, err = apiClient.RenewAccessToken(session.RefreshToken)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Println("User Session Tokens :- ", session.UserSessionTokens)
	}).Methods(http.MethodGet)

	r.HandleFunc("/profile", func(writer http.ResponseWriter, request *http.Request) {
		// Get User Profile
		//profile, err := apiClient.GetUserProfile()
		//if err != nil {
		//	fmt.Println(err.Error())
		//	return
		//}
		//
		//fmt.Println("User Profile :- ", session.UserProfile)
		//fmt.Println("User Session Object :- ", session)

	})

	r.HandleFunc("/order", func(writer http.ResponseWriter, request *http.Request) {
		//Place Order
		//order, err := apiClient.PlaceOrder(smartapi.OrderParams{
		//	Variety:         "NORMAL",
		//	TradingSymbol:   "SBIN-EQ",
		//	SymbolToken:     "3045",
		//	TransactionType: "BUY",
		//	Exchange:        "NSE",
		//	OrderType:       "LIMIT",
		//	ProductType:     "INTRADAY",
		//	Duration:        "DAY",
		//	Price:           "19500",
		//	SquareOff:       "0",
		//	StopLoss:        "0",
		//	Quantity:        "1",
		//})
		//
		//if err != nil {
		//	fmt.Println(err.Error())
		//	return
		//}
		//
		//fmt.Println("Placed Order ID and Script :- ", order)
	})

	r.HandleFunc("/option", func(writer http.ResponseWriter, request *http.Request) {
		body, _ := ioutil.ReadAll(request.Body)
		var param = make(map[string]string)
		json.Unmarshal(body, &param)

		clientCode := param["clientCode"]
		if clientCode == "" {
			writer.Write([]byte("clientCode is required"))
			writer.WriteHeader(400)
			return
		}

		mutex.Lock()
		userSession, ok := userSessions[clientCode]
		mutex.Unlock()

		if !ok {
			writer.Write([]byte("clientCode not found"))
			writer.WriteHeader(400)
			return
		}

		if userSession.session.FeedToken == "" {
			fmt.Println("feed token not set")
			return
		}

		ltp := smartStream.New(clientCode, userSession.session.FeedToken)
		strategy := strategy.New()

		strategy.Algo(ltp, param["expiry"], param["index"], userSession.apiClient)

	}).Methods(http.MethodPost)

	port := os.Getenv("HTTP_PLATFORM_PORT")

	// default back to 8080 for local dev
	if port == "" {
		port = "8000"
	}

	http.ListenAndServe(":"+port, r)
}
