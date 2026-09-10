package strategy

import (
	"strings"
	"testing"
	"time"

	"cryptotrading/internal/models"
)

func testParams() SBParams {
	return SBParams{
		SwingLookback:   5,
		MinFVGSizePct:   0.1,
		MaxBarsForSweep: 3,
		StopBufferPct:   0.1,
		RiskRewardRatio: 2.0,

		MinDisplacementBodyPct:   0.6,
		MaxOpposingWickPct:       0.25,
		OTEMinRetrace:            0.62,
		OTEMaxRetrace:            0.79,
		MaxBarsForOTE:            8,
		BreakerLookback:          10,
		RequireBreakerConfluence: true,
		RequireSMTDivergence:     true,
	}
}

// buildBullishSetup returns an 11-candle series (indices 0-10) engineered to
// satisfy every ICT-2026 tier:
//   - idx6 is a bearish order-block candle (later the Breaker Block)
//   - idx7 sweeps below the idx0-6 swing low (98.0) then closes back above it
//   - idx8 is a strong bullish displacement candle (body 94.7% of range,
//     small lower wick), whose high (104.1) is the leg extreme "B"
//   - idx7/8/9 form a qualifying bullish Fair Value Gap [100.5, 100.7]
//   - the Breaker Block [100.2, 100.5] overlaps the OTE zone [~99.28, ~100.32]
//   - idx10 is the first bar to retrace into the OTE zone (Low=99.9),
//     without having closed beyond the stop (97.902) on any prior bar
//
// buildBullishAnchor returns a same-length, same-timestamps reference
// series that does NOT sweep a new low in sync with idx7 (its low over
// idx7-10 stays above its own prior swing low), confirming SMT divergence.
func buildBullishSetup(start time.Time) []models.Candle {
	ts := func(i int) time.Time { return start.Add(time.Duration(i) * 5 * time.Minute) }
	return []models.Candle{
		{Ts: ts(0), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(1), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(2), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(3), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(4), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(5), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		// order block / future Breaker Block: bearish candle, body [100.2,100.5]
		{Ts: ts(6), Open: 100.5, High: 100.6, Low: 100.1, Close: 100.2, Volume: 10},
		// sweep bar (c0 of the FVG): wicks below 100.1 (the swing low) then closes back above it
		{Ts: ts(7), Open: 100.2, High: 100.5, Low: 98.0, Close: 100.4, Volume: 20},
		// displacement bar (c1): strong bullish body, small lower wick
		{Ts: ts(8), Open: 100.4, High: 104.1, Low: 100.3, Close: 104.0, Volume: 20},
		// c2: low (100.7) gaps above c0's high (100.5) -> bullish FVG [100.5, 100.7]
		{Ts: ts(9), Open: 100.9, High: 101.2, Low: 100.7, Close: 101.0, Volume: 20},
		// OTE pullback bar: dips to 99.9, inside the ~[99.28, 100.32] OTE zone
		{Ts: ts(10), Open: 100.9, High: 101.0, Low: 99.9, Close: 100.0, Volume: 20},
	}
}

func buildBullishAnchor(start time.Time, symbol string, syncSweep bool) []models.Candle {
	ts := func(i int) time.Time { return start.Add(time.Duration(i) * 5 * time.Minute) }
	c := []models.Candle{
		{Symbol: symbol, Ts: ts(0), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(1), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(2), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(3), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(4), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(5), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(6), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(7), Open: 50.3, High: 50.4, Low: 50.25, Close: 50.35, Volume: 10},
		{Symbol: symbol, Ts: ts(8), Open: 50.35, High: 50.45, Low: 50.3, Close: 50.4, Volume: 10},
		{Symbol: symbol, Ts: ts(9), Open: 50.4, High: 50.45, Low: 50.3, Close: 50.35, Volume: 10},
		{Symbol: symbol, Ts: ts(10), Open: 50.35, High: 50.4, Low: 50.3, Close: 50.35, Volume: 10},
	}
	if syncSweep {
		// Anchor ALSO sweeps a new low in sync with the main symbol -
		// broad-market weakness, not an isolated stop-hunt -> no divergence.
		c[7] = models.Candle{Symbol: symbol, Ts: ts(7), Open: 50.3, High: 50.35, Low: 50.1, Close: 50.15, Volume: 10}
	}
	return c
}

// buildBearishSetup mirrors buildBullishSetup for the short side: idx6 is a
// bullish order block, idx7 sweeps above the swing high then closes back
// below it, idx8 is a strong bearish displacement candle, idx7/8/9 form a
// bearish FVG, and idx10 retraces up into the OTE zone.
func buildBearishSetup(start time.Time) []models.Candle {
	ts := func(i int) time.Time { return start.Add(time.Duration(i) * 5 * time.Minute) }
	return []models.Candle{
		{Ts: ts(0), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(1), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(2), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(3), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(4), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		{Ts: ts(5), Open: 100.3, High: 100.4, Low: 100.2, Close: 100.3, Volume: 10},
		// order block / future Breaker Block: bullish candle, body [100.1,100.5]
		{Ts: ts(6), Open: 100.1, High: 100.6, Low: 100.0, Close: 100.5, Volume: 10},
		// sweep bar: wicks above 100.6 (the swing high) then closes back below it
		{Ts: ts(7), Open: 100.5, High: 102.5, Low: 100.4, Close: 100.3, Volume: 20},
		// displacement bar: strong bearish body, small upper wick
		{Ts: ts(8), Open: 100.3, High: 100.4, Low: 97.0, Close: 97.5, Volume: 20},
		// c2: high (97.8) gaps below c0's low (100.4) -> bearish FVG [97.8, 100.4]
		{Ts: ts(9), Open: 97.6, High: 97.8, Low: 97.4, Close: 97.5, Volume: 20},
		// OTE pullback bar: rises to 100.5, inside the ~[100.41, 101.35] OTE zone
		{Ts: ts(10), Open: 97.6, High: 100.5, Low: 97.5, Close: 97.8, Volume: 20},
	}
}

func buildBearishAnchor(start time.Time, symbol string) []models.Candle {
	ts := func(i int) time.Time { return start.Add(time.Duration(i) * 5 * time.Minute) }
	return []models.Candle{
		{Symbol: symbol, Ts: ts(0), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(1), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(2), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(3), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(4), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(5), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(6), Open: 50.3, High: 50.4, Low: 50.2, Close: 50.3, Volume: 10},
		// Anchor does NOT make a new high in sync with the main symbol's sweep.
		{Symbol: symbol, Ts: ts(7), Open: 50.3, High: 50.35, Low: 50.25, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(8), Open: 50.3, High: 50.35, Low: 50.25, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(9), Open: 50.3, High: 50.35, Low: 50.25, Close: 50.3, Volume: 10},
		{Symbol: symbol, Ts: ts(10), Open: 50.3, High: 50.35, Low: 50.25, Close: 50.3, Volume: 10},
	}
}

var baseStart = time.Date(2026, 1, 15, 3, 5, 0, 0, time.UTC)

func approxEqual(a, b, tol float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= tol
}

func TestDecideSilverBullet_BullishFullSetup(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)

	if sig.Action != models.SignalBuy {
		t.Fatalf("expected BUY, got %s (%s)", sig.Action, sig.Reason)
	}
	if !(sig.StopLoss < sig.Entry && sig.Entry < sig.TakeProfit) {
		t.Errorf("expected StopLoss < Entry < TakeProfit, got %.4f < %.4f < %.4f", sig.StopLoss, sig.Entry, sig.TakeProfit)
	}
	if sig.FVGLow != 100.5 || sig.FVGHigh != 100.7 {
		t.Errorf("expected FVG zone [100.5, 100.7], got [%.4f, %.4f]", sig.FVGLow, sig.FVGHigh)
	}
	if sig.DisplacementBodyPct < 0.6 {
		t.Errorf("expected displacement body pct >= 0.6, got %.4f", sig.DisplacementBodyPct)
	}
	if !approxEqual(sig.OTELow, 99.281, 0.01) || !approxEqual(sig.OTEHigh, 100.318, 0.01) {
		t.Errorf("expected OTE zone ~[99.281, 100.318], got [%.4f, %.4f]", sig.OTELow, sig.OTEHigh)
	}
	if sig.BreakerLow != 100.2 || sig.BreakerHigh != 100.5 {
		t.Errorf("expected Breaker zone [100.2, 100.5], got [%.4f, %.4f]", sig.BreakerLow, sig.BreakerHigh)
	}
	if !sig.SMTConfirmed || sig.SMTAnchorSymbol != "BTCUSDT" {
		t.Errorf("expected SMT confirmed against BTCUSDT, got confirmed=%v anchor=%q", sig.SMTConfirmed, sig.SMTAnchorSymbol)
	}
}

func TestDecideSilverBullet_BearishFullSetup(t *testing.T) {
	candles := buildBearishSetup(baseStart)
	anchor := buildBearishAnchor(baseStart, "BTCUSDT")
	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)

	if sig.Action != models.SignalSell {
		t.Fatalf("expected SELL, got %s (%s)", sig.Action, sig.Reason)
	}
	if !(sig.TakeProfit < sig.Entry && sig.Entry < sig.StopLoss) {
		t.Errorf("expected TakeProfit < Entry < StopLoss, got %.4f < %.4f < %.4f", sig.TakeProfit, sig.Entry, sig.StopLoss)
	}
	if sig.FVGLow != 97.8 || sig.FVGHigh != 100.4 {
		t.Errorf("expected FVG zone [97.8, 100.4], got [%.4f, %.4f]", sig.FVGLow, sig.FVGHigh)
	}
	if !sig.SMTConfirmed {
		t.Errorf("expected SMT confirmed, got %v (%s)", sig.SMTConfirmed, sig.Reason)
	}
}

