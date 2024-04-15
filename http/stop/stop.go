package stop

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/TredingInGo/AutomationService/users"
)

type Handler struct {
	activeUsers users.ActiveUsers
}

func New(users users.ActiveUsers) Handler {
	return Handler{
		activeUsers: users,
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

	// remove from active users
	h.activeUsers.Remove(clientID)

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(fmt.Sprintf(`cancel context called for clientID: %v\n`, clientID)))
}
