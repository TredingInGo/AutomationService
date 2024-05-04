package Simulation

import (
	smartapigo "github.com/TredingInGo/smartapi"
	"log"
	"math"
	"time"
)

func PlaceBuyOrder(ohlcData []OHLC, index int, sl, takeProfit, buyPrice float64, length int) int {

	// Loop from the current index + 1 to the end of the OHLC data
	totalTrade++
	positionSize := int64(calculatePositionSize(buyPrice, sl))
	log.Printf("PostionSize: %v", positionSize)
	// When entering a trade, append the trade record with entry information
	trades = append(trades, tradeRecord{entryTimestamp: ohlcData[index].Timestamp, entryPrice: sl, profit: false})

	for i := index + 1; i < length; i++ {
		if ohlcData[i].High >= sl {
			// Stop Loss triggered, calculate loss
			loss := (buyPrice - sl) * float64(positionSize)
			lossCount++
			amount += loss
			log.Printf("Trade result: Loss %.2f\n", loss)
			// When exiting a trade, update the exit information
			minAmount = math.Min(minAmount, amount)
			trade = append(trade, tradeReport{amount, 0.0, loss, "breakOut", buyPrice, takeProfit, sl})
			return i
		} else if ohlcData[i].Low <= takeProfit {
			// Take Profit triggered, calculate profit
			profit := (buyPrice - takeProfit) * float64(positionSize)
			profitCount++
			amount += profit
			maxAmount = math.Max(maxAmount, amount)
			tp := (-buyPrice + sl) + buyPrice
			trades[len(trades)-1].exitTimestamp = ohlcData[i].Timestamp
			trades[len(trades)-1].exitPrice = tp
			trade = append(trade, tradeReport{amount, profit, 0.0, "breakOut", buyPrice, takeProfit, sl})
			log.Printf("Trade result: Profit %.2f\n", profit)
			return i
		}

	}

	// when time is 3:00 PM square of positions
	profit := (-ohlcData[length-1].Close + buyPrice) * float64(positionSize)
	amount += profit
	if profit > 0 {
		trade = append(trade, tradeReport{amount, profit, 0.0, "breakOut", buyPrice, takeProfit, sl})
		profitCount++
	} else {
		trade = append(trade, tradeReport{amount, 0.0, profit, "breakOut", buyPrice, takeProfit, sl})
		lossCount++
	}
	maxAmount = math.Max(maxAmount, amount)
	minAmount = math.Min(minAmount, amount)
	return length - 1
	// If the loop completes without hitting Stop Loss or Take Profit

}

func PlaceSellOrder(ohlcData []OHLC, index int, sl, takeProfit, sellPrice float64) {

	// Loop from the current index + 1 to the end of the OHLC data
	totalTrade++
	positionSize := int64(calculatePositionSize(sellPrice, sl))
	log.Printf("PositionSize: %v\n", positionSize)
	// When entering a trade, append the trade record with entry information
	trades = append(trades, tradeRecord{entryTimestamp: ohlcData[index].Timestamp, entryPrice: sl, profit: false})

	for i := index + 1; i < len(ohlcData); i++ {
		if ohlcData[i].High >= sl {
			// Stop Loss triggered, calculate loss
			loss := (sellPrice - sl) * float64(positionSize)
			lossCount++
			amount += loss
			log.Printf("Trade result: Loss %.2f\n", loss)
			// When exiting a trade, update the exit information
			tp := (sl - sellPrice) + sellPrice
			trades[len(trades)-1].exitTimestamp = ohlcData[i].Timestamp
			trades[len(trades)-1].exitPrice = tp
			trade = append(trade, tradeReport{amount, 0.0, loss, "breakOut", ohlcData[i].High, takeProfit, sl})

			minAmount = math.Min(minAmount, amount)

			return
		} else if ohlcData[i].Low <= takeProfit {
			// Take Profit triggered, calculate profit
			profit := (sellPrice - takeProfit) * float64(positionSize) * 5
			profitCount++
			amount += profit
			maxAmount = math.Max(maxAmount, amount)
			tp := (sl - sellPrice) + sellPrice
			trades[len(trades)-1].exitTimestamp = ohlcData[i].Timestamp
			trades[len(trades)-1].exitPrice = tp
			trade = append(trade, tradeReport{amount, profit, 0.0, "breakOut", ohlcData[i].High, takeProfit, sl})
			log.Printf("Trade result: Profit %.2f\n", profit)
			return
		} else {
			// when time is 3:00 PM square of positions
			tradeTime, err := time.Parse("2006-01-02 15:04:05-07:00", ohlcData[index].Timestamp)
			if err != nil {
				log.Println("Error parsing timestamp:", err)
				return
			}

			if tradeTime.Hour() == 15 && tradeTime.Minute() == 0 {
				profit := (sellPrice - ohlcData[i].Close) * 5
				amount += profit
				if profit > 0 {
					trade = append(trade, tradeReport{amount, profit, 0.0, "breakOut", ohlcData[i].High, takeProfit, sl})
					profitCount++
				} else {
					trade = append(trade, tradeReport{amount, 0.0, profit, "breakOut", ohlcData[i].High, takeProfit, sl})
					lossCount++
				}
				maxAmount = math.Max(maxAmount, amount)
				minAmount = math.Min(minAmount, amount)
				return
				// Perform your square-off logic here...
			}
		}
	}

	// If the loop completes without hitting Stop Loss or Take Profit
	log.Println("Trade result: No Stop Loss or Take Profit triggered.")
}

func calculatePositionSize(buyPrice, sl float64) float64 {
	if amount/buyPrice <= 1 {
		return 0
	}
	maxRiskPercent := 0.05 // 2% maximum risk allowed
	maxRiskAmount := amount * maxRiskPercent
	riskPerShare := math.Max(1, buyPrice-sl)
	positionSize := maxRiskAmount / riskPerShare
	return math.Min(amount/buyPrice, positionSize)
}

func simulateBuyorder(data []smartapigo.CandleResponse, quantity, idx int, price, sl, tp float64) (float64, int) {
	for i := idx; i < len(data); i++ {
		if data[i].Low <= sl {
			return (sl - price) * float64(quantity), i + 1
		} else if data[i].High >= tp {
			return (tp - price) * float64(quantity), i + 1
		}

	}
	return (data[len(data)-1].Close - price) * float64(quantity), len(data)
}

func simulateSellorder(data []smartapigo.CandleResponse, quantity, idx int, price, sl, tp float64) (float64, int) {
	for i := idx; i < len(data); i++ {
		if data[i].High >= sl {
			return (price - sl) * float64(quantity), i + 1
		} else if data[i].Low <= tp {
			return (price - tp) * float64(quantity), i + 1
		}

	}
	return (data[len(data)-1].Close - price) * float64(quantity), len(data)
}
