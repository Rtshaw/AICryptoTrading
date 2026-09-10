-- AI daily watchlist selection history (internal/watchlistai), mirroring
-- the sibling TWSEDailyTrading project's ai_watchlist_runs/ai_watchlist_picks
-- tables: each run records which candidates were considered and which the
-- AI picked (with rationale), so past selections stay auditable even after
-- the watchlist table itself moves on to a newer run's picks.
CREATE TABLE ai_watchlist_runs (
    id              SERIAL PRIMARY KEY,
    run_date        DATE NOT NULL UNIQUE,
    candidate_count INT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_watchlist_picks (
    id                    SERIAL PRIMARY KEY,
    run_id                INT NOT NULL REFERENCES ai_watchlist_runs(id) ON DELETE CASCADE,
    rank                  INT NOT NULL,
    symbol                TEXT NOT NULL,
    rationale             TEXT NOT NULL,
    price                 NUMERIC(24,10),
    price_change_pct_24h  NUMERIC(10,4),
    quote_volume_24h      NUMERIC(24,4)
);
CREATE INDEX idx_ai_watchlist_picks_run ON ai_watchlist_picks(run_id, rank);
