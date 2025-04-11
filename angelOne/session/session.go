package session

import (
	"context"
	"fmt"
	"github.com/TredingInGo/AutomationService/totp"
	"github.com/TredingInGo/AutomationService/user"
	smartapi "github.com/TredingInGo/smartapi"
	"log"
)

type Session struct {
	activeUsers user.Users
}

func NewAngelOneSession(users user.Users) Session {
	return Session{activeUsers: users}
}

func (s *Session) AngelOneSession(response user.LoginResponse, userId string) error {

	apiClient := smartapi.New(response.AngelOneClientCode, response.AngelOnePassword, response.AngelOneMarketKey)
	session, err := apiClient.GenerateSession(totp.GetTOPT(response.AngelOneClientCode))
	if err != nil {
		errorMessage := fmt.Sprintf("Error generating session: %s", err.Error())
		log.Printf("Error: %v\n", errorMessage)
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	user := &user.UserInfo{
		ApiClient:  apiClient,
		Session:    session,
		Ctx:        ctx,
		CancelFunc: cancel,
	}

	success := s.activeUsers.Set(userId, user)
	if !success {
		log.Println("Error while saving client info")
		return err
	}
	return nil

}
