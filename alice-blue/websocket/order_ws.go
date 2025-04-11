package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

const orderWSSessionURL = "https://ant.aliceblueonline.com/order-notify/ws/createWsToken"
const orderWSURL = "wss://ant.aliceblueonline.com/order-notify/websocket"

type OrderTokenResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  []struct {
		OrderToken string `json:"orderToken"`
	} `json:"result"`
}

type OrderWSAuthPayload struct {
	OrderToken string `json:"orderToken"`
	UserID     string `json:"userId"`
}

func FetchOrderToken(bearerToken string) (string, error) {
	req, err := http.NewRequest("GET", orderWSSessionURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result OrderTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if len(result.Result) == 0 {
		return "", fmt.Errorf("no order token found")
	}

	return result.Result[0].OrderToken, nil
}

func ConnectOrderWebSocket(orderToken, userID string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(orderWSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect order ws: %w", err)
	}

	payload := OrderWSAuthPayload{
		OrderToken: orderToken,
		UserID:     userID,
	}
	err = conn.WriteJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("auth payload failed: %w", err)
	}

	log.Println("✅ Connected to order status WebSocket")
	return conn, nil
}

func SendHeartbeat(conn *websocket.Conn, userID string) error {
	return conn.WriteJSON(map[string]string{
		"heartbeat": "h",
		"userId":    userID,
	})
}
