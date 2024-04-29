package smartStream

import (
	"github.com/TredingInGo/smartapi/models"
	"github.com/TredingInGo/smartapi/smartstream"
	"time"
)

var (
	currTime = time.Now()
	baseTime = time.Date(currTime.Year(), currTime.Month(), currTime.Day(), 9, 0, 0, 0, time.Local)
)

type SmartStream interface {
	Connect(ch chan *models.SnapQuote, mode models.SmartStreamSubsMode, tokenInfo []models.TokenInfo)
	STOP()
}

type ltp struct {
	client *smartstream.WebSocket
}

func New(clientCode, feedToken string) SmartStream {
	return ltp{client: smartstream.New(clientCode, feedToken)}
}

func (l ltp) Connect(ch chan *models.SnapQuote, mode models.SmartStreamSubsMode, tokenInfo []models.TokenInfo) {
	l.client.SetOnConnected(onConnected(l.client, mode, tokenInfo))
	l.client.SetOnSnapquote(onLTP(ch))
	l.client.Connect()
}

func (l ltp) STOP() {
	l.client.Stop()
}
