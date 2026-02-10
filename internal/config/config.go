// Package config provides application-wide configuration management.
// All configuration is loaded from files and environment variables.
// No configuration is hardcoded in strategy or broker logic.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Mode defines whether the system runs in paper or live trading mode.
type Mode string

const (
	ModePaper Mode = "paper"
	ModeLive  Mode = "live"
)

// Config holds all system configuration.
// Loaded once at startup and passed as read-only to all components.
type Config struct {
	// ActiveBroker selects which broker implementation to use (e.g. "dhan").
	ActiveBroker string `json:"active_broker"`

	// TradingMode controls whether orders are actually placed (live) or simulated (paper).
	TradingMode Mode `json:"trading_mode"`

	// Capital is the total capital available for trading (INR).
	Capital float64 `json:"capital"`

	// Risk configuration limits.
	Risk RiskConfig `json:"risk"`

	// Paths for file-based communication with Python AI layer.
	Paths PathsConfig `json:"paths"`

	// Broker-specific configuration (API keys, endpoints, etc.).
	BrokerConfig map[string]json.RawMessage `json:"broker_config"`

	// Database connection string.
	DatabaseURL string `json:"database_url"`

	// MarketCalendarPath points to the exchange calendar data file.
	MarketCalendarPath string `json:"market_calendar_path"`

	// Webhook server configuration for receiving broker postback notifications.
	Webhook WebhookConfig `json:"webhook"`
}

// WebhookConfig holds settings for the order postback HTTP server.
type WebhookConfig struct {
	// Enabled controls whether the webhook server starts.
	Enabled bool `json:"enabled"`

	// Port is the HTTP port the webhook server listens on.
	Port int `json:"port"`

	// Path is the URL path for the postback endpoint (default: /webhook/dhan/order).
	Path string `json:"path"`
}

// RiskConfig defines hard risk guardrails.
// These limits are enforced by the risk module and cannot be overridden by strategies or AI.
type RiskConfig struct {
	// MaxRiskPerTradePct is the maximum percentage of capital risked on a single trade.
	MaxRiskPerTradePct float64 `json:"max_risk_per_trade_pct"`

	// MaxOpenPositions limits concurrent open positions.
	MaxOpenPositions int `json:"max_open_positions"`

	// MaxDailyLossPct is the maximum daily loss as a percentage of capital.
	MaxDailyLossPct float64 `json:"max_daily_loss_pct"`

	// MaxCapitalDeploymentPct limits how much total capital can be deployed at once.
	MaxCapitalDeploymentPct float64 `json:"max_capital_deployment_pct"`
}

// PathsConfig defines filesystem paths for inter-layer communication.
type PathsConfig struct {
	// AIOutputDir is where Python writes scoring outputs (JSON/Parquet).
	AIOutputDir string `json:"ai_output_dir"`

	// MarketDataDir is where cached market data lives.
	MarketDataDir string `json:"market_data_dir"`

	// LogDir is where all system logs are written.
	LogDir string `json:"log_dir"`
}

// Load reads configuration from a JSON file.
// Environment variables override file values where applicable.
func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("config: resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("config: read file %s: %w", absPath, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse json: %w", err)
	}

	// Environment variable overrides.
	if v := os.Getenv("ALGO_TRADING_MODE"); v != "" {
		cfg.TradingMode = Mode(v)
	}
	if v := os.Getenv("ALGO_DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("ALGO_ACTIVE_BROKER"); v != "" {
		cfg.ActiveBroker = v
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required configuration fields are present and sane.
func (c *Config) Validate() error {
	if c.ActiveBroker == "" {
		return fmt.Errorf("active_broker is required")
	}
	if c.TradingMode != ModePaper && c.TradingMode != ModeLive {
		return fmt.Errorf("trading_mode must be 'paper' or 'live', got %q", c.TradingMode)
	}
	if c.Capital <= 0 {
		return fmt.Errorf("capital must be positive, got %f", c.Capital)
	}
	if c.Risk.MaxRiskPerTradePct <= 0 || c.Risk.MaxRiskPerTradePct > 100 {
		return fmt.Errorf("max_risk_per_trade_pct must be in (0, 100], got %f", c.Risk.MaxRiskPerTradePct)
	}
	if c.Risk.MaxOpenPositions <= 0 {
		return fmt.Errorf("max_open_positions must be positive, got %d", c.Risk.MaxOpenPositions)
	}
	if c.Risk.MaxDailyLossPct <= 0 || c.Risk.MaxDailyLossPct > 100 {
		return fmt.Errorf("max_daily_loss_pct must be in (0, 100], got %f", c.Risk.MaxDailyLossPct)
	}
	if c.Risk.MaxCapitalDeploymentPct <= 0 || c.Risk.MaxCapitalDeploymentPct > 100 {
		return fmt.Errorf("max_capital_deployment_pct must be in (0, 100], got %f", c.Risk.MaxCapitalDeploymentPct)
	}
	if c.Paths.AIOutputDir == "" {
		return fmt.Errorf("paths.ai_output_dir is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}

	// Live mode has stricter requirements to prevent accidental real trading.
	if c.TradingMode == ModeLive {
		if err := c.validateLiveMode(); err != nil {
			return fmt.Errorf("live mode: %w", err)
		}
	}

	return nil
}

// validateLiveMode enforces extra safety checks when running with real money.
func (c *Config) validateLiveMode() error {
	// Broker config must exist for the active broker.
	if c.BrokerConfig == nil {
		return fmt.Errorf("broker_config is required for live trading")
	}
	if _, ok := c.BrokerConfig[c.ActiveBroker]; !ok {
		return fmt.Errorf("broker_config[%q] is required for live trading", c.ActiveBroker)
	}

	// Safety cap: max 5 open positions in live mode.
	if c.Risk.MaxOpenPositions > 5 {
		return fmt.Errorf("max_open_positions cannot exceed 5 in live mode (got %d)", c.Risk.MaxOpenPositions)
	}

	// Safety cap: max 2%% risk per trade in live mode.
	if c.Risk.MaxRiskPerTradePct > 2.0 {
		return fmt.Errorf("max_risk_per_trade_pct cannot exceed 2%% in live mode (got %.1f%%)", c.Risk.MaxRiskPerTradePct)
	}

	// Safety cap: max 70%% capital deployment in live mode.
	if c.Risk.MaxCapitalDeploymentPct > 70.0 {
		return fmt.Errorf("max_capital_deployment_pct cannot exceed 70%% in live mode (got %.1f%%)", c.Risk.MaxCapitalDeploymentPct)
	}

	return nil
}
