package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// clockSync tracks the offset between local time and Binance server time so
// signed requests don't get rejected with -1021 "Timestamp for this request
// is outside of the recvWindow" due to local clock drift.
type clockSync struct {
	mu     sync.RWMutex
	offset time.Duration
}

func newClockSync() *clockSync {
	return &clockSync{}
}

func (c *clockSync) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Now().Add(c.offset)
}

// Sync fetches Binance's server time and updates the offset. Call once at
// startup and then periodically (e.g. hourly) from a background loop.
func (c *clockSync) Sync(ctx context.Context, restBase string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, restBase+"/fapi/v1/time", nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		ServerTime int64 `json:"serverTime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}

	serverTime := time.UnixMilli(out.ServerTime)
	c.mu.Lock()
	c.offset = time.Until(serverTime)
	c.mu.Unlock()
	return nil
}

// SyncClock exposes the client's clock sync for the startup/periodic loop in cmd/server.
func (c *Client) SyncClock(ctx context.Context) error {
	return c.clock.Sync(ctx, c.restBase)
}
