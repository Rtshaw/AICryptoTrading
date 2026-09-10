ALTER TABLE settings RENAME COLUMN margin_usd TO notional_cap_usd;
ALTER TABLE settings ALTER COLUMN notional_cap_usd SET DEFAULT 10.00;
UPDATE settings SET notional_cap_usd = 10.00 WHERE id = 1;
