-- The Silver Bullet setup (sweep + Fair Value Gap) that each AI signal
-- evaluated, so the dashboard chart can mark it even for signals loaded
-- from history rather than just ones received live over the websocket.
ALTER TABLE ai_signals ADD COLUMN sweep_ts    TIMESTAMPTZ;
ALTER TABLE ai_signals ADD COLUMN sweep_price NUMERIC(24,10);
ALTER TABLE ai_signals ADD COLUMN fvg_ts      TIMESTAMPTZ;
ALTER TABLE ai_signals ADD COLUMN fvg_low     NUMERIC(24,10);
ALTER TABLE ai_signals ADD COLUMN fvg_high    NUMERIC(24,10);
