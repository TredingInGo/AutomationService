package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	BaseURL = "https://ant.aliceblueonline.com/rest/AliceBlueAPIService/api" // Replace with actual base URL
)

type EncryptionKeyResponse struct {
	UserID string `json:"userId"`
	EncKey string `json:"encKey"`
	Stat   string `json:"stat"`
	EMsg   string `json:"emsg"`
}

type SessionIDResponse struct {
	Stat      string `json:"stat"`
	SessionID string `json:"sessionID"`
	EMsg      string `json:"emsg"`
}

func AuthenticateAndGetSession(userID, apiKey string) (string, string, error) {
	encKey, err := GetEncryptionKey(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get encryption key: %v", err)
	}

	// Step 2: Generate SHA-256 Hash
	hashed := GenerateSHA256(userID, apiKey, encKey)

	// Step 3: Get Session ID
	sessionID, err := GetSessionID(userID, hashed)
	if err != nil {
		return "", "", fmt.Errorf("failed to get session ID: %v", err)
	}

	return sessionID, hashed, nil
}

func GetEncryptionKey(userID string) (string, error) {
	url := BaseURL + "/customer/getAPIEncpkey"
	payload := map[string]string{"userId": userID}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call encKey API: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result EncryptionKeyResponse
	json.Unmarshal(body, &result)

	if result.EncKey != "" {
		return result.EncKey, nil
	}

	return "", fmt.Errorf("encKey error: %s", result.EMsg)
}

// GenerateSHA256 returns SHA256 hash of userID + apiKey + encKey
func GenerateSHA256(userID, apiKey, encKey string) string {
	data := userID + apiKey + encKey
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GetSessionID generates the session ID using hashed userData
func GetSessionID(userID, hashedData string) (string, error) {
	url := BaseURL + "/customer/getUserSID"
	payload := map[string]string{
		"userId":   userID,
		"userData": hashedData,
	}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call session ID API: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result SessionIDResponse
	json.Unmarshal(body, &result)

	if result.Stat == "Ok" {
		return result.SessionID, nil
	}

	return "", fmt.Errorf("sessionID error: %s", result.EMsg)
}
