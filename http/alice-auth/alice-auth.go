package alice_auth

import (
	"encoding/json"
	"github.com/TredingInGo/AutomationService/alice-blue/auth"
	"net/http"
)

type SessionRequest struct {
	UserID string `json:"userId"`
	APIKey string `json:"apiKey"`
}

type SessionResponse struct {
	SessionID   string `json:"sessionId,omitempty"`
	BearerToken string `json:"bearerToken"`
	Error       string `json:"error,omitempty"`
}

// AliceBlueSessionHandler handles POST /session
func AliceBlueSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req SessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request format"}`, http.StatusBadRequest)
		return
	}

	sessionID, bearerToken, err := auth.AuthenticateAndGetSession(req.UserID, req.APIKey)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SessionResponse{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(SessionResponse{SessionID: sessionID, BearerToken: bearerToken})
}
