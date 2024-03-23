package clients

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TredingInGo/AutomationService/utils"
)

// HTTPClient represents an HTTP client.
type HTTPClient interface {
	Do(method, rURL string, params map[string]interface{}, headers http.Header) (HttpResponse, error)
	DoEnvelope(method, url string, params map[string]interface{}, headers http.Header) ([]byte, error)
	GetClient() *httpClient
}

type httpClient struct {
	client *http.Client
	hLog   *log.Logger
	debug  bool
}

func NewHTTPClient(h *http.Client, hLog *log.Logger, debug bool) HTTPClient {
	if hLog == nil {
		hLog = log.New(os.Stdout, "base.HTTP: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	if h == nil {
		h = &http.Client{
			Timeout: time.Duration(5) * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   10,
				ResponseHeaderTimeout: time.Second * time.Duration(5),
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	return &httpClient{
		hLog:   hLog,
		client: h,
		debug:  debug,
	}
}

type HttpResponse struct {
	Body       []byte
	StatusCode int
	headers    http.Header
}

type serviceResponse struct {
	Status    bool            `json:"status"`
	ErrorCode string          `json:"errorcode"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
}

// Do executes an HTTP request and returns the response.
func (h *httpClient) Do(method, rURL string, params map[string]interface{}, headers http.Header) (HttpResponse, error) {
	var (
		resp       = HttpResponse{}
		postParams io.Reader
		err        error
	)

	if method == http.MethodPost && params != nil {
		jsonParams, err := json.Marshal(params)

		if err != nil {
			return resp, err
		}

		postParams = bytes.NewBuffer(jsonParams)
	}

	req, err := http.NewRequest(method, rURL, postParams)

	if err != nil {
		h.hLog.Printf("Request preparation failed: %v", err)
		return resp, err
	}

	if headers != nil {
		req.Header = headers
	}

	// If a content-type isn't set, set the default one.
	if req.Header.Get("Content-Type") == "" {
		if method == http.MethodPost || method == http.MethodPut {
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	// If the request method is GET or DELETE, add the params as QueryString.
	//if method == http.MethodGet || method == http.MethodDelete {
	//	req.URL.RawQuery = params.Encode()
	//}

	r, err := h.client.Do(req)
	if err != nil {
		h.hLog.Printf("Request failed: %v", err)
		return resp, err
	}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.hLog.Printf("Unable to read response: %v", err)
		return resp, err
	}

	resp.StatusCode = r.StatusCode
	resp.Body = body
	resp.headers = r.Header

	if h.debug {
		h.hLog.Printf("%s %s -- %d %v", method, req.URL.RequestURI(), resp.StatusCode, req.Header)
	}

	return resp, nil
}

// DoEnvelope makes an HTTP request and parses the JSON response (fastglue envelop structure)
func (h *httpClient) DoEnvelope(method, url string, params map[string]interface{}, headers http.Header) ([]byte, error) {
	if params == nil {
		params = map[string]interface{}{}
	}

	// Send custom headers set
	if headers == nil {
		headers = map[string][]string{}
	}

	localIp, publicIp, mac, err := utils.GetIpAndMac()

	if err != nil {
		return nil, err
	}

	// Add Kite Connect version to header
	headers.Add("Content-Type", "application/json")
	headers.Add("X-ClientLocalIP", localIp)
	headers.Add("X-ClientPublicIP", publicIp)
	headers.Add("X-MACAddress", mac)
	headers.Add("Accept", "application/json")
	headers.Add("X-UserType", "USER")
	headers.Add("X-SourceID", "WEB")

	resp, err := h.Do(method, url, params, headers)
	if err != nil {
		return nil, err
	}

	// Successful request, but error envelope.
	if resp.StatusCode >= http.StatusBadRequest {
		var e serviceResponse
		if err := json.Unmarshal(resp.Body, &e); err != nil {
			//h.hLog.Printf("Error parsing JSON response: %v", err)
			return nil, err
		}

		return nil, NewError(e.ErrorCode, e.Message, e.Data)
	}

	// We now unmarshal the body.
	sr := serviceResponse{}

	if err := json.Unmarshal(resp.Body, &sr); err != nil {
		//h.hLog.Printf("Error parsing JSON response: %v | %s", err, resp.Body)
		return nil, err
	}

	if !sr.Status {
		return nil, NewError(sr.ErrorCode, sr.Message, sr.Data)
	}

	return sr.Data, nil
}

// GetClient return's the underlying net/http client.
func (h *httpClient) GetClient() *httpClient {
	return h
}
