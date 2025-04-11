package handler

import (
	"database/sql"
	"encoding/json"
	"github.com/TredingInGo/AutomationService/alice-blue/auth"
	"github.com/TredingInGo/AutomationService/alice-blue/websocket"
	"github.com/TredingInGo/AutomationService/user"
	"net/http"
)

type LoginResponse struct {
	Name                 string `json:"name"`
	AliceBlueClientID    string `json:"aliceBlueClientId"`
	AliceBlueSessionCode string `json:"aliceBlueSessionCode"`
	AngelOneClientCode   string `json:"angelOneClientCode"`
	AliceBlueBearerToken string `json:"alice_blue_bearer_token"`
}
type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds user.Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	authSvc := user.NewService(h.db)
	res, err := authSvc.LoginUser(creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	userInfo := user.New()

	//session := session.NewAngelOneSession(userInfo)
	//err = session.AngelOneSession(*res, creds.UserId)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusUnauthorized)
	//	return
	//}
	userdetails, exist := userInfo.Get(creds.UserId)
	if !exist {
		userdetails = &user.UserInfo{}
	}
	userdetails.AliceBlueClientID = res.AliceClientId
	sessionID, bearerToken, err := auth.AuthenticateAndGetSession(res.AliceClientId, res.AliceApiKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	userdetails.AliceBlueSessionCode = sessionID
	userdetails.Name = res.Name
	userdetails.UserId = creds.UserId
	userdetails.AliceBlueBearerToken = bearerToken

	success := userInfo.Set(creds.UserId, userdetails)
	if !success {
		http.Error(w, "Error while saving session for clientID: "+creds.UserId, http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Name:                 userdetails.Name,
		AliceBlueClientID:    userdetails.AliceBlueClientID,
		AliceBlueSessionCode: userdetails.AliceBlueSessionCode,
		AngelOneClientCode:   userdetails.Session.AccessToken,
		AliceBlueBearerToken: bearerToken,
	}
	websocket.Test_WebSocket(userInfo)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
