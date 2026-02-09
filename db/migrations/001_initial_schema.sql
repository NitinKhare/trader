-- Initial database schema for algoTradingAgent.
-- Uses Postgres with TimescaleDB extension.
--
-- Tables:
--   candles       - OHLCV market data (hypertable)
--   signals       - Strategy-generated signals
--   trades        - Trade records (open and closed)
--   positions     - Current position state
--   ai_scores     - AI scoring outputs
--   trade_logs    - Detailed audit trail
--   market_regime - Daily market regime snapshots

-- Enable TimescaleDB extension.
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ============================================================================
-- CANDLES: Daily OHLCV data.
-- This is a TimescaleDB hypertable partitioned by date for efficient range queries.
-- ============================================================================
CREATE TABLE IF NOT EXISTS candles (
    symbol      VARCHAR(20)    NOT NULL,
    date        DATE           NOT NULL,
    open        NUMERIC(12,2)  NOT NULL,
    high        NUMERIC(12,2)  NOT NULL,
    low         NUMERIC(12,2)  NOT NULL,
    close       NUMERIC(12,2)  NOT NULL,
    volume      BIGINT         NOT NULL,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    PRIMARY KEY (symbol, date)
);

-- Convert to TimescaleDB hypertable.
SELECT create_hypertable('candles', 'date', if_not_exists => TRUE);

-- Index for fast symbol lookups.
CREATE INDEX IF NOT EXISTS idx_candles_symbol ON candles (symbol, date DESC);

-- ============================================================================
-- SIGNALS: Strategy-generated trade signals.
-- Every signal is recorded regardless of whether it was approved by risk management.
-- ============================================================================
CREATE TABLE IF NOT EXISTS signals (
    id              BIGSERIAL      PRIMARY KEY,
    strategy_id     VARCHAR(50)    NOT NULL,
    signal_id       VARCHAR(100)   NOT NULL UNIQUE,
    symbol          VARCHAR(20)    NOT NULL,
    action          VARCHAR(10)    NOT NULL,  -- BUY, HOLD, EXIT, SKIP
    price           NUMERIC(12,2),
    stop_loss       NUMERIC(12,2),
    target          NUMERIC(12,2),
    quantity        INT,
    reason          TEXT           NOT NULL,
    -- AI score snapshot at signal time.
    trend_strength_score    NUMERIC(6,4),
    breakout_quality_score  NUMERIC(6,4),
    volatility_score        NUMERIC(6,4),
    risk_score              NUMERIC(6,4),
    liquidity_score         NUMERIC(6,4),
    -- Risk management outcome.
    approved                BOOLEAN        NOT NULL DEFAULT FALSE,
    rejection_reason        TEXT,
    signal_date             DATE           NOT NULL,
    created_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_signals_date ON signals (signal_date DESC);
CREATE INDEX IF NOT EXISTS idx_signals_strategy ON signals (strategy_id, signal_date DESC);

-- ============================================================================
-- TRADES: Complete trade records.
-- Tracks the full lifecycle from entry to exit.
-- ============================================================================
CREATE TABLE IF NOT EXISTS trades (
    id              BIGSERIAL      PRIMARY KEY,
    strategy_id     VARCHAR(50)    NOT NULL,
    signal_id       VARCHAR(100)   NOT NULL,
    symbol          VARCHAR(20)    NOT NULL,
    side            VARCHAR(10)    NOT NULL,  -- BUY or SELL
    quantity        INT            NOT NULL,
    entry_price     NUMERIC(12,2)  NOT NULL,
    exit_price      NUMERIC(12,2),
    stop_loss       NUMERIC(12,2)  NOT NULL,
    target          NUMERIC(12,2),
    entry_time      TIMESTAMPTZ    NOT NULL,
    exit_time       TIMESTAMPTZ,
    exit_reason     VARCHAR(30),  -- stop_loss, target, time_exit, strategy_invalidation, manual
    pnl             NUMERIC(14,2),
    status          VARCHAR(10)    NOT NULL DEFAULT 'open',  -- open, closed
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_status ON trades (status);
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades (symbol, status);
CREATE INDEX IF NOT EXISTS idx_trades_strategy ON trades (strategy_id, status);

-- ============================================================================
-- AI_SCORES: Daily AI scoring outputs for audit trail.
-- ============================================================================
CREATE TABLE IF NOT EXISTS ai_scores (
    id                      BIGSERIAL      PRIMARY KEY,
    symbol                  VARCHAR(20)    NOT NULL,
    score_date              DATE           NOT NULL,
    trend_strength_score    NUMERIC(6,4)   NOT NULL,
    breakout_quality_score  NUMERIC(6,4)   NOT NULL,
    volatility_score        NUMERIC(6,4)   NOT NULL,
    risk_score              NUMERIC(6,4)   NOT NULL,
    liquidity_score         NUMERIC(6,4)   NOT NULL,
    composite_score         NUMERIC(6,4),
    rank                    INT,
    created_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    UNIQUE (symbol, score_date)
);

CREATE INDEX IF NOT EXISTS idx_ai_scores_date ON ai_scores (score_date DESC);

-- ============================================================================
-- MARKET_REGIME: Daily market regime snapshots.
-- ============================================================================
CREATE TABLE IF NOT EXISTS market_regime (
    id          BIGSERIAL      PRIMARY KEY,
    regime_date DATE           NOT NULL UNIQUE,
    regime      VARCHAR(10)    NOT NULL,  -- BULL, SIDEWAYS, BEAR
    confidence  NUMERIC(5,4)   NOT NULL,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- TRADE_LOGS: Detailed audit trail.
-- Every action the system takes is logged here.
-- ============================================================================
CREATE TABLE IF NOT EXISTS trade_logs (
    id          BIGSERIAL      PRIMARY KEY,
    timestamp   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    strategy_id VARCHAR(50),
    symbol      VARCHAR(20),
    action      VARCHAR(20)    NOT NULL,
    reason_code VARCHAR(50)    NOT NULL,
    message     TEXT           NOT NULL,
    inputs_json JSONB,         -- Snapshot of inputs for this decision.
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trade_logs_time ON trade_logs (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trade_logs_strategy ON trade_logs (strategy_id, timestamp DESC);
