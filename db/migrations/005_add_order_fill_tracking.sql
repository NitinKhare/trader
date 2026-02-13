-- Migration 005: Add order fill tracking for accurate trade closure
--
-- Problem being solved:
-- 1. Exit prices didn't represent actual sales (just limit order prices)
-- 2. No way to verify if orders actually filled
-- 3. P&L was misleading (entry_price = exit_price = 0 P&L)
--
-- Solution:
-- - Track entry order status and actual fill price
-- - Track exit order status and actual fill price
-- - Only close trades when orders are FILLED (confirmed by broker)
-- - Use actual fill prices for accurate P&L calculation

-- ============================================================================
-- Add order fill tracking to trades table
-- ============================================================================

-- Entry order tracking
ALTER TABLE trades ADD COLUMN IF NOT EXISTS entry_order_status VARCHAR(20) DEFAULT 'PENDING';
-- PENDING = order placed, not filled yet
-- COMPLETED = order filled (trade is actually open)
-- REJECTED = order failed (trade never opened)
-- CANCELLED = order was cancelled

ALTER TABLE trades ADD COLUMN IF NOT EXISTS entry_fill_price NUMERIC(12,2);
-- Actual price the entry order filled at (may differ from entry_price if market moved)

ALTER TABLE trades ADD COLUMN IF NOT EXISTS entry_fill_time TIMESTAMPTZ;
-- When the entry order actually filled

-- Exit order tracking
ALTER TABLE trades ADD COLUMN IF NOT EXISTS exit_order_id VARCHAR(100);
-- The order ID of the exit order placed
-- (different from order_id which is entry order)

ALTER TABLE trades ADD COLUMN IF NOT EXISTS exit_order_status VARCHAR(20) DEFAULT 'PENDING';
-- PENDING = exit order placed, waiting for fill
-- COMPLETED = exit order filled (trade actually closed)
-- REJECTED = exit order failed (trade still open)
-- CANCELLED = exit order was cancelled
-- NULL = no exit order placed yet

ALTER TABLE trades ADD COLUMN IF NOT EXISTS exit_fill_price NUMERIC(12,2);
-- Actual price the exit order filled at
-- This is the REAL exit price (not the limit price we set)

ALTER TABLE trades ADD COLUMN IF NOT EXISTS exit_fill_time TIMESTAMPTZ;
-- When the exit order actually filled

-- Trade state tracking
ALTER TABLE trades ADD COLUMN IF NOT EXISTS position_state VARCHAR(20) DEFAULT 'ENTRY_PENDING';
-- ENTRY_PENDING = entry order placed, waiting for fill
-- ENTRY_FILLED = entry order filled, holding position, no exit signal yet
-- EXIT_PENDING = exit signal triggered, exit order placed, waiting for fill
-- EXIT_FILLED = exit order filled, trade closed
-- CANCELLED = trade was cancelled

-- ============================================================================
-- Update indexes for new columns
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_trades_entry_order_status ON trades (entry_order_status) WHERE entry_order_status != 'COMPLETED';
CREATE INDEX IF NOT EXISTS idx_trades_exit_order_status ON trades (exit_order_status) WHERE exit_order_status IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trades_position_state ON trades (position_state);
CREATE INDEX IF NOT EXISTS idx_trades_exit_order_id ON trades (exit_order_id) WHERE exit_order_id IS NOT NULL;

-- ============================================================================
-- Update existing closed trades to mark them with exit fill data
-- This handles trades closed before this migration
-- ============================================================================

-- For closed trades without exit fill data, mark them as exit filled with their exit price
UPDATE trades
SET exit_order_status = 'COMPLETED',
    exit_fill_price = exit_price,
    position_state = 'EXIT_FILLED'
WHERE status = 'closed' AND exit_order_status IS NULL;

-- For open trades, mark entry as completed (they're holding positions)
UPDATE trades
SET entry_order_status = 'COMPLETED',
    entry_fill_price = entry_price,
    position_state = 'ENTRY_FILLED'
WHERE status = 'open' AND entry_order_status IS NULL;

-- ============================================================================
-- Helper views for monitoring order fill status
-- ============================================================================

-- Trades waiting for entry to fill
CREATE OR REPLACE VIEW pending_entry_orders AS
SELECT
  id, symbol, entry_price, quantity, entry_time, order_id,
  'Waiting for entry order to fill' as status
FROM trades
WHERE entry_order_status = 'PENDING' AND position_state = 'ENTRY_PENDING'
ORDER BY entry_time ASC;

-- Trades with entry filled, waiting for exit
CREATE OR REPLACE VIEW open_positions_awaiting_exit AS
SELECT
  id, symbol, entry_fill_price, quantity, entry_fill_time, order_id,
  stop_loss, target,
  'Entry filled, monitoring for exit signals' as status
FROM trades
WHERE position_state = 'ENTRY_FILLED'
ORDER BY entry_fill_time ASC;

-- Trades with exit order pending
CREATE OR REPLACE VIEW pending_exit_orders AS
SELECT
  id, symbol, exit_order_id, exit_price, quantity, entry_fill_price,
  'Waiting for exit order to fill' as status
FROM trades
WHERE exit_order_status = 'PENDING' AND position_state = 'EXIT_PENDING'
ORDER BY exit_time ASC;

-- Trades that are fully closed
CREATE OR REPLACE VIEW closed_trades AS
SELECT
  id, symbol, entry_fill_price, exit_fill_price, quantity,
  (exit_fill_price - entry_fill_price) * quantity as pnl,
  entry_fill_time, exit_fill_time,
  'Trade completed' as status
FROM trades
WHERE position_state = 'EXIT_FILLED' AND status = 'closed'
ORDER BY exit_fill_time DESC;
