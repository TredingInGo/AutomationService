package smartStream

import (
	"time"

	"github.com/TredingInGo/smartapi/models"
	"github.com/TredingInGo/smartapi/smartstream"
)

var (
	currTime = time.Now()
	baseTime = time.Date(currTime.Year(), currTime.Month(), currTime.Day(), 9, 0, 0, 0, time.Local)
)

type ltp struct {
	client *smartstream.WebSocket
}

func New(clientCode, feedToken string) ltp {
	return ltp{client: smartstream.New(clientCode, feedToken)}
}

func (l ltp) Connect(ch chan *models.SnapQuote, mode models.SmartStreamSubsMode, exchangeType models.ExchangeType, token string) {
	l.client.SetOnConnected(onConnected(l.client, mode, exchangeType, token))
	l.client.SetOnSnapquote(onLTP(ch))
	l.client.Connect()
}