// TestDecideSilverBullet_NoTimeGating confirms a qualifying setup is
// detected regardless of what time of day it falls on - by explicit user
// decision this detector does not restrict to any session window (crypto
// trades 24/7; see the package doc comment).
func TestDecideSilverBullet_NoTimeGating(t *testing.T) {
	for _, hour := range []int{0, 3, 9, 12, 18, 23} {
		start := time.Date(2026, 1, 15, hour, 5, 0, 0, time.UTC)
		candles := buildBullishSetup(start)
		anchor := buildBullishAnchor(start, "BTCUSDT", false)
		sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
		if sig.Action != models.SignalBuy {
			t.Errorf("hour=%d: expected BUY regardless of time of day, got %s (%s)", hour, sig.Action, sig.Reason)
		}
	}
}

func TestDecideSilverBullet_SweepWithNoFVG(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	// Collapse the gap: c2 (idx9)'s low no longer clears c0 (idx7)'s high.
	candles[9].Low = 100.5
	candles[9].Close = 100.55
	candles[9].High = 100.6

	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD when no FVG forms, got %s (%s)", sig.Action, sig.Reason)
	}
}

func TestDecideSilverBullet_SubThresholdFVGIgnored(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	// Shrink the gap to ~0.01%, well below MinFVGSizePct=0.1%.
	candles[9].Low = 100.51
	candles[9].Close = 100.55
	candles[9].High = 100.6

	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD for a sub-threshold FVG, got %s (%s)", sig.Action, sig.Reason)
	}
}

