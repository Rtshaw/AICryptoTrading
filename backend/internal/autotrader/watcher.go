package autotrader

import (
	"context"
	"log"

	"cryptotrading/internal/models"
	"cryptotrading/internal/signalengine"
	"cryptotrading/internal/strategy"
	"cryptotrading/internal/ws"
)

// OnCandleClose is wired into marketdata.Service.OnCandleClose. It runs the
// Silver Bullet detector (internal/strategy.DecideSilverBullet) on every
// candle close, ALWAYS records an auto_trade_log entry for the evaluation
// (the transparency requirement), and only calls the AI (and potentially
// places a real order) when a genuinely NEW sweep+FVG setup is found - "new"
// meaning its FVG bar timestamp hasn't been acted on before for this symbol
// (see lastSignaledFVG), which is the Silver Bullet equivalent of the
// reference project's rule-transition gate. This must run in its own
// goroutine (it makes network calls) - the caller (marketdata.Service)
// already does this.
func (t *Trader) OnCandleClose(ctx context.Context, c models.Candle) {
	symbol := c.Symbol

	// cmd/server/streams.go keeps strategy.AnchorSymbols (BTC/ETH) streaming
	// at all times for SMT divergence lookups, even when the AI daily
	// watchlist selection (internal/watchlistai) has dropped them from the
	// tradable set - so a candle close for one of them must NOT be evaluated
	// for trading here, only ever read as reference data via GetCandles.
	if onWatchlist, err := t.isWatchlisted(ctx, symbol); err != nil || !onWatchlist {
		return
	}

	params := t.SBParams()
	candles, err := t.Market.GetCandles(ctx, symbol, 100)
	if err != nil || len(candles) < params.SwingLookback+5 {
		return // not enough history yet; nothing meaningful to evaluate or log
	}
	anchorCandles, err := t.Market.GetCandles(ctx, strategy.AnchorSymbolFor(symbol), 100)
	if err != nil {
		anchorCandles = nil // SMT confirmation fails closed (see strategy.checkSMT) rather than blocking evaluation entirely
	}
	setup := strategy.DecideSilverBullet(candles, anchorCandles, params, c.Ts)

	entry := models.AutoTradeLogEntry{Symbol: symbol, Ts: c.Ts, RuleAction: setup.Action, RuleReason: setup.Reason}

	t.mu.Lock()
	lastFVG, seen := t.lastSignaledFVG[symbol]
	isNewSetup := setup.Action != models.SignalHold && (!seen || setup.FVGTs.After(lastFVG))
	if isNewSetup {
		t.lastSignaledFVG[symbol] = setup.FVGTs
	}
	t.mu.Unlock()

	if !isNewSetup {
		entry.Decision = models.DecisionHold
		t.insertLog(ctx, entry)
		return
	}

	log.Printf("autotrader: Silver Bullet setup for %s: %s (%s) - requesting AI evaluation", symbol, setup.Action, setup.Reason)

	if !t.AI.Enabled() {
		entry.Decision = models.DecisionSkipped
		entry.SkipReason = "ai_not_configured"
		t.insertLog(ctx, entry)
		return
	}

	signal, err := signalengine.GenerateAndBroadcast(ctx, t.signalDeps(), symbol, setup)
	if err != nil {
		entry.Decision = models.DecisionError
		entry.SkipReason = err.Error()
		t.insertLog(ctx, entry)
		return
	}
	entry.AISignalID = &signal.ID

	if signal.Action == models.SignalHold {
		entry.Decision = models.DecisionHold
		entry.SkipReason = "ai_overrode_rule_to_hold"
		t.insertLog(ctx, entry)
		return
	}

	decision, reason, orderID := t.execute(ctx, signal)
	entry.Decision = decision
	entry.SkipReason = reason
	entry.OrderID = orderID
	t.insertLog(ctx, entry)
}

// isWatchlisted reports whether symbol is currently enabled in the user's
// trading watchlist - as opposed to being streamed only as an SMT anchor
// reference (see strategy.AnchorSymbols).
func (t *Trader) isWatchlisted(ctx context.Context, symbol string) (bool, error) {
	var exists bool
	err := t.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM watchlist WHERE symbol = $1 AND enabled)`, symbol).Scan(&exists)
	return exists, err
}

func (t *Trader) insertLog(ctx context.Context, e models.AutoTradeLogEntry) {
	err := t.Pool.QueryRow(ctx, `
		INSERT INTO auto_trade_log (symbol, ts, rule_action, rule_reason, ai_signal_id, decision, skip_reason, order_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
	`, e.Symbol, e.Ts, e.RuleAction, e.RuleReason, e.AISignalID, e.Decision, nullIfEmpty(e.SkipReason), e.OrderID).Scan(&e.ID)
	if err != nil {
		log.Printf("autotrader: insert auto_trade_log failed: %v", err)
		return
	}
	t.Hub.Broadcast(ws.Message{Type: "autotrade_log", Data: e})
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
