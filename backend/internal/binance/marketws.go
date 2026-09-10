package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"cryptotrading/internal/models"
)

// KlineEvent is one kline update for the configured interval. Closed is
// true only on the bar that finalizes the interval (Binance's "x" field) -
// callers should persist every event (for the live-updating current bar)
// but only fire close-triggered logic (indicator recompute, signal engine)
// when Closed.
type KlineEvent struct {
	Candle models.Candle
	Closed bool
}

// MarkPriceEvent carries the mark price and funding rate for a symbol,
// pushed roughly once a second by the @markPrice stream.
type MarkPriceEvent struct {
	Symbol          string
	MarkPrice       float64
	FundingRate     float64
	NextFundingTime time.Time
}

type combinedStreamEnvelope struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

type klineStreamData struct {
	Symbol string `json:"s"`
	Kline  struct {
		StartTime int64  `json:"t"`
		Symbol    string `json:"s"`
		Interval  string `json:"i"`
		Open      numStr `json:"o"`
		Close     numStr `json:"c"`
		High      numStr `json:"h"`
		Low       numStr `json:"l"`
		Volume    numStr `json:"v"`
		Closed    bool   `json:"x"`
	} `json:"k"`
}

type markPriceStreamData struct {
	Symbol          string `json:"s"`
	MarkPrice       numStr `json:"p"`
	FundingRate     numStr `json:"r"`
	NextFundingTime int64  `json:"T"`
}

// RunMarketStream connects to the combined kline+markPrice WebSocket for the
// given symbols and dispatches events until ctx is cancelled, reconnecting
// with backoff on any disconnect. onKline/onMarkPrice may be called from
// this goroutine - callers needing to do slow work (DB writes, AI calls)
// should hand off to their own goroutine rather than blocking here.
func (c *Client) RunMarketStream(ctx context.Context, symbols []string, interval string, onKline func(KlineEvent), onMarkPrice func(MarkPriceEvent)) error {
	streams := make([]string, 0, len(symbols)*2)
	for _, s := range symbols {
		lower := strings.ToLower(s)
		streams = append(streams, lower+"@kline_"+interval, lower+"@markPrice@1s")
	}
	streamParam := strings.Join(streams, "/")
	// Binance's combined-stream path expects literal "/" separators between
	// stream names, so the streams value is NOT query-escaped as a whole.
	wsURL := c.wsBase + "/stream?streams=" + streamParam

	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := c.runMarketStreamOnce(ctx, wsURL, onKline, onMarkPrice)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Printf("binance: market stream disconnected (%v), reconnecting in %s", err, backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// readIdleTimeout bounds how long we'll wait for a frame before treating
// the connection as dead and reconnecting. Plain WS keepalive (TCP-level)
// can leave a connection that completed its handshake but silently never
// delivers data frames again - observed in practice behind some
// firewalls/VPNs/antivirus that interfere with long-lived WSS streams while
// leaving standard HTTPS REST calls unaffected. Without an explicit
// deadline, ReadMessage blocks forever in that case and the reconnect loop
// never fires, so the chart just goes silently stale with no error logged.
const readIdleTimeout = 30 * time.Second

func (c *Client) runMarketStreamOnce(ctx context.Context, wsURL string, onKline func(KlineEvent), onMarkPrice func(MarkPriceEvent)) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	log.Printf("binance: market stream connected (%d streams)", strings.Count(wsURL, "@kline_"))

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		conn.SetReadDeadline(time.Now().Add(readIdleTimeout))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read (no frame received for %s, connection may be silently blocked by a firewall/VPN/AV even though the handshake succeeded): %w", readIdleTimeout, err)
		}

		var env combinedStreamEnvelope
		if err := json.Unmarshal(msg, &env); err != nil {
			log.Printf("binance: market stream: bad envelope: %v", err)
			continue
		}

		switch {
		case strings.Contains(env.Stream, "@kline_"):
			var d klineStreamData
			if err := json.Unmarshal(env.Data, &d); err != nil {
				log.Printf("binance: market stream: bad kline payload: %v", err)
				continue
			}
			onKline(KlineEvent{
				Candle: models.Candle{
					Symbol: d.Kline.Symbol,
					Ts:     time.UnixMilli(d.Kline.StartTime),
					Open:   d.Kline.Open.Float(),
					High:   d.Kline.High.Float(),
					Low:    d.Kline.Low.Float(),
					Close:  d.Kline.Close.Float(),
					Volume: d.Kline.Volume.Float(),
				},
				Closed: d.Kline.Closed,
			})
		case strings.Contains(env.Stream, "@markPrice"):
			var d markPriceStreamData
			if err := json.Unmarshal(env.Data, &d); err != nil {
				log.Printf("binance: market stream: bad markPrice payload: %v", err)
				continue
			}
			onMarkPrice(MarkPriceEvent{
				Symbol:          d.Symbol,
				MarkPrice:       d.MarkPrice.Float(),
				FundingRate:     d.FundingRate.Float(),
				NextFundingTime: time.UnixMilli(d.NextFundingTime),
			})
		}
	}
}
