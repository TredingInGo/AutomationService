package websocket

import (
	"encoding/json"
	"github.com/TredingInGo/AutomationService/user"
	"log"
)

func Test_WebSocket(user user.Users) {
	activeUser, exist := user.Get("pritivardhan.856@gmail.com")
	if !exist {
		log.Println("User Not Found")
	}
	clientID := activeUser.AliceBlueClientID
	bearerToken := activeUser.AliceBlueSessionCode

	// Load tokens
	//if err := token.LoadAllTokens(); err != nil {
	//	log.Fatal("❌ Failed to load tokens:", err)
	//}

	//niftyToken, err := token.GetOptionToken("NIFTY", "NSE", "IDX")

	//if err != true {
	//	log.Fatal("❌ Could not find NIFTY token:", err)
	//}

	fullToken := "NSE|26000#NSE|26009"
	_, err := GetWebSocketSessionID(bearerToken, clientID)
	if err != nil {
		log.Println(err)
		return
	}
	conn, err := ConnectMarketWebSocket(clientID, bearerToken)
	if err != nil {
		log.Println(err)
		return
		//log.Fatal("❌ Connection error:", err)
	}

	defer conn.Close()

	err = SubscribeMarketTokens(conn, fullToken)
	if err != nil {
		log.Fatal("❌ Subscription failed:", err)
	}
	log.Println("✅ Subscribed to", fullToken)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msg, &data); err != nil {
			log.Println("Unmarshal error:", string(msg))
			continue
		}

		log.Println("tick: ", data)

	}
}
