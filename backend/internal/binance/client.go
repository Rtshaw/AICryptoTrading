// Package binance is a minimal client for Binance USDS-M Futures: REST
// (market data, account, order placement) and the market/user-data
// WebSocket streams. This talks directly to Binance's own public API -
// unlike the reference TWSE project, no local bridge process is needed.
//
// Every order-placement path in this codebase MUST go through
// Filters.MaxQtyForCap (see filters.go) before calling PlaceMarketOrder.
// This client itself does not enforce the $10 notional cap - that's a
// caller responsibility, deliberately kept out of the transport layer so
// there's exactly one place (internal/autotrader) that owns sizing policy.
package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrCode is a parsed Binance API error response, e.g. {"code":-4046,"msg":"No need to change margin type."}.
type ErrCode struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *ErrCode) Error() string {
	return fmt.Sprintf("binance: code=%d msg=%s", e.Code, e.Msg)
}

// AsErrCode extracts the Binance error code from err, if any.
func AsErrCode(err error) (*ErrCode, bool) {
	var ec *ErrCode
	if errors.As(err, &ec) {
		return ec, true
	}
	return nil, false
}

type Client struct {
	apiKey     string
	apiSecret  string
	restBase   string
	wsBase     string
	httpClient *http.Client

	clock *clockSync
}

func NewClient(apiKey, apiSecret, restBase, wsBase string) *Client {
	return &Client{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		restBase:   strings.TrimRight(restBase, "/"),
		wsBase:     strings.TrimRight(wsBase, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		clock:      newClockSync(),
	}
}

// do issues a REST request. When signed is true, timestamp/recvWindow and
// the HMAC signature are appended and the API key header is set; when
// false, the request is fully public (still sends the header if present,
// which is harmless and required for listenKey endpoints specifically -
// callers pass signed=false but must set needsKeyHeader for those).
func (c *Client) do(ctx context.Context, method, path string, params url.Values, signed bool, needsKeyHeader bool, out any) error {
	if params == nil {
		params = url.Values{}
	}

	if signed {
		needsKeyHeader = true
		params.Set("timestamp", strconv.FormatInt(c.clock.Now().UnixMilli(), 10))
		if params.Get("recvWindow") == "" {
			params.Set("recvWindow", "5000")
		}
		query := encodeSorted(params)
		params.Set("signature", sign(c.apiSecret, query))
	}

	query := encodeSorted(params)
	reqURL := c.restBase + path
	var req *http.Request
	var err error
	if method == http.MethodGet || method == http.MethodDelete {
		if query != "" {
			reqURL += "?" + query
		}
		req, err = http.NewRequestWithContext(ctx, method, reqURL, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, strings.NewReader(query))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if err != nil {
		return err
	}
	if needsKeyHeader {
		req.Header.Set("X-MBX-APIKEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("binance: request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("binance: read response %s %s: %w", method, path, err)
	}

	if resp.StatusCode >= 400 {
		var ec ErrCode
		if jsonErr := json.Unmarshal(body, &ec); jsonErr == nil && ec.Code != 0 {
			return &ec
		}
		return fmt.Errorf("binance: %s %s returned HTTP %d: %s", method, path, resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("binance: parse response %s %s: %w (body: %s)", method, path, err, truncate(body, 500))
		}
	}
	return nil
}

func encodeSorted(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(v.Get(k)))
	}
	return b.String()
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// numStr unmarshals a Binance JSON field that may arrive as either a string
// (the common case: "0.00123400") or a bare number, into a float64. Binance
// sends almost all numeric fields as strings; this guards against the rare
// exception without requiring every struct field to be `string`-typed.
type numStr float64

func (n *numStr) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*n = 0
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("binance: parse numeric field %q: %w", s, err)
	}
	*n = numStr(f)
	return nil
}

func (n numStr) Float() float64 { return float64(n) }
