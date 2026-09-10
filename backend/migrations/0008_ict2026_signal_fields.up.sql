-- The ICT-2026 upgrade's additional confluence fields (Optimal Trade Entry
-- zone, Breaker Block/Unicorn Model zone, SMT divergence) that each AI
-- signal evaluated - same reasoning as migration 0006's sweep/FVG columns:
-- lets the dashboard chart mark them even for signals loaded from history.
ALTER TABLE ai_signals ADD COLUMN ote_low            NUMERIC(24,10);
ALTER TABLE ai_signals ADD COLUMN ote_high           NUMERIC(24,10);
ALTER TABLE ai_signals ADD COLUMN breaker_low        NUMERIC(24,10);
ALTER TABLE ai_signals ADD COLUMN breaker_high       NUMERIC(24,10);
ALTER TABLE ai_signals ADD COLUMN smt_anchor_symbol  TEXT;
ALTER TABLE ai_signals ADD COLUMN smt_confirmed      BOOLEAN;
