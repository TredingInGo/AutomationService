package start

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/TredingInGo/AutomationService/user"
)

type cred struct {
	clientCode, password, marketKey string
}

type Handler struct {
	activeUsers user.Users
	list        map[string]*cred
	mu          sync.Mutex
}

func New(users user.Users) Handler {
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
	isTestMode := os.Getenv("TEST_MODE") == "true"

	for t := range ticker.C {
		if t.Hour() == 10 && t.Minute() == 0 || isTestMode {
			startTime := time.Now().Format("2006-01-02 15:04:05")
			log.Println("Running starter at: ", startTime)

			for _, creds := range h.list {
				// check if intra-day is already running
				user, exists := h.activeUsers.Get(creds.clientCode)
				if exists && user.IsIntraDayRunning {
					log.Println("Intra-day already running for user: ", creds.clientCode)

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
					log.Println("Error while creating session automatically for clientID ", creds.clientCode,
						" error ", err)
					continue
				}

				log.Println("response from auto session api ", string(resp))

				// start intra-day
				data = map[string]interface{}{
					"clientCode": creds.clientCode,
				}

				resp, err = post(host, "/intra-day", data)
				if err != nil {
					log.Println("Error while starting equity intra-day automatically for clientID ", creds.clientCode,
						" error ", err)
					continue
				}
				log.Println("response from auto option api ", string(resp))

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
