package websocket

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const marketWSURL = "wss://ws1.aliceblueonline.com/NorenWS/"

type MarketWSRequest struct {
	SUserToken string `json:"susertoken"`
	T          string `json:"t"` // always "c" for connection
	ActID      string `json:"actid"`
	UID        string `json:"uid"`
	Source     string `json:"source"` // always "API"
}

func hashSHA256(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

const wsSessionURL = "https://ant.aliceblueonline.com/rest/AliceBlueAPIService/api/ws/createWsSession"

type WSSessionResponse struct {
	Stat   string `json:"stat"`
	Result struct {
		WsSess string `json:"wsSess"`
	} `json:"result"`
	Message string `json:"message"`
}

// GetWebSocketSessionID calls the REST API to fetch a wsSess token
func GetWebSocketSessionID(sessionID, ClientId string) (string, error) {
	url := "https://ant.aliceblueonline.com/rest/AliceBlueAPIService/api/ws/createWsSession"
	payload := map[string]string{
		"loginType": "API",
	}
	body, _ := json.Marshal(payload)

	// Headers — this is the KEY part
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ClientId+" "+sessionID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// If the method is wrong, this will return 405
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return "", fmt.Errorf("received 405 Method Not Allowed: make sure you're using POST with correct headers")
	}

	// Read and parse the response
	respBody, _ := io.ReadAll(resp.Body)
	var wsResp WSSessionResponse
	if err := json.Unmarshal(respBody, &wsResp); err != nil {
		return "", fmt.Errorf("json decode error: %v", err)
	}

	if wsResp.Stat != "Ok" {
		return "", fmt.Errorf("failed to create ws session: %v", wsResp.Message)
	}
	log.Println("wsSession created:", wsResp.Result.WsSess)
	return wsResp.Result.WsSess, nil
}

// ConnectMarketWebSocket uses wsSess and connects to real-time feed
func ConnectMarketWebSocket(clientID, sessionId string) (*websocket.Conn, error) {
	// Step 1: Double SHA256 hash the session ID
	log.Println("clientId: ", clientID, " SessionId: ", sessionId)
	hashed := hashSHA256(hashSHA256(sessionId))
	log.Println("Hashed Valie ", hashed)
	// Step 2: Connect to the WebSocket
	header := http.Header{}
	conn, resp, err := websocket.DefaultDialer.Dial(marketWSURL, header)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
		}
		return nil, fmt.Errorf("❌ failed to dial websocket: %v", err)
	}

	// Step 3: Build auth payload as a map and convert to JSON
	authPayload := map[string]string{
		"susertoken": hashed,
		"t":          "c",
		"actid":      clientID + "_API",
		"uid":        clientID + "_API",
		"source":     "API",
	}

	payloadBytes, err := json.Marshal(authPayload)
	log.Println("Payload: ", authPayload)
	if err != nil {
		return nil, fmt.Errorf("❌ error marshaling payload: %v", err)
	}

	// Step 4: Send raw JSON as string
	if err := conn.WriteMessage(websocket.TextMessage, payloadBytes); err != nil {
		return nil, fmt.Errorf("❌ failed to send login payload: %v", err)
	}

	log.Println("✅ Sent login payload. Awaiting server response...")

	// Step 5: Read the response
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("❌ read message failed: %v", err)
	}

	log.Printf("🔁 WS Response: %s\n", string(msg))
	//GetWebSocketSessionID(sessionId, clientID)
	// Optional: parse and check response status
	var res map[string]interface{}
	if err := json.Unmarshal(msg, &res); err == nil {
		if res["t"] == "ck" && res["s"] == "NOT_OK" {
			return nil, fmt.Errorf("❌ WebSocket handshake rejected: %v", res)
		}
	}
	StartHeartbeat(conn, clientID)
	log.Println("✅ WebSocket authenticated successfully")
	return conn, nil
}

func SubscribeMarketTokens(conn *websocket.Conn, tokens string) error {
	msg := map[string]string{"k": tokens, "t": "d"}
	return conn.WriteJSON(msg)
}
func StartHeartbeat(ws *websocket.Conn, userID string) func() {
	stopChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(55 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				payload := map[string]string{
					"heartbeat": "h",
					"userId":    userID,
				}
				heartbeatJSON, err := json.Marshal(payload)
				if err != nil {
					log.Println("❌ Failed to marshal heartbeat:", err)
					continue
				}
				if err := ws.WriteMessage(websocket.TextMessage, heartbeatJSON); err != nil {
					log.Println("❌ Failed to send heartbeat:", err)
					return
				}
				log.Println("❤️ Heartbeat sent")

			case <-stopChan:
				log.Println("🛑 Stopping heartbeat and closing WebSocket connection")
				ws.Close()
				return
			}
		}
	}()

	// Return a function to stop heartbeat and close connection
	return func() {
		stopChan <- struct{}{}
	}
}
