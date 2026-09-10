-- The candle interval is now configurable via KLINE_INTERVAL (e.g. switched
-- from 15m to 5m), so the table name "candles_15m" would be misleading.
-- Renamed to the interval-agnostic "candles". Existing rows are truncated
-- rather than kept: mixing bars of different durations in the same
-- (symbol, ts) series would silently corrupt the indicator math (SMA/RSI/
-- MACD all assume uniform bar spacing) - safe to drop here since this is
-- just recent seed/history data, re-backfilled automatically on next startup.
ALTER TABLE candles_15m RENAME TO candles;
TRUNCATE candles;
