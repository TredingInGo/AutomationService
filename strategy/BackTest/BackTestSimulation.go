package BackTest

import (
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"time"
)

func simulate(spot, sl, tp float64, data []smartapigo.CandleResponse, idx *int, orderType string) float64 {
	//trailingStopLoss := sl
	isOrderTriggered := false
	for *idx < len(data)-1 {
		if isOrderTriggered == false && data[*idx].Close > spot+(spot*0.1) {
			log.Println("current Price is 1% more than spot price")
			return 0.0
		}
		if isOrderTriggered == false && data[*idx].Close >= spot && data[*idx].Close <= spot+1 {
			log.Println("Order triggered")
			isOrderTriggered = true
			spot = data[*idx].Close
		}

		if isOrderTriggered {
			currentTime := data[*idx].Timestamp
			compareTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 15, 15, 0, 0, currentTime.Location())
			if currentTime.After(compareTime) {
				if orderType == "BUY" {
					return data[*idx].Close - spot
				} else {
					return spot - data[*idx].Close
				}
			}
			if orderType == "BUY" {

				if data[*idx].Close >= tp {
					return tp - spot
				}
				if data[*idx].Close <= sl {
					return sl - spot
				}
				if spot+5 < data[*idx].Close {
					//trailingStopLoss += 5
					//spot += 5
					//sl = trailingStopLoss
				}
			}

			if orderType == "SELL" {
				if data[*idx].Close >= sl {
					return spot - sl
				}
				if data[*idx].Close <= tp {
					return spot - tp
				}
				if spot-5 > data[*idx].Close {
					//trailingStopLoss -= 5
					//spot -= 5
					//sl = trailingStopLoss
				}
			}
			*idx++
		}

	}
	return 0
}
