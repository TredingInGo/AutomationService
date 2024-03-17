package main

import (
	"encoding/json"
	"fmt"
	"github.com/TredingInGo/AutomationService/Simulation"
	"github.com/TredingInGo/AutomationService/historyData"
	"github.com/TredingInGo/AutomationService/strategy"
	smartapi "github.com/TredingInGo/smartapi"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	accessToken, feedToken, refreshToken string
	apiClient                            *smartapi.Client
	session                              smartapi.UserSession
	err                                  error
	userSessions                         = make(map[string]*clientSession)
)

type clientSession struct {
	apiClient *smartapi.Client
	session   smartapi.UserSession
}

func init() {
	accessToken = os.Getenv("ACCESS_TOKEN")
	feedToken = os.Getenv("FEED_TOKEN")
	refreshToken = os.Getenv("REFRESH_TOKEN")
}
func sendPing() {
	for {
		time.Sleep(120 * time.Second)
		url := "https://tredingingo.onrender.com/ping"

		resp, err := http.Get(url)
		if err != nil {
			log.Fatalf("Error occurred while calling the API: %s", err.Error())
		}
		defer resp.Body.Close() // Make sure to close the response body at the end

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error occurred while reading the response body: %s", err.Error())
		}
		fmt.Println("API Response:", string(body))

	}

}
func main() {
	fmt.Println("Starting the server, time: ", time.Now())
	mutex := sync.Mutex{}

	defer func() {
		recover()
	}()
	go sendPing()
	r := mux.NewRouter()

	r.HandleFunc("/session", func(writer http.ResponseWriter, request *http.Request) {

		m := map[string]string{}
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, "Error reading request body", http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(body, &m)
		if err != nil {
			http.Error(writer, "Error parsing JSON request body", http.StatusBadRequest)
			return
		}
		apiClient := smartapi.New(m["clientCode"], m["password"], m["marketKey"])
		session, err := apiClient.GenerateSession(m["totp"])
		if err != nil {
			errorMessage := fmt.Sprintf("Error generating session: %s", err.Error())
			http.Error(writer, errorMessage, http.StatusInternalServerError)
			return
		}

		mutex.Lock()
		userSessions[m["clientCode"]] = &clientSession{
			apiClient: apiClient,
			session:   session,
		}
		mutex.Unlock()

		setEnv(session)

		successMessage := fmt.Sprintf("User Session Tokens: %v", session.UserSessionTokens)
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(map[string]string{"message": "Connected successfully with angel one", "sessionTokens": successMessage})
	}).Methods(http.MethodPost)

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

	r.HandleFunc("/intra-day", func(writer http.ResponseWriter, request *http.Request) {
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
		strategy.TrendFollowingStretgy(userSession.apiClient, db)
	}).Methods(http.MethodPost)

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
		// Simulation.CollectData(db, apiClient) //this will populate list of stocks.
		strategy.SwingScreener(userSession.apiClient, db)

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
		session.UserProfile, err = apiClient.GetUserProfile()
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Println("User Profile :- ", session.UserProfile)
		fmt.Println("User Session Object :- ", session)

	})

	r.HandleFunc("/order", func(writer http.ResponseWriter, request *http.Request) {
		//Place Order
		order, err := apiClient.PlaceOrder(smartapi.OrderParams{
			Variety:         "NORMAL",
			TradingSymbol:   "SBIN-EQ",
			SymbolToken:     "3045",
			TransactionType: "BUY",
			Exchange:        "NSE",
			OrderType:       "LIMIT",
			ProductType:     "INTRADAY",
			Duration:        "DAY",
			Price:           "19500",
			SquareOff:       "0",
			StopLoss:        "0",
			Quantity:        "1",
		})

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Println("Placed Order ID and Script :- ", order)
	})

	port := os.Getenv("HTTP_PLATFORM_PORT")

	// default back to 8080 for local dev
	if port == "" {
		port = "8000"
	}

	http.ListenAndServe(":"+port, r)
}

func setEnv(session smartapi.UserSession) {
	os.Setenv("ACCESS_TOKEN", session.AccessToken)
	os.Setenv("FEED_TOKEN", session.FeedToken)
	os.Setenv("REFRESH_TOKEN", session.RefreshToken)

	feedToken = session.FeedToken
	accessToken = session.AccessToken
	refreshToken = session.RefreshToken
}
