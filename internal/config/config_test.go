package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return path
}

func TestConfig_LoadValid(t *testing.T) {
	path := writeTestConfig(t, `{
		"active_broker": "dhan",
		"trading_mode": "paper",
		"capital": 500000,
		"risk": {
			"max_risk_per_trade_pct": 1.0,
			"max_open_positions": 5,
			"max_daily_loss_pct": 3.0,
			"max_capital_deployment_pct": 80.0
		},
		"paths": {
			"ai_output_dir": "./ai_outputs",
			"market_data_dir": "./market_data",
			"log_dir": "./logs"
		},
		"broker_config": {},
		"database_url": "postgres://localhost/test",
		"market_calendar_path": "./holidays.json"
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ActiveBroker != "dhan" {
		t.Errorf("expected dhan, got %s", cfg.ActiveBroker)
	}
	if cfg.TradingMode != ModePaper {
		t.Errorf("expected paper, got %s", cfg.TradingMode)
	}
	if cfg.Capital != 500000 {
		t.Errorf("expected 500000, got %f", cfg.Capital)
	}
}

func TestConfig_RejectsInvalidMode(t *testing.T) {
	path := writeTestConfig(t, `{
		"active_broker": "dhan",
		"trading_mode": "invalid",
		"capital": 500000,
		"risk": {
			"max_risk_per_trade_pct": 1.0,
			"max_open_positions": 5,
			"max_daily_loss_pct": 3.0,
			"max_capital_deployment_pct": 80.0
		},
		"paths": {"ai_output_dir": "./ai_outputs"},
		"database_url": "postgres://localhost/test"
	}`)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid trading mode")
	}
}

func TestConfig_RejectsZeroCapital(t *testing.T) {
	path := writeTestConfig(t, `{
		"active_broker": "dhan",
		"trading_mode": "paper",
		"capital": 0,
		"risk": {
			"max_risk_per_trade_pct": 1.0,
			"max_open_positions": 5,
			"max_daily_loss_pct": 3.0,
			"max_capital_deployment_pct": 80.0
		},
		"paths": {"ai_output_dir": "./ai_outputs"},
		"database_url": "postgres://localhost/test"
	}`)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for zero capital")
	}
}

func TestConfig_EnvOverride(t *testing.T) {
	// Live mode env override with valid live mode config
	// (broker_config with "dhan" key, safe risk limits).
	path := writeTestConfig(t, `{
		"active_broker": "dhan",
		"trading_mode": "paper",
		"capital": 500000,
		"risk": {
			"max_risk_per_trade_pct": 1.0,
			"max_open_positions": 5,
			"max_daily_loss_pct": 3.0,
			"max_capital_deployment_pct": 70.0
		},
		"paths": {"ai_output_dir": "./ai_outputs"},
		"broker_config": {"dhan": {"api_key": "test", "secret": "test"}},
		"database_url": "postgres://localhost/test"
	}`)

	os.Setenv("ALGO_TRADING_MODE", "live")
	defer os.Unsetenv("ALGO_TRADING_MODE")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TradingMode != ModeLive {
		t.Errorf("expected env override to live, got %s", cfg.TradingMode)
	}
}

// ────────────────────────────────────────────────────────────────────
// Live mode validation tests
// ────────────────────────────────────────────────────────────────────

// validLiveConfig returns a Config that passes all live mode validations.
func validLiveConfig() Config {
	return Config{
		ActiveBroker: "dhan",
		TradingMode:  ModeLive,
		Capital:      500000,
		Risk: RiskConfig{
			MaxRiskPerTradePct:      1.0,
			MaxOpenPositions:        5,
			MaxDailyLossPct:         3.0,
			MaxCapitalDeploymentPct: 70.0,
		},
		Paths: PathsConfig{
			AIOutputDir: "./ai_outputs",
		},
		BrokerConfig: map[string]json.RawMessage{
			"dhan": json.RawMessage(`{"api_key":"test","secret":"test"}`),
		},
		DatabaseURL: "postgres://localhost/test",
	}
}

func TestLiveMode_RequiresBrokerConfig(t *testing.T) {
	cfg := validLiveConfig()
	cfg.BrokerConfig = nil // Remove broker config

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when broker_config is nil in live mode")
	}
	if !strings.Contains(err.Error(), "broker_config") {
		t.Errorf("error should mention broker_config, got: %v", err)
	}
}

func TestLiveMode_RequiresActiveBrokerInConfig(t *testing.T) {
	cfg := validLiveConfig()
	cfg.BrokerConfig = map[string]json.RawMessage{
		"other_broker": json.RawMessage(`{}`),
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when active broker not in broker_config")
	}
	if !strings.Contains(err.Error(), "dhan") {
		t.Errorf("error should mention active broker name, got: %v", err)
	}
}

func TestLiveMode_MaxPositionsCap(t *testing.T) {
	cfg := validLiveConfig()
	cfg.Risk.MaxOpenPositions = 10 // Exceeds live mode cap of 5

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when max_open_positions > 5 in live mode")
	}
	if !strings.Contains(err.Error(), "max_open_positions") {
		t.Errorf("error should mention max_open_positions, got: %v", err)
	}
}

func TestLiveMode_MaxRiskPerTradeCap(t *testing.T) {
	cfg := validLiveConfig()
	cfg.Risk.MaxRiskPerTradePct = 5.0 // Exceeds live mode cap of 2%

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when max_risk_per_trade_pct > 2 in live mode")
	}
	if !strings.Contains(err.Error(), "max_risk_per_trade_pct") {
		t.Errorf("error should mention max_risk_per_trade_pct, got: %v", err)
	}
}

func TestLiveMode_MaxCapitalDeploymentCap(t *testing.T) {
	cfg := validLiveConfig()
	cfg.Risk.MaxCapitalDeploymentPct = 90.0 // Exceeds live mode cap of 70%

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when max_capital_deployment_pct > 70 in live mode")
	}
	if !strings.Contains(err.Error(), "max_capital_deployment_pct") {
		t.Errorf("error should mention max_capital_deployment_pct, got: %v", err)
	}
}

func TestLiveMode_RequiresDatabaseURL(t *testing.T) {
	cfg := validLiveConfig()
	cfg.DatabaseURL = "" // Remove DB URL

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when database_url is empty")
	}
	if !strings.Contains(err.Error(), "database_url") {
		t.Errorf("error should mention database_url, got: %v", err)
	}
}

func TestLiveMode_ValidConfigPasses(t *testing.T) {
	cfg := validLiveConfig()
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("valid live config should pass validation, got: %v", err)
	}
}

func TestPaperMode_SkipsLiveChecks(t *testing.T) {
	// Paper mode should NOT enforce live mode restrictions.
	cfg := Config{
		ActiveBroker: "dhan",
		TradingMode:  ModePaper,
		Capital:      500000,
		Risk: RiskConfig{
			MaxRiskPerTradePct:      5.0, // Would fail live mode, but fine for paper
			MaxOpenPositions:        10,  // Would fail live mode, but fine for paper
			MaxDailyLossPct:         10.0,
			MaxCapitalDeploymentPct: 100.0, // Would fail live mode, but fine for paper
		},
		Paths: PathsConfig{
			AIOutputDir: "./ai_outputs",
		},
		DatabaseURL: "postgres://localhost/test",
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("paper mode should not enforce live mode caps, got: %v", err)
	}
}
