package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func watcherLogger() *log.Logger {
	return log.New(os.Stdout, "[watcher-test] ", log.LstdFlags)
}

func writeWatcherTestConfig(t *testing.T, path string, cfg *Config) {
	t.Helper()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func baseTestConfig() *Config {
	return &Config{
		ActiveBroker: "dhan",
		TradingMode:  ModePaper,
		Capital:      500000,
		Risk: RiskConfig{
			MaxRiskPerTradePct:      1.0,
			MaxOpenPositions:        5,
			MaxDailyLossPct:         3.0,
			MaxCapitalDeploymentPct: 80.0,
		},
		Paths: PathsConfig{
			AIOutputDir: "./ai_outputs",
		},
		DatabaseURL: "postgres://test@localhost/test?sslmode=disable",
	}
}

func TestConfigWatcher_DetectsChange(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	initial := baseTestConfig()
	writeWatcherTestConfig(t, cfgPath, initial)

	watcher := NewConfigWatcher(cfgPath, initial, watcherLogger())

	changed := make(chan bool, 1)
	watcher.OnChange(func(old, new *Config) {
		changed <- true
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer watcher.Stop()

	// Wait a moment then modify the file.
	time.Sleep(100 * time.Millisecond)
	updated := baseTestConfig()
	updated.Risk.MaxOpenPositions = 3 // change risk param
	writeWatcherTestConfig(t, cfgPath, updated)

	// Manually trigger check instead of waiting for poll interval.
	watcher.checkForChanges()

	select {
	case <-changed:
		// Success — change was detected.
		current := watcher.Current()
		if current.Risk.MaxOpenPositions != 3 {
			t.Errorf("expected MaxOpenPositions=3, got %d", current.Risk.MaxOpenPositions)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for config change notification")
	}
}

func TestConfigWatcher_IgnoresInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	initial := baseTestConfig()
	writeWatcherTestConfig(t, cfgPath, initial)

	watcher := NewConfigWatcher(cfgPath, initial, watcherLogger())

	changed := make(chan bool, 1)
	watcher.OnChange(func(old, new *Config) {
		changed <- true
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer watcher.Stop()

	// Write invalid JSON.
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(cfgPath, []byte("not valid json"), 0644)
	watcher.checkForChanges()

	select {
	case <-changed:
		t.Error("should NOT fire callback for invalid JSON")
	case <-time.After(100 * time.Millisecond):
		// Good — invalid config was ignored.
	}

	// Config should still be the original.
	current := watcher.Current()
	if current.Risk.MaxOpenPositions != 5 {
		t.Errorf("expected original MaxOpenPositions=5, got %d", current.Risk.MaxOpenPositions)
	}
}

func TestConfigWatcher_IgnoresNonRiskChanges(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	initial := baseTestConfig()
	writeWatcherTestConfig(t, cfgPath, initial)

	watcher := NewConfigWatcher(cfgPath, initial, watcherLogger())

	changed := make(chan bool, 1)
	watcher.OnChange(func(old, new *Config) {
		changed <- true
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer watcher.Stop()

	// Change only non-risk fields.
	time.Sleep(100 * time.Millisecond)
	updated := baseTestConfig()
	updated.Capital = 1000000 // non-risk field
	writeWatcherTestConfig(t, cfgPath, updated)
	watcher.checkForChanges()

	select {
	case <-changed:
		t.Error("should NOT fire callback for non-risk changes")
	case <-time.After(100 * time.Millisecond):
		// Good.
	}
}

func TestConfigWatcher_IgnoresValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	initial := baseTestConfig()
	writeWatcherTestConfig(t, cfgPath, initial)

	watcher := NewConfigWatcher(cfgPath, initial, watcherLogger())

	changed := make(chan bool, 1)
	watcher.OnChange(func(old, new *Config) {
		changed <- true
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer watcher.Stop()

	// Write config that fails validation (max_open_positions = 0).
	time.Sleep(100 * time.Millisecond)
	updated := baseTestConfig()
	updated.Risk.MaxOpenPositions = 0 // invalid
	writeWatcherTestConfig(t, cfgPath, updated)
	watcher.checkForChanges()

	select {
	case <-changed:
		t.Error("should NOT fire callback for invalid config")
	case <-time.After(100 * time.Millisecond):
		// Good.
	}
}

func TestRiskConfigChanged(t *testing.T) {
	base := RiskConfig{
		MaxRiskPerTradePct:      1.0,
		MaxOpenPositions:        5,
		MaxDailyLossPct:         3.0,
		MaxCapitalDeploymentPct: 80.0,
	}

	// Same config.
	if riskConfigChanged(base, base) {
		t.Error("identical configs should not be flagged as changed")
	}

	// Change one field.
	modified := base
	modified.MaxOpenPositions = 3
	if !riskConfigChanged(base, modified) {
		t.Error("should detect MaxOpenPositions change")
	}

	// Change trailing stop.
	modified2 := base
	modified2.TrailingStop.Enabled = true
	if !riskConfigChanged(base, modified2) {
		t.Error("should detect TrailingStop change")
	}

	// Change circuit breaker.
	modified3 := base
	modified3.CircuitBreaker.MaxConsecutiveFailures = 5
	if !riskConfigChanged(base, modified3) {
		t.Error("should detect CircuitBreaker change")
	}
}

func TestConfigWatcher_StopIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	writeWatcherTestConfig(t, cfgPath, baseTestConfig())

	watcher := NewConfigWatcher(cfgPath, baseTestConfig(), watcherLogger())
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Should not panic when called multiple times.
	watcher.Stop()
	watcher.Stop()
	watcher.Stop()
}
