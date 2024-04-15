package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func SendPing() {
	if env := os.Getenv("ENV"); env != "production" {
		return
	}

	ticker := time.NewTicker(2 * time.Minute)

	for range ticker.C {
		url := "https://tredingingo.onrender.com/ping"

		resp, err := http.Get(url)
		if err != nil {
			log.Fatalf("Error occurred while calling the API: %s", err.Error())
		}
		defer resp.Body.Close() // Make sure to close the response body at the end

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error occurred while reading the response body: %s", err.Error())
		}
		fmt.Println("API Response:", string(body))
	}
}
