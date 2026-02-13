-- Migration 003: Expand order_id and sl_order_id column sizes.
-- Paper broker order IDs can get long after many trades (PAPER-1, PAPER-2, etc.)
-- Dhan API order IDs are also variable length strings.
-- Expanding from VARCHAR(50) to VARCHAR(100) provides ample room.

ALTER TABLE trades ALTER COLUMN order_id TYPE VARCHAR(100);
ALTER TABLE trades ALTER COLUMN sl_order_id TYPE VARCHAR(100);

-- Drop and recreate indexes with new column type
DROP INDEX IF EXISTS idx_trades_order_id;
DROP INDEX IF EXISTS idx_trades_sl_order_id;

CREATE INDEX idx_trades_order_id ON trades (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX idx_trades_sl_order_id ON trades (sl_order_id) WHERE sl_order_id IS NOT NULL;
