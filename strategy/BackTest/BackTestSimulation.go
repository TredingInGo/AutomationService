package BackTest

import (
	smartapigo "github.com/TredingInGo/smartapi"
	"time"
)

func simulate(spot, sl, tp float64, data []smartapigo.CandleResponse, idx *int, orderType string) float64 {
	//trailingStopLoss := sl
	//isOrderTriggered := false
	*idx++
	for *idx < len(data)-1 {

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
			if data[*idx].Close <= tp {
				return spot - tp
			}
			if data[*idx].Close >= sl {
				return spot - sl
			}

			if spot-5 > data[*idx].Close {
				//trailingStopLoss -= 5
				//spot -= 5
				//sl = trailingStopLoss
			}
		}
		*idx++
	}

	return 0
}
