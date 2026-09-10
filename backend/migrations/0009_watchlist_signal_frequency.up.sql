-- How many Silver Bullet trades the currently-live ICT-2026 rules actually
-- produced for this pick over the last internal/watchlistai.
-- SignalScanLookbackDays (see AnnotateSignalFrequency) - the primary
-- ranking signal the AI daily selection now uses, kept here for audit/
-- display. DEFAULT 0 backfills existing rows (picked before this column
-- existed) without needing a NULL-tolerant scan.
ALTER TABLE ai_watchlist_picks ADD COLUMN recent_signal_count INT NOT NULL DEFAULT 0;
