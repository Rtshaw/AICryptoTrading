// Package strategy computes shared technical indicators and a deterministic
// rule-based trading decision. The same indicator values feed both the AI
// prompt (internal/ai) and the signal engine, so what the rule engine saw
// and what the AI was told about always stay in sync.
package strategy

import "cryptotrading/internal/models"

// SMA returns the simple moving average series aligned to closes; entries
// before `period` samples are available are zero (caller should only read
// from index period-1 onward).
func SMA(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	sum := 0.0
	for i, c := range closes {
		sum += c
		if i >= period {
			sum -= closes[i-period]
		}
		if i >= period-1 {
			out[i] = sum / float64(period)
		}
	}
	return out
}

// RSI returns the Wilder relative-strength-index series.
func RSI(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if len(closes) < 2 {
		return out
	}
	var avgGain, avgLoss float64
	for i := 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		gain, loss := 0.0, 0.0
		if change > 0 {
			gain = change
		} else {
			loss = -change
		}
		if i <= period {
			avgGain += gain / float64(period)
			avgLoss += loss / float64(period)
			if i == period {
				out[i] = rsiFromAvg(avgGain, avgLoss)
			}
			continue
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		out[i] = rsiFromAvg(avgGain, avgLoss)
	}
	return out
}

func rsiFromAvg(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}

// EMA returns the exponential moving average series.
func EMA(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if len(closes) == 0 {
		return out
	}
	k := 2.0 / float64(period+1)
	out[0] = closes[0]
	for i := 1; i < len(closes); i++ {
		out[i] = closes[i]*k + out[i-1]*(1-k)
	}
	return out
}

// MACD returns the MACD line, signal line, and histogram using the
// conventional 12/26/9 periods.
func MACD(closes []float64) (macd, signal, hist []float64) {
	ema12 := EMA(closes, 12)
	ema26 := EMA(closes, 26)
	macd = make([]float64, len(closes))
	for i := range closes {
		macd[i] = ema12[i] - ema26[i]
	}
	signal = EMA(macd, 9)
	hist = make([]float64, len(closes))
	for i := range closes {
		hist[i] = macd[i] - signal[i]
	}
	return macd, signal, hist
}

// VWAP returns the cumulative volume-weighted average price, resetting at
// every UTC calendar-day boundary. Crypto trades 24/7 with no exchange
// session to key off (unlike an equity market's open/close), so the day
// boundary is the natural periodic reset. A defensive gap check also resets
// the accumulator if there's a large time gap between candles (e.g. after
// stream downtime), so a stale accumulator never silently carries forward.
func VWAP(candles []models.Candle) []float64 {
	out := make([]float64, len(candles))
	var cumPV, cumVol float64
	for i, c := range candles {
		if i > 0 {
			prev := candles[i-1]
			newUTCDay := c.Ts.UTC().YearDay() != prev.Ts.UTC().YearDay() || c.Ts.UTC().Year() != prev.Ts.UTC().Year()
			gap := c.Ts.Sub(prev.Ts).Minutes() > 60
			if newUTCDay || gap {
				cumPV, cumVol = 0, 0
			}
		}
		typical := (c.High + c.Low + c.Close) / 3
		cumPV += typical * c.Volume
		cumVol += c.Volume
		if cumVol > 0 {
			out[i] = cumPV / cumVol
		} else {
			out[i] = c.Close
		}
	}
	return out
}

// VolumeRatio compares each bar's volume to the trailing `period`-bar average
// (excluding the bar itself); a value of 2.0 means "twice the recent average".
func VolumeRatio(candles []models.Candle, period int) []float64 {
	out := make([]float64, len(candles))
	var sum float64
	for i, c := range candles {
		if i >= period {
			avg := sum / float64(period)
			if avg > 0 {
				out[i] = c.Volume / avg
			}
			sum -= candles[i-period].Volume
		}
		sum += c.Volume
	}
	return out
}

func closesOf(candles []models.Candle) []float64 {
	out := make([]float64, len(candles))
	for i, c := range candles {
		out[i] = c.Close
	}
	return out
}
