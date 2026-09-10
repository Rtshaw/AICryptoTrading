package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"cryptotrading/internal/models"
)

// ExchangeInfoSymbol is the subset of GET /fapi/v1/exchangeInfo's per-symbol
// fields this project needs: trading status and the LOT_SIZE/MIN_NOTIONAL/
// PRICE_FILTER filters used to size and validate orders.
type ExchangeInfoSymbol struct {
	Symbol            string           `json:"symbol"`
	Status            string           `json:"status"` // "TRADING" when live
	ContractType      string           `json:"contractType"`
	QuoteAsset        string           `json:"quoteAsset"`
	QuantityPrecision int              `json:"quantityPrecision"`
	PricePrecision    int              `json:"pricePrecision"`
	Filters           []map[string]any `json:"filters"`
}

type exchangeInfoResponse struct {
	Symbols []ExchangeInfoSymbol `json:"symbols"`
}

// ExchangeInfo fetches the full symbol/filter table. Unauthenticated.
func (c *Client) ExchangeInfo(ctx context.Context) ([]ExchangeInfoSymbol, error) {
	var out exchangeInfoResponse
	if err := c.do(ctx, http.MethodGet, "/fapi/v1/exchangeInfo", nil, false, false, &out); err != nil {
		return nil, err
	}
	return out.Symbols, nil
}

// klineRow mirrors Binance's REST kline array-of-arrays shape:
// [openTime, open, high, low, close, volume, closeTime, ...].
type klineRow [12]any

// Klines returns historical candles for symbol, oldest first.
func (c *Client) Klines(ctx context.Context, symbol, interval string, limit int) ([]models.Candle, error) {
	params := url.Values{
		"symbol":   {symbol},
		"interval": {interval},
		"limit":    {strconv.Itoa(limit)},
	}
	var raw []klineRow
	if err := c.do(ctx, http.MethodGet, "/fapi/v1/klines", params, false, false, &raw); err != nil {
		return nil, err
	}

	out := make([]models.Candle, 0, len(raw))
	for _, k := range raw {
		openTimeMs, _ := k[0].(float64)
		out = append(out, models.Candle{
			Symbol: symbol,
			Ts:     time.UnixMilli(int64(openTimeMs)),
			Open:   parseAny(k[1]),
			High:   parseAny(k[2]),
			Low:    parseAny(k[3]),
			Close:  parseAny(k[4]),
			Volume: parseAny(k[5]),
		})
	}
	return out, nil
}

func parseAny(v any) float64 {
	switch t := v.(type) {
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	case float64:
		return t
	default:
		return 0
	}
}

// KlinesRange fetches every kline between start and end (inclusive),
// paginating past Binance's 1500-per-call limit via startTime/endTime -
// needed to pull enough history (weeks to months of 5m bars) for a
// meaningful backtest train/validation split, far more than the single
// SeedHistory call (limit-only, most-recent-N) used for live chart seeding.
func (c *Client) KlinesRange(ctx context.Context, symbol, interval string, start, end time.Time) ([]models.Candle, error) {
	const pageLimit = 1500
	var out []models.Candle
	cursor := start

	for cursor.Before(end) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		params := url.Values{
			"symbol":    {symbol},
			"interval":  {interval},
			"startTime": {strconv.FormatInt(cursor.UnixMilli(), 10)},
			"endTime":   {strconv.FormatInt(end.UnixMilli(), 10)},
			"limit":     {strconv.Itoa(pageLimit)},
		}
		var raw []klineRow
		if err := c.do(ctx, http.MethodGet, "/fapi/v1/klines", params, false, false, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			break
		}

		for _, k := range raw {
			openTimeMs, _ := k[0].(float64)
			out = append(out, models.Candle{
				Symbol: symbol,
				Ts:     time.UnixMilli(int64(openTimeMs)),
				Open:   parseAny(k[1]),
				High:   parseAny(k[2]),
				Low:    parseAny(k[3]),
				Close:  parseAny(k[4]),
				Volume: parseAny(k[5]),
			})
		}

		last := out[len(out)-1]
		if !last.Ts.After(cursor) {
			break // safety valve against an infinite loop if the API ever stops advancing
		}
		cursor = last.Ts.Add(time.Millisecond)

		if len(raw) < pageLimit {
			break // reached the end of available data before `end`
		}
	}

	return out, nil
}

// Ticker24hr is the subset of GET /fapi/v1/ticker/24hr this project needs
// for ranking candidates by liquidity in the AI daily watchlist selection
// (internal/watchlistai) - quote volume is the standard "how much money
// actually traded" liquidity signal, more meaningful across symbols of
// wildly different price/precision than raw base-asset volume.
type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	LastPrice          numStr `json:"lastPrice"`
	PriceChangePercent numStr `json:"priceChangePercent"`
	QuoteVolume        numStr `json:"quoteVolume"`
}

// Ticker24hrAll fetches the rolling 24h stats for every symbol in one call
// (no `symbol` param). Unauthenticated.
func (c *Client) Ticker24hrAll(ctx context.Context) ([]Ticker24hr, error) {
	var out []Ticker24hr
	if err := c.do(ctx, http.MethodGet, "/fapi/v1/ticker/24hr", nil, false, false, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PremiumIndexResult is the subset of GET /fapi/v1/premiumIndex used for
// funding-rate context - a crypto-perpetual-specific signal with no analog
// in the reference TWSE (equities) project.
type PremiumIndexResult struct {
	Symbol          string `json:"symbol"`
	MarkPrice       numStr `json:"markPrice"`
	LastFundingRate numStr `json:"lastFundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

func (c *Client) PremiumIndex(ctx context.Context, symbol string) (*PremiumIndexResult, error) {
	params := url.Values{"symbol": {symbol}}
	var out PremiumIndexResult
	if err := c.do(ctx, http.MethodGet, "/fapi/v1/premiumIndex", params, false, false, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PremiumIndexAll fetches funding rate/mark price for every symbol in one
// call (no `symbol` param) - used by internal/watchlistai to avoid one REST
// call per candidate when ranking dozens of symbols.
func (c *Client) PremiumIndexAll(ctx context.Context) ([]PremiumIndexResult, error) {
	var out []PremiumIndexResult
	if err := c.do(ctx, http.MethodGet, "/fapi/v1/premiumIndex", nil, false, false, &out); err != nil {
		return nil, err
	}
	return out, nil
}
