package config

import (
	"os"
	"path/filepath"
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
		"paths": {"ai_output_dir": "./ai_outputs"},
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
