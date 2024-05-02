package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/TredingInGo/AutomationService/totp"
	"github.com/TredingInGo/AutomationService/user"
	smartapi "github.com/TredingInGo/smartapi"
)

type Session struct {
	activeUsers user.Users
}

func New(users user.Users) Session {
	return Session{activeUsers: users}
}

func (s *Session) Session(writer http.ResponseWriter, request *http.Request) {

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

	clientID := m["clientCode"]
	if clientID == "" {
		http.Error(writer, "clientCode is missing", http.StatusBadRequest)
		return
	}

	// check if session already present
	userInfo, exists := s.activeUsers.Get(clientID)
	if exists {
		writer.Write([]byte(fmt.Sprintf(
			`Session Already Present
					Tokens: %v
			`, userInfo.Session)))

		return
	}

	apiClient := smartapi.New(m["clientCode"], m["password"], m["marketKey"])
	session, err := apiClient.GenerateSession(totp.GetTOPT(m["clientCode"]))
	if err != nil {
		errorMessage := fmt.Sprintf("Error generating session: %s", err.Error())
		http.Error(writer, errorMessage, http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	user := &user.UserInfo{
		ApiClient:  apiClient,
		Session:    session,
		Ctx:        ctx,
		CancelFunc: cancel,
	}

	success := s.activeUsers.Set(clientID, user)
	if !success {
		http.Error(writer, "Error while saving session for clientID"+clientID, http.StatusInternalServerError)
		return
	}

	successMessage := fmt.Sprintf("User Session Tokens: %v", session.UserSessionTokens)
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(map[string]string{"message": "Trading Session Connected successfully with angel one broker", "sessionTokens->": successMessage})
}
