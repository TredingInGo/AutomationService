package stop

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	for time := range ticker.C {
		if time.Hour() == 3 && time.Minute() == 16 {
			h.activeUsers.RemoveAll()
		}
	}
}

func (h *Handler) Stop(writer http.ResponseWriter, request *http.Request) {
	mp := map[string]string{}
	body, _ := io.ReadAll(request.Body)

	err := json.Unmarshal(body, &mp)
	if err != nil {
		fmt.Println(err)
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
