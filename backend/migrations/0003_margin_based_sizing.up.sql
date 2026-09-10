-- Sizing model changed from a fixed notional cap to a fixed per-order
-- margin amount, with notional = margin * leverage (scales with leverage
-- by design - no separate absolute notional ceiling, per explicit user
-- decision). 5.00 mirrors the user's stated $5 fixed margin.
ALTER TABLE settings RENAME COLUMN notional_cap_usd TO margin_usd;
ALTER TABLE settings ALTER COLUMN margin_usd SET DEFAULT 5.00;
UPDATE settings SET margin_usd = 5.00 WHERE id = 1;
