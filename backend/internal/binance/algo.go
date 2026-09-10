package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"cryptotrading/internal/models"
)

// AlgoOrderResponse is the response shape for POST/GET /fapi/v1/algoOrder.
type AlgoOrderResponse struct {
	AlgoID       int64  `json:"algoId"`
	ClientAlgoID string `json:"clientAlgoId"`
	Symbol       string `json:"symbol"`
	Side         string `json:"side"`
	OrderType    string `json:"orderType"`
	AlgoStatus   string `json:"algoStatus"` // NEW | TRIGGERED | FILLED | CANCELED | EXPIRED
	TriggerPrice numStr `json:"triggerPrice"`
	ActualPrice  numStr `json:"actualPrice"` // average execution price; 0 until filled
}

// PlaceClosePositionAlgoOrder submits a STOP_MARKET or TAKE_PROFIT_MARKET
// conditional order via Binance's Algo Order service. USDS-M Futures
// migrated conditional order types (STOP_MARKET/TAKE_PROFIT_MARKET/STOP/
// TAKE_PROFIT/TRAILING_STOP_MARKET) off the legacy POST /fapi/v1/order
// endpoint on 2025-12-09 - that endpoint now rejects them with error -4120,
// so this MUST go through /fapi/v1/algoOrder instead.
//
// closePosition=true closes the entire current position when triggered (no
// quantity needed - avoids any mismatch with the actual position size, and
// stays correct even if the position was partially reduced after this order
// was placed). Binance allows at most one active closePosition order per
// type per direction and automatically cancels the sibling stop-loss/
// take-profit order once either one fires and the position goes flat - no
// manual OCO (one-cancels-other) bookkeeping is needed on our end.
func (c *Client) PlaceClosePositionAlgoOrder(ctx context.Context, symbol string, side models.OrderSide, orderType string, triggerPrice float64, clientAlgoID string) (*AlgoOrderResponse, error) {
	params := url.Values{
		"algoType":      {"CONDITIONAL"},
		"symbol":        {symbol},
		"side":          {string(side)},
		"type":          {orderType}, // "STOP_MARKET" or "TAKE_PROFIT_MARKET"
		"triggerPrice":  {strconv.FormatFloat(triggerPrice, 'f', -1, 64)},
		"closePosition": {"true"},
		// MARK_PRICE (not the default CONTRACT_PRICE/last-trade-price) so a
		// brief last-price wick can't trigger the stop by itself.
		"workingType": {"MARK_PRICE"},
	}
	if clientAlgoID != "" {
		params.Set("clientAlgoId", clientAlgoID)
	}

	var out AlgoOrderResponse
	if err := c.do(ctx, http.MethodPost, "/fapi/v1/algoOrder", params, true, true, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QueryAlgoOrder fetches the current status of a previously-placed algo
// order (signed) - the algo-order counterpart to QueryOrder, used by
// internal/autotrader.ReconcilePendingOrders.
func (c *Client) QueryAlgoOrder(ctx context.Context, algoID int64) (*AlgoOrderResponse, error) {
	params := url.Values{"algoId": {strconv.FormatInt(algoID, 10)}}
	var out AlgoOrderResponse
	if err := c.do(ctx, http.MethodGet, "/fapi/v1/algoOrder", params, true, true, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelAlgoOrder cancels a resting algo order (signed).
func (c *Client) CancelAlgoOrder(ctx context.Context, algoID int64) error {
	params := url.Values{"algoId": {strconv.FormatInt(algoID, 10)}}
	return c.do(ctx, http.MethodDelete, "/fapi/v1/algoOrder", params, true, true, nil)
}
