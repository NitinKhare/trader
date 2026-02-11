package risk

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/config"
)

func cbLogger() *log.Logger {
	return log.New(os.Stdout, "[cb-test] ", log.LstdFlags)
}

func TestCircuitBreaker_ConsecutiveTrip(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 3,
	}, cbLogger())

	cb.RecordFailure("api error 1")
	cb.RecordFailure("api error 2")
	if cb.IsTripped() {
		t.Error("should not be tripped after 2 failures (threshold=3)")
	}

	cb.RecordFailure("api error 3")
	if !cb.IsTripped() {
		t.Error("should be tripped after 3 consecutive failures")
	}

	reason := cb.TripReason()
	if reason == "" {
		t.Error("expected non-empty trip reason")
	}
}

func TestCircuitBreaker_SuccessResetsConsecutive(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 3,
	}, cbLogger())

	cb.RecordFailure("fail 1")
	cb.RecordFailure("fail 2")
	cb.RecordSuccess() // should reset counter
	cb.RecordFailure("fail 3")
	cb.RecordFailure("fail 4")

	if cb.IsTripped() {
		t.Error("should not be tripped — success reset consecutive counter")
	}

	if cb.ConsecutiveFailures() != 2 {
		t.Errorf("expected consecutive=2 after reset+2 fails, got %d", cb.ConsecutiveFailures())
	}
}

func TestCircuitBreaker_HourlyTrip(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxFailuresPerHour: 5,
	}, cbLogger())

	for i := 0; i < 4; i++ {
		cb.RecordFailure("api error")
		cb.RecordSuccess() // reset consecutive, but hourly still counts
	}

	if cb.IsTripped() {
		t.Error("should not be tripped after 4 hourly failures (threshold=5)")
	}

	cb.RecordFailure("api error 5")
	if !cb.IsTripped() {
		t.Error("should be tripped after 5 hourly failures")
	}
}

func TestCircuitBreaker_CooldownAutoReset(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
		CooldownMinutes:        1, // will simulate with time manipulation
	}, cbLogger())

	cb.RecordFailure("fail")
	cb.RecordFailure("fail")
	if !cb.IsTripped() {
		t.Fatal("should be tripped")
	}

	// Manually set trippedAt to 2 minutes ago to simulate cooldown expiry.
	cb.mu.Lock()
	cb.trippedAt = time.Now().Add(-2 * time.Minute)
	cb.mu.Unlock()

	if cb.IsTripped() {
		t.Error("should auto-reset after cooldown expires")
	}

	// Counters should be reset too.
	if cb.ConsecutiveFailures() != 0 {
		t.Errorf("expected consecutive=0 after auto-reset, got %d", cb.ConsecutiveFailures())
	}
}

func TestCircuitBreaker_NoCooldown(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
		CooldownMinutes:        0, // no auto-reset
	}, cbLogger())

	cb.RecordFailure("fail")
	cb.RecordFailure("fail")
	if !cb.IsTripped() {
		t.Fatal("should be tripped")
	}

	// Even after time passes, should stay tripped.
	cb.mu.Lock()
	cb.trippedAt = time.Now().Add(-1 * time.Hour)
	cb.mu.Unlock()

	if !cb.IsTripped() {
		t.Error("should stay tripped with CooldownMinutes=0")
	}
}

func TestCircuitBreaker_ManualReset(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
	}, cbLogger())

	cb.RecordFailure("fail")
	cb.RecordFailure("fail")
	if !cb.IsTripped() {
		t.Fatal("should be tripped")
	}

	cb.Reset()
	if cb.IsTripped() {
		t.Error("should not be tripped after manual reset")
	}
	if cb.TripReason() != "" {
		t.Error("trip reason should be empty after reset")
	}
}

func TestCircuitBreaker_Disabled(t *testing.T) {
	// All thresholds at zero = disabled.
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{}, cbLogger())

	for i := 0; i < 100; i++ {
		cb.RecordFailure("fail")
	}
	if cb.IsTripped() {
		t.Error("should never trip when all thresholds are 0 (disabled)")
	}
}

func TestCircuitBreaker_UpdateConfig(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 10, // high threshold
	}, cbLogger())

	cb.RecordFailure("fail")
	cb.RecordFailure("fail")
	cb.RecordFailure("fail")
	if cb.IsTripped() {
		t.Error("should not be tripped (threshold=10)")
	}

	// Lower the threshold via config update.
	cb.UpdateConfig(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 3,
	})

	// Record one more failure — should now exceed new threshold.
	// But note: existing count is 3, new threshold is 3.
	// We need the NEXT failure to trip it.
	// Actually, already at 3 with threshold 3, but RecordFailure checks >=.
	// The check happens inside RecordFailure, and consecutiveFailures is already 3.
	// We need to call RecordFailure again for the check to trigger.
	cb.RecordFailure("fail after config change")
	if !cb.IsTripped() {
		t.Error("should be tripped after config update lowered threshold")
	}
}

func TestCircuitBreaker_HourlyPruning(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxFailuresPerHour: 3,
	}, cbLogger())

	// Add 2 failures "from the past" (more than 1 hour ago).
	cb.mu.Lock()
	pastTime := time.Now().Add(-2 * time.Hour)
	cb.hourlyFailures = append(cb.hourlyFailures, pastTime, pastTime)
	cb.mu.Unlock()

	// These should be pruned. Add 2 more recent failures.
	cb.RecordFailure("recent fail 1")
	cb.RecordSuccess()
	cb.RecordFailure("recent fail 2")

	if cb.IsTripped() {
		t.Error("should not be tripped — old failures should be pruned (2 recent < 3)")
	}

	// Verify hourly count only includes recent.
	hourly := cb.HourlyFailures()
	if hourly != 2 {
		t.Errorf("expected 2 hourly failures (after pruning), got %d", hourly)
	}
}

func TestCircuitBreaker_AlreadyTripped_IgnoresMore(t *testing.T) {
	cb := NewCircuitBreaker(config.CircuitBreakerConfig{
		MaxConsecutiveFailures: 2,
	}, cbLogger())

	cb.RecordFailure("fail 1")
	cb.RecordFailure("fail 2") // trips

	reason := cb.TripReason()

	// More failures shouldn't change the trip reason.
	cb.RecordFailure("fail 3")
	cb.RecordFailure("fail 4")

	if cb.TripReason() != reason {
		t.Error("trip reason should not change after already tripped")
	}
}
