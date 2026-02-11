// Package strategy - indicators.go provides shared technical indicator calculations.
//
// These are used by multiple strategies (trend follow, mean reversion, breakout, momentum).
// All functions are stateless and deterministic — given the same candle slice, they
// return the same result.
package strategy

import "math"

// CalculateATR computes the Average True Range over the given period.
// True Range = max(high-low, |high-prevClose|, |low-prevClose|).
// Returns the simple average of the last `period` true ranges.
// Falls back to last candle's range if insufficient data.
func CalculateATR(candles []Candle, period int) float64 {
	if len(candles) == 0 {
		return 0
	}
	if len(candles) < period+1 {
		last := candles[len(candles)-1]
		return last.High - last.Low
	}

	var totalTR float64
	for i := len(candles) - period; i < len(candles); i++ {
		curr := candles[i]
		prev := candles[i-1]

		tr1 := curr.High - curr.Low
		tr2 := math.Abs(curr.High - prev.Close)
		tr3 := math.Abs(curr.Low - prev.Close)

		tr := math.Max(tr1, math.Max(tr2, tr3))
		totalTR += tr
	}

	return totalTR / float64(period)
}

// CalculateRSI computes the Relative Strength Index over the given period.
// Uses the Wilder smoothing method (exponential moving average of gains/losses).
// Returns a value between 0 and 100.
// Returns 50 (neutral) if insufficient data.
func CalculateRSI(candles []Candle, period int) float64 {
	if len(candles) < period+1 {
		return 50 // neutral if insufficient data
	}

	// Calculate initial average gain and loss over the first `period` changes.
	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		change := candles[i].Close - candles[i-1].Close
		if change > 0 {
			gainSum += change
		} else {
			lossSum += math.Abs(change)
		}
	}

	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)

	// Apply Wilder smoothing for remaining candles.
	for i := period + 1; i < len(candles); i++ {
		change := candles[i].Close - candles[i-1].Close
		var gain, loss float64
		if change > 0 {
			gain = change
		} else {
			loss = math.Abs(change)
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	if avgLoss == 0 {
		return 100 // no losses → RSI is maxed
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// CalculateSMA computes the Simple Moving Average of closing prices over the given period.
// Uses the last `period` candles. Returns 0 if insufficient data.
func CalculateSMA(candles []Candle, period int) float64 {
	if len(candles) < period || period <= 0 {
		return 0
	}

	var sum float64
	for i := len(candles) - period; i < len(candles); i++ {
		sum += candles[i].Close
	}
	return sum / float64(period)
}

// CalculateROC computes the Rate of Change (percentage) over the given period.
// ROC = (currentClose - closeNPeriodsAgo) / closeNPeriodsAgo
// Returns 0 if insufficient data or division by zero.
func CalculateROC(candles []Candle, period int) float64 {
	if len(candles) < period+1 || period <= 0 {
		return 0
	}

	current := candles[len(candles)-1].Close
	past := candles[len(candles)-1-period].Close

	if past == 0 {
		return 0
	}

	return (current - past) / past
}

// HighestHigh returns the highest high price over the last `period` candles.
// Returns 0 if no candles.
func HighestHigh(candles []Candle, period int) float64 {
	if len(candles) == 0 || period <= 0 {
		return 0
	}

	start := len(candles) - period
	if start < 0 {
		start = 0
	}

	highest := candles[start].High
	for i := start + 1; i < len(candles); i++ {
		if candles[i].High > highest {
			highest = candles[i].High
		}
	}
	return highest
}

// LowestLow returns the lowest low price over the last `period` candles.
// Returns 0 if no candles.
func LowestLow(candles []Candle, period int) float64 {
	if len(candles) == 0 || period <= 0 {
		return 0
	}

	start := len(candles) - period
	if start < 0 {
		start = 0
	}

	lowest := candles[start].Low
	for i := start + 1; i < len(candles); i++ {
		if candles[i].Low < lowest {
			lowest = candles[i].Low
		}
	}
	return lowest
}

// AverageVolume computes the average volume over the last `period` candles.
// Returns 0 if insufficient data.
func AverageVolume(candles []Candle, period int) float64 {
	if len(candles) == 0 || period <= 0 {
		return 0
	}

	start := len(candles) - period
	if start < 0 {
		start = 0
	}

	var totalVol float64
	count := 0
	for i := start; i < len(candles); i++ {
		totalVol += float64(candles[i].Volume)
		count++
	}

	if count == 0 {
		return 0
	}
	return totalVol / float64(count)
}
