package stop

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TredingInGo/AutomationService/user"
)

type Handler struct {
	activeUsers user.Users
}

func New(users user.Users) Handler {
	h := Handler{
		activeUsers: users,
	}

	go h.stopper()

	return h
}

func (h *Handler) stopper() {
	ticker := time.NewTicker(1 * time.Minute)
	isTestMode := os.Getenv("TEST_MODE") == "true"

	for t := range ticker.C {
		if t.Hour() == 3 && t.Minute() == 16 || isTestMode {
			// in test_mode, call after 15 minutes of auto start time
			if isTestMode {
				// creating a timer to wait for 15 minutes
				<-time.NewTimer(15 * time.Minute).C
			}

			h.activeUsers.RemoveAll()
		}
	}
}

func (h *Handler) Stop(writer http.ResponseWriter, request *http.Request) {
	mp := map[string]string{}
	body, _ := io.ReadAll(request.Body)

	err := json.Unmarshal(body, &mp)
	if err != nil {
		log.Println(err)
	}

	clientID := mp["clientCode"]
	if clientID == "" {
		writer.Write([]byte(`clientID missing`))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	userInfo, exists := h.activeUsers.Get(clientID)
	if !exists {
		writer.Write([]byte(`clientID not found`))
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	// call cancel context
	userInfo.CancelFunc()

	// remove from active user
	h.activeUsers.Remove(clientID)

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(fmt.Sprintf(`cancel context called for clientID: %v\n`, clientID)))
}
