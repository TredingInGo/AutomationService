package strategy

// buy ........
// if moving avg 7 cut moving avg 22 from down to up
// buy signal triggered.
// sl = low of the last candle.
// buying price  = above high of last candle
// TP = 2 * (buying price - sl)

// sell ...
// if moving avg 7 cut moving avg 22 from up side down
// sell signal triggered.
// sl = high of the last candle.
// buying price  = above low of last candle
// TP = 2 * (sl - buying price )

//func (s strategy) oneRsStrategy(data *models.SnapQuote) {
//
//	ltp := float64(data.LastTradedPrice) / 100
//	ma := getMovingAvg(s.pastData, ltp)
//	buy := data.BestFiveBuy
//	sell := data.BestFiveSell
//	ratio := 0.0
//	log.Printf("Best buy %v", buy)
//	log.Println("Best Sell ", sell)
//	buySum := 0.0
//	sellSum := 0.0
//	for i := 0; i < 5; i++ {
//		buySum += float64(buy[i].Quantity)
//		sellSum += float64(sell[i].Quantity)
//
//	}
//	ratio = buySum / math.Max(1.0, sellSum)
//	log.Printf("ma = %v, ratio = %v", ma, ratio)
//	if ma < ltp && ratio > 4 && t.flag == false {
//		// buy order
//		t = trade{
//			spot:      ltp,
//			sl:        ltp - 0.5,
//			tp:        ltp + 1.0,
//			qty:       1000.0,
//			orderType: "BUY",
//			flag:      true,
//		}
//		log.Printf("Trade %v", t)
//
//	}
//
//	if ma > ltp && ratio <= 0.25 && t.flag == false {
//		// sell order
//
//		t = trade{
//			spot:      ltp,
//			sl:        ltp + 0.5,
//			tp:        ltp - 1.0,
//			qty:       1000.0,
//			orderType: "SELL",
//			flag:      true,
//		}
//		log.Printf("Trade %v", t)
//	}
//}
//
//func getMovingAvg(data []smartapigo.CandleResponse, lastPrice float64) float64 {
//	sum := lastPrice
//	length := len(data)
//	for i := length - 9; i < length; i++ {
//		sum += data[i].Close
//	}
//	return sum / 10.0
//}
