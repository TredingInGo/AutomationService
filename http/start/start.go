package start

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/TredingInGo/AutomationService/users"
)

type cred struct {
	clientCode, password, marketKey string
}

type Handler struct {
	activeUsers users.ActiveUsers
	list        map[string]*cred
	mu          sync.Mutex
}

func New(users users.ActiveUsers) Handler {
	h := Handler{
		activeUsers: users,
		list:        make(map[string]*cred),
		mu:          sync.Mutex{},
	}

	go h.starter()

	return h
}

func (h *Handler) starter() {
	ticker := time.NewTicker(1 * time.Minute)

	count := 0
	for t := range ticker.C {
		if t.Hour() == 10 && t.Minute() == 0 {
			count++
			for _, creds := range h.list {
				// check if intra-day is already running
				user, exists := h.activeUsers.Get(creds.clientCode)
				if exists && user.IsIntraDayRunning {
					fmt.Println("Intra-day already running for user: ", creds.clientCode)

					continue
				}

				host := os.Getenv("service_host")
				if host == "" {
					host = "https://tredingingo.onrender.com"
				}

				// create a session
				data := map[string]interface{}{
					"clientCode": creds.clientCode,
					"password":   creds.password,
					"marketKey":  creds.marketKey,
				}

				resp, err := post(host, "/session", data)
				if err != nil {
					fmt.Println("Error while creating session automatically for clientID ", creds.clientCode,
						" error ", err)
					continue
				}

				fmt.Println("response from auto session api ", string(resp))

				// start intra-day
				data = map[string]interface{}{
					"clientCode": creds.clientCode,
				}

				resp, err = post(host, "/intra-day", data)
				if err != nil {
					fmt.Println("Error while starting intra-day automatically for clientID ", creds.clientCode,
						" error ", err)
					continue
				}

				fmt.Println("response from auto intra-day api ", string(resp))
			}
		}
	}
}

func (h *Handler) Start(writer http.ResponseWriter, request *http.Request) {
	mp := map[string]string{}
	body, _ := io.ReadAll(request.Body)

	json.Unmarshal(body, &mp)

	clientID := mp["clientCode"]
	if clientID == "" {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`clientCode missing`))
		return
	}

	marketKey := mp["marketKey"]
	if marketKey == "" {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`marketKey missing`))
		return
	}

	password := mp["password"]
	if password == "" {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`password missing`))
		return
	}

	h.mu.Lock()
	h.list[clientID] = &cred{
		clientCode: clientID,
		password:   password,
		marketKey:  marketKey,
	}

	h.mu.Unlock()

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(`added to the auto starter list`))
}

func post(host, endpoint string, data map[string]interface{}) ([]byte, error) {
	reqBody, _ := json.Marshal(data)

	resp, err := http.Post(host+endpoint, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)

	return body, nil
}
