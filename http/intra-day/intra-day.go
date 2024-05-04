package intra_day

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/TredingInGo/AutomationService/Simulation"
	"github.com/TredingInGo/AutomationService/strategy"
	"github.com/TredingInGo/AutomationService/user"
)

type IntraDay struct {
	activeUsers user.Users
}

func New(users user.Users) IntraDay {
	return IntraDay{activeUsers: users}
}

func (i *IntraDay) IntraDay(writer http.ResponseWriter, request *http.Request) {
	body, _ := ioutil.ReadAll(request.Body)
	var param = make(map[string]string)
	json.Unmarshal(body, &param)

	clientCode := param["clientCode"]
	if clientCode == "" {
		writer.WriteHeader(400)
		writer.Write([]byte("clientCode is required"))
		return
	}

	userInfo, exists := i.activeUsers.Get(clientCode)
	if !exists {
		writer.WriteHeader(400)
		writer.Write([]byte("clientCode not found"))
		return
	}

	if userInfo.Session.FeedToken == "" {
		log.Println("feed token not set")
		return
	}

	db := Simulation.Connect()
	go strategy.TrendFollowingStretgy(userInfo.Ctx, userInfo.ApiClient, db)

	userInfo.IsIntraDayRunning = true
	i.activeUsers.Update(clientCode, userInfo)

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(`started for clientID: ` + clientCode))
}
