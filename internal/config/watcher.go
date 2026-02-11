// Package config - watcher.go provides config file hot-reload support.
//
// The watcher polls the config file for changes (stat-based, every 5 seconds)
// and notifies registered callbacks when risk parameters change.
//
// Only risk configuration is reloadable. Broker config, database URL,
// trading mode, and other structural settings require an engine restart.
package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// ConfigWatcher monitors the config file for changes and invokes callbacks
// when risk-related fields change. It uses stat-based polling (no external
// dependencies like fsnotify required).
type ConfigWatcher struct {
	path      string
	logger    *log.Logger
	mu        sync.RWMutex
	current   *Config
	lastMod   time.Time
	onChange  []func(old, new *Config)
	done      chan struct{}
	stopped   bool
}

// NewConfigWatcher creates a watcher for the given config file path.
// initial is the currently loaded config. The watcher does not start
// until Start() is called.
func NewConfigWatcher(path string, initial *Config, logger *log.Logger) *ConfigWatcher {
	if logger == nil {
		logger = log.New(log.Writer(), "", log.LstdFlags)
	}
	return &ConfigWatcher{
		path:    path,
		logger:  logger,
		current: initial,
		done:    make(chan struct{}),
	}
}

// OnChange registers a callback that will be called when the config file
// changes and the new config passes validation. Multiple callbacks may
// be registered. Callbacks receive the old and new config values.
//
// Only risk config changes trigger callbacks. Changes to broker config,
// database URL, or trading mode are ignored (they require a restart).
func (w *ConfigWatcher) OnChange(fn func(old, new *Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onChange = append(w.onChange, fn)
}

// Start begins polling the config file for changes. It returns immediately;
// the watcher runs in a background goroutine. Returns an error if the
// initial file stat fails.
func (w *ConfigWatcher) Start() error {
	info, err := os.Stat(w.path)
	if err != nil {
		return err
	}
	w.lastMod = info.ModTime()
	w.logger.Printf("[config-watcher] watching %s for changes (poll interval: 5s)", w.path)

	go w.pollLoop()
	return nil
}

// Stop stops the config watcher. Safe to call multiple times.
func (w *ConfigWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
		w.logger.Println("[config-watcher] stopped")
	}
}

// Current returns the most recently loaded valid config.
func (w *ConfigWatcher) Current() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.current
}

// ────────────────────────────────────────────────────────────────────
// Internal
// ────────────────────────────────────────────────────────────────────

func (w *ConfigWatcher) pollLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.checkForChanges()
		}
	}
}

func (w *ConfigWatcher) checkForChanges() {
	info, err := os.Stat(w.path)
	if err != nil {
		w.logger.Printf("[config-watcher] stat error: %v", err)
		return
	}

	if !info.ModTime().After(w.lastMod) {
		return // file hasn't changed
	}
	w.lastMod = info.ModTime()

	// Read and parse new config.
	data, err := os.ReadFile(w.path)
	if err != nil {
		w.logger.Printf("[config-watcher] read error: %v", err)
		return
	}

	var newCfg Config
	if err := json.Unmarshal(data, &newCfg); err != nil {
		w.logger.Printf("[config-watcher] parse error (keeping old config): %v", err)
		return
	}

	// Validate the new config.
	if err := newCfg.Validate(); err != nil {
		w.logger.Printf("[config-watcher] validation error (keeping old config): %v", err)
		return
	}

	// Check if risk-related fields actually changed.
	w.mu.RLock()
	oldCfg := w.current
	w.mu.RUnlock()

	if !riskConfigChanged(oldCfg.Risk, newCfg.Risk) {
		w.logger.Printf("[config-watcher] file changed but risk config unchanged, skipping")
		return
	}

	// Log what changed.
	w.logRiskChanges(oldCfg.Risk, newCfg.Risk)

	// Apply the new config and notify callbacks.
	w.mu.Lock()
	w.current = &newCfg
	callbacks := make([]func(old, new *Config), len(w.onChange))
	copy(callbacks, w.onChange)
	w.mu.Unlock()

	for _, fn := range callbacks {
		fn(oldCfg, &newCfg)
	}
}

// riskConfigChanged returns true if any risk-related field changed.
func riskConfigChanged(old, new RiskConfig) bool {
	if old.MaxRiskPerTradePct != new.MaxRiskPerTradePct {
		return true
	}
	if old.MaxOpenPositions != new.MaxOpenPositions {
		return true
	}
	if old.MaxDailyLossPct != new.MaxDailyLossPct {
		return true
	}
	if old.MaxCapitalDeploymentPct != new.MaxCapitalDeploymentPct {
		return true
	}
	if old.MaxPerSector != new.MaxPerSector {
		return true
	}
	if old.MaxHoldDays != new.MaxHoldDays {
		return true
	}
	if old.TrailingStop.Enabled != new.TrailingStop.Enabled ||
		old.TrailingStop.TrailPct != new.TrailingStop.TrailPct ||
		old.TrailingStop.ActivationPct != new.TrailingStop.ActivationPct {
		return true
	}
	if old.CircuitBreaker.MaxConsecutiveFailures != new.CircuitBreaker.MaxConsecutiveFailures ||
		old.CircuitBreaker.MaxFailuresPerHour != new.CircuitBreaker.MaxFailuresPerHour ||
		old.CircuitBreaker.CooldownMinutes != new.CircuitBreaker.CooldownMinutes {
		return true
	}
	return false
}

func (w *ConfigWatcher) logRiskChanges(old, new RiskConfig) {
	if old.MaxRiskPerTradePct != new.MaxRiskPerTradePct {
		w.logger.Printf("[config-watcher] max_risk_per_trade_pct: %.2f -> %.2f", old.MaxRiskPerTradePct, new.MaxRiskPerTradePct)
	}
	if old.MaxOpenPositions != new.MaxOpenPositions {
		w.logger.Printf("[config-watcher] max_open_positions: %d -> %d", old.MaxOpenPositions, new.MaxOpenPositions)
	}
	if old.MaxDailyLossPct != new.MaxDailyLossPct {
		w.logger.Printf("[config-watcher] max_daily_loss_pct: %.2f -> %.2f", old.MaxDailyLossPct, new.MaxDailyLossPct)
	}
	if old.MaxCapitalDeploymentPct != new.MaxCapitalDeploymentPct {
		w.logger.Printf("[config-watcher] max_capital_deployment_pct: %.2f -> %.2f", old.MaxCapitalDeploymentPct, new.MaxCapitalDeploymentPct)
	}
	if old.MaxPerSector != new.MaxPerSector {
		w.logger.Printf("[config-watcher] max_per_sector: %d -> %d", old.MaxPerSector, new.MaxPerSector)
	}
	if old.MaxHoldDays != new.MaxHoldDays {
		w.logger.Printf("[config-watcher] max_hold_days: %d -> %d", old.MaxHoldDays, new.MaxHoldDays)
	}
	if old.TrailingStop != new.TrailingStop {
		w.logger.Printf("[config-watcher] trailing_stop: enabled=%v trail=%.2f%% activation=%.2f%%",
			new.TrailingStop.Enabled, new.TrailingStop.TrailPct, new.TrailingStop.ActivationPct)
	}
	if old.CircuitBreaker != new.CircuitBreaker {
		w.logger.Printf("[config-watcher] circuit_breaker: consecutive=%d hourly=%d cooldown=%dmin",
			new.CircuitBreaker.MaxConsecutiveFailures, new.CircuitBreaker.MaxFailuresPerHour, new.CircuitBreaker.CooldownMinutes)
	}
}
