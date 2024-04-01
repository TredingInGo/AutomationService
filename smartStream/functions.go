package smartStream

import (
	"fmt"
	"github.com/TredingInGo/smartapi/smartstream"
	"log"
	"time"

	"github.com/TredingInGo/AutomationService/clients"
	"github.com/TredingInGo/smartapi/models"
)

func MakeCandle(ch <-chan *models.LTPInfo, duration int) {
	//candleDuration := 5
	candles := make([]*clients.CandleResponse, 0)
	//t, err := time.Parse("15:04:05", baseTime)
	//if err != nil {
	//	fmt.Println("Error parsing time:", err)
	//	return
	//}

	//formattedBaseTime := t.Format("15:04:05")
	lastSegStart := time.Time{}

	for data := range ch {
		epochSeconds := int64(data.ExchangeFeedTimeEpochMillis) / 1000
		dataTimeFormatted := time.Unix(epochSeconds, 0)
		if len(candles) == 0 {
			candles = append(candles, &clients.CandleResponse{
				Timestamp: dataTimeFormatted,
				Open:      float64(data.LastTradedPrice) / 100,
				High:      float64(data.LastTradedPrice) / 100,
				Low:       float64(data.LastTradedPrice) / 100,
				Close:     float64(data.LastTradedPrice) / 100,
				Volume:    0,
			})
			tempTime := dataTimeFormatted.Sub(baseTime)
			fmt.Println("temp time", tempTime)
			tempTimeInSec := tempTime.Seconds()
			thresHoldTime := (int(tempTimeInSec)) / (duration)
			thresHoldTime++
			lastSegStart = baseTime.Add(time.Duration(thresHoldTime*duration) * time.Second)

		} else {
			if lastSegStart.After(dataTimeFormatted) {
				lastData := candles[len(candles)-1]
				ltp := float64(data.LastTradedPrice) / 100
				if lastData.Low > ltp {
					lastData.Low = ltp
				}

				if lastData.High < ltp {
					lastData.High = ltp
				}

				lastData.Close = ltp

				candles[len(candles)-1] = lastData
			} else {
				fmt.Println(candles[len(candles)-1])
				candles = append(candles, &clients.CandleResponse{
					Timestamp: dataTimeFormatted,
					Open:      float64(data.LastTradedPrice) / 100,
					High:      float64(data.LastTradedPrice) / 100,
					Low:       float64(data.LastTradedPrice) / 100,
					Close:     float64(data.LastTradedPrice) / 100,
					Volume:    0,
				})

				lastSegStart = lastSegStart.Add(time.Duration(duration) * time.Second)
			}
		}
	}

	for _, data := range candles {
		fmt.Println(data)
	}
}

func onConnected(client *smartstream.WebSocket, mode models.SmartStreamSubsMode, tokenInfo []models.TokenInfo) func() {
	return func() {
		log.Printf("connected")
		err := client.Subscribe(mode, tokenInfo)
		if err != nil {
			log.Printf("error while subscribing")
		}
	}
}

func onSnapquote(snapquote models.SnapQuote) {
	log.Printf("%d", snapquote.BestFiveSell[0])
}

func onLTP(chForCandle chan *models.SnapQuote) func(ltpInfo models.SnapQuote) {
	return func(ltpInfo models.SnapQuote) {
		//log.Println(ltpInfo)
		chForCandle <- &ltpInfo
	}
}
