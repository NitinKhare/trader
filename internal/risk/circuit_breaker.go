// Package risk - circuit_breaker.go provides automatic trading halt
// when repeated failures or unusual conditions are detected.
//
// The circuit breaker tracks:
//   - Consecutive order/API failures (e.g. 5 in a row → trip)
//   - Total failures within a rolling hour (e.g. 10/hour → trip)
//
// When tripped, all new trade entries are blocked until:
//   - The cooldown period expires (auto-reset), or
//   - Manual reset is called.
//
// EXIT orders are never blocked — we always want to be able to close positions.
package risk

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

// CircuitBreaker monitors system health and halts trading when thresholds
// are breached. It is thread-safe and intended to be shared across all
// market-hour jobs.
type CircuitBreaker struct {
	mu                  sync.Mutex
	config              config.CircuitBreakerConfig
	consecutiveFailures int
	hourlyFailures      []time.Time // timestamps of failures within the last hour
	tripped             bool
	trippedAt           time.Time
	tripReason          string
	logger              *log.Logger
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
// Pass a nil logger to use a default no-op logger.
func NewCircuitBreaker(cfg config.CircuitBreakerConfig, logger *log.Logger) *CircuitBreaker {
	if logger == nil {
		logger = log.New(log.Writer(), "", log.LstdFlags)
	}
	return &CircuitBreaker{
		config: cfg,
		logger: logger,
	}
}

// RecordFailure records a failure event and checks whether thresholds
// have been breached. If a threshold is exceeded, the breaker trips.
func (cb *CircuitBreaker) RecordFailure(reason string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.tripped {
		return // already tripped, no need to record more
	}

	now := time.Now()

	// Increment consecutive failures.
	cb.consecutiveFailures++

	// Add to hourly failures (pruning old entries).
	cb.hourlyFailures = append(cb.hourlyFailures, now)
	cb.pruneHourlyFailures(now)

	// Check consecutive failure threshold.
	if cb.config.MaxConsecutiveFailures > 0 &&
		cb.consecutiveFailures >= cb.config.MaxConsecutiveFailures {
		cb.trip(fmt.Sprintf("consecutive failures: %d >= %d (last: %s)",
			cb.consecutiveFailures, cb.config.MaxConsecutiveFailures, reason))
		return
	}

	// Check hourly failure threshold.
	if cb.config.MaxFailuresPerHour > 0 &&
		len(cb.hourlyFailures) >= cb.config.MaxFailuresPerHour {
		cb.trip(fmt.Sprintf("hourly failures: %d >= %d (last: %s)",
			len(cb.hourlyFailures), cb.config.MaxFailuresPerHour, reason))
		return
	}

	cb.logger.Printf("[circuit-breaker] failure recorded: %s (consecutive=%d, hourly=%d)",
		reason, cb.consecutiveFailures, len(cb.hourlyFailures))
}

// RecordSuccess records a successful operation and resets the consecutive
// failure counter. Hourly failures are NOT reset by successes.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures = 0
}

// IsTripped returns true if the circuit breaker is currently tripped.
// It also checks cooldown: if the cooldown period has expired since
// tripping, the breaker auto-resets and returns false.
func (cb *CircuitBreaker) IsTripped() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.tripped {
		return false
	}

	// Check cooldown auto-reset.
	if cb.config.CooldownMinutes > 0 {
		cooldownDuration := time.Duration(cb.config.CooldownMinutes) * time.Minute
		if time.Since(cb.trippedAt) >= cooldownDuration {
			cb.logger.Printf("[circuit-breaker] cooldown expired (%.0f min), auto-resetting",
				cooldownDuration.Minutes())
			cb.resetInternal()
			return false
		}
	}

	return true
}

// TripReason returns the reason the circuit breaker was tripped.
// Returns empty string if not tripped.
func (cb *CircuitBreaker) TripReason() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if !cb.tripped {
		return ""
	}
	return cb.tripReason
}

// Reset manually resets the circuit breaker, clearing all counters.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.tripped {
		cb.logger.Printf("[circuit-breaker] manually reset (was tripped: %s)", cb.tripReason)
	}
	cb.resetInternal()
}

// UpdateConfig updates the circuit breaker configuration.
// Used for config hot-reload. Does NOT reset the tripped state.
func (cb *CircuitBreaker) UpdateConfig(cfg config.CircuitBreakerConfig) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.config = cfg
	cb.logger.Printf("[circuit-breaker] config updated: max_consecutive=%d max_hourly=%d cooldown=%d min",
		cfg.MaxConsecutiveFailures, cfg.MaxFailuresPerHour, cfg.CooldownMinutes)
}

// ConsecutiveFailures returns the current consecutive failure count (for status/debug).
func (cb *CircuitBreaker) ConsecutiveFailures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.consecutiveFailures
}

// HourlyFailures returns the current hourly failure count (for status/debug).
func (cb *CircuitBreaker) HourlyFailures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	now := time.Now()
	cb.pruneHourlyFailures(now)
	return len(cb.hourlyFailures)
}

// ────────────────────────────────────────────────────────────────────
// Internal helpers
// ────────────────────────────────────────────────────────────────────

func (cb *CircuitBreaker) trip(reason string) {
	cb.tripped = true
	cb.trippedAt = time.Now()
	cb.tripReason = reason
	cb.logger.Printf("[circuit-breaker] TRIPPED: %s", reason)
}

func (cb *CircuitBreaker) resetInternal() {
	cb.tripped = false
	cb.trippedAt = time.Time{}
	cb.tripReason = ""
	cb.consecutiveFailures = 0
	cb.hourlyFailures = nil
}

// pruneHourlyFailures removes entries older than 1 hour from the sliding window.
func (cb *CircuitBreaker) pruneHourlyFailures(now time.Time) {
	cutoff := now.Add(-1 * time.Hour)
	i := 0
	for i < len(cb.hourlyFailures) && cb.hourlyFailures[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		cb.hourlyFailures = cb.hourlyFailures[i:]
	}
}