func TestDecideSilverBullet_NoSweepNoSignal(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	// Raise the sweep bar's low so it never dips below the prior swing low.
	candles[7].Low = 100.3
	candles[7].Open = 100.3
	candles[7].High = 100.5

	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD without a genuine sweep, got %s (%s)", sig.Action, sig.Reason)
	}
}

// TestDecideSilverBullet_WeakDisplacementIgnored verifies the ICT-2026
// displacement filter: a wide-ranged, small-bodied idx8 (indecisive, mostly
// wick) must not qualify as the FVG's displacement candle even though the
// same sweep and FVG gap are otherwise present.
func TestDecideSilverBullet_WeakDisplacementIgnored(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	candles[8] = models.Candle{Ts: candles[8].Ts, Open: 100.4, High: 104.1, Low: 97.0, Close: 101.0, Volume: 20}

	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD for a weak (non-displacement) candle, got %s (%s)", sig.Action, sig.Reason)
	}
}

// TestDecideSilverBullet_NoBreakerConfluenceIgnored verifies the Unicorn
// Model gate: moving the order-block candle away from the OTE zone (so it
// no longer overlaps) must block the signal when RequireBreakerConfluence
// is on, even though sweep/displacement/FVG are all still valid.
func TestDecideSilverBullet_NoBreakerConfluenceIgnored(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	candles[6] = models.Candle{Ts: candles[6].Ts, Open: 105.0, High: 105.1, Low: 104.8, Close: 104.9, Volume: 10}

	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD without breaker/OTE confluence, got %s (%s)", sig.Action, sig.Reason)
	}
}

