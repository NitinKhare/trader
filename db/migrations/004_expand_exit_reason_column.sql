-- Migration 004: Expand exit_reason column size.
-- Exit reason strings from strategies can be longer than VARCHAR(30).
-- Examples: "mean reversion target reached", "price crossed above SMA(20)"
-- Expanding from VARCHAR(30) to VARCHAR(200) provides ample room.

ALTER TABLE trades ALTER COLUMN exit_reason TYPE VARCHAR(200);
