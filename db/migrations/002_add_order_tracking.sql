-- Migration 002: Add order tracking columns to trades table.
-- Enables tracking both entry order IDs and stop-loss order IDs per trade.

ALTER TABLE trades ADD COLUMN IF NOT EXISTS order_id VARCHAR(50);
ALTER TABLE trades ADD COLUMN IF NOT EXISTS sl_order_id VARCHAR(50);

-- Partial indexes for fast order ID lookups (used by webhook postback matching).
CREATE INDEX IF NOT EXISTS idx_trades_order_id ON trades (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trades_sl_order_id ON trades (sl_order_id) WHERE sl_order_id IS NOT NULL;