// TestDecideSilverBullet_BreakerConfluenceOptionalWhenDisabled confirms the
// same non-overlapping order block from the test above stops gating the
// signal once RequireBreakerConfluence is turned off.
func TestDecideSilverBullet_BreakerConfluenceOptionalWhenDisabled(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	candles[6] = models.Candle{Ts: candles[6].Ts, Open: 105.0, High: 105.1, Low: 104.8, Close: 104.9, Volume: 10}

	params := testParams()
	params.RequireBreakerConfluence = false
	sig := DecideSilverBullet(candles, anchor, params, candles[len(candles)-1].Ts)
	if sig.Action != models.SignalBuy {
		t.Fatalf("expected BUY once breaker confluence isn't required, got %s (%s)", sig.Action, sig.Reason)
	}
}

// TestDecideSilverBullet_OTENotReachedIgnored verifies that price running
// away without ever retracing into the OTE zone leaves the setup
// unconsummated (HOLD), even though sweep/displacement/FVG/breaker all qualify.
func TestDecideSilverBullet_OTENotReachedIgnored(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", false)
	candles[10] = models.Candle{Ts: candles[10].Ts, Open: 104.0, High: 104.3, Low: 103.8, Close: 104.1, Volume: 20}

	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD when price never retraces into the OTE zone, got %s (%s)", sig.Action, sig.Reason)
	}
}

// TestDecideSilverBullet_SMTNotConfirmedHolds verifies the SMT divergence
// gate: when the anchor symbol ALSO sweeps a new low in sync (broad-market
// weakness, not an isolated stop-hunt), the otherwise-complete setup must
// HOLD instead of firing BUY.
func TestDecideSilverBullet_SMTNotConfirmedHolds(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", true)

	sig := DecideSilverBullet(candles, anchor, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD when anchor doesn't diverge, got %s (%s)", sig.Action, sig.Reason)
	}
	if !strings.Contains(sig.Reason, "SMT") {
		t.Errorf("expected the HOLD reason to mention SMT, got %q", sig.Reason)
	}
}

// TestDecideSilverBullet_SMTOptionalWhenDisabled confirms the same
// non-divergent anchor stops gating the signal once RequireSMTDivergence is
// turned off.
func TestDecideSilverBullet_SMTOptionalWhenDisabled(t *testing.T) {
	candles := buildBullishSetup(baseStart)
	anchor := buildBullishAnchor(baseStart, "BTCUSDT", true)

	params := testParams()
	params.RequireSMTDivergence = false
	sig := DecideSilverBullet(candles, anchor, params, candles[len(candles)-1].Ts)
	if sig.Action != models.SignalBuy {
		t.Fatalf("expected BUY once SMT divergence isn't required, got %s (%s)", sig.Action, sig.Reason)
	}
}

// TestDecideSilverBullet_MissingAnchorDataFailsClosed verifies SMT
// confirmation - and therefore the whole signal - fails closed (HOLD, not a
// panic or a false BUY) when no anchor data is available at all.
func TestDecideSilverBullet_MissingAnchorDataFailsClosed(t *testing.T) {
	candles := buildBullishSetup(baseStart)

	sig := DecideSilverBullet(candles, nil, testParams(), candles[len(candles)-1].Ts)
	if sig.Action != models.SignalHold {
		t.Fatalf("expected HOLD with no anchor data, got %s (%s)", sig.Action, sig.Reason)
	}
}
