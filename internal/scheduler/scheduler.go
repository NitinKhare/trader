// Package scheduler manages the system's job lifecycle.
//
// Job schedule (from spec):
//
// Nightly jobs (most important):
//   - Fetch new market data
//   - Run AI scoring
//   - Generate next-day watchlist
//
// Market hour jobs:
//   - Monitor watchlist
//   - Execute pre-planned trades
//   - Manage exits only
//
// Weekly jobs:
//   - Rebuild stock universe
//   - Refresh fundamentals (if used)
package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nitinkhare/algoTradingAgent/internal/market"
)

// JobType categorizes when a job should run.
type JobType string

const (
	JobTypeNightly    JobType = "NIGHTLY"
	JobTypeMarketHour JobType = "MARKET_HOUR"
	JobTypeWeekly     JobType = "WEEKLY"
)

// Job represents a scheduled task.
type Job struct {
	Name     string
	Type     JobType
	RunFunc  func(ctx context.Context) error
}

// Scheduler manages and executes jobs based on market state.
type Scheduler struct {
	calendar *market.Calendar
	jobs     []Job
	logger   *log.Logger
}

// New creates a new scheduler.
func New(calendar *market.Calendar, logger *log.Logger) *Scheduler {
	return &Scheduler{
		calendar: calendar,
		logger:   logger,
	}
}

// RegisterJob adds a job to the scheduler.
func (s *Scheduler) RegisterJob(job Job) {
	s.jobs = append(s.jobs, job)
	s.logger.Printf("[scheduler] registered job: %s (type: %s)", job.Name, job.Type)
}

// RunNightlyJobs executes all nightly jobs in sequence.
// These run after market close, typically around 6–8 PM IST.
// This is the most important job cycle — it prepares the next trading day.
func (s *Scheduler) RunNightlyJobs(ctx context.Context) error {
	s.logger.Println("[scheduler] starting nightly job cycle")

	for _, job := range s.jobs {
		if job.Type != JobTypeNightly {
			continue
		}

		s.logger.Printf("[scheduler] running nightly job: %s", job.Name)
		start := time.Now()

		if err := job.RunFunc(ctx); err != nil {
			s.logger.Printf("[scheduler] FAILED nightly job %s: %v", job.Name, err)
			return fmt.Errorf("nightly job %s failed: %w", job.Name, err)
		}

		s.logger.Printf("[scheduler] completed nightly job %s in %v", job.Name, time.Since(start))
	}

	s.logger.Println("[scheduler] nightly job cycle complete")
	return nil
}

// RunMarketHourJobs executes market-hour jobs.
// These run during market hours (9:15 AM – 3:30 PM IST).
// They monitor the watchlist and execute pre-planned trades.
func (s *Scheduler) RunMarketHourJobs(ctx context.Context) error {
	now := time.Now()

	if !s.calendar.IsMarketOpen(now) {
		s.logger.Println("[scheduler] market is closed, skipping market-hour jobs")
		return nil
	}

	s.logger.Println("[scheduler] starting market-hour job cycle")

	for _, job := range s.jobs {
		if job.Type != JobTypeMarketHour {
			continue
		}

		s.logger.Printf("[scheduler] running market-hour job: %s", job.Name)
		if err := job.RunFunc(ctx); err != nil {
			s.logger.Printf("[scheduler] FAILED market-hour job %s: %v", job.Name, err)
			// Market-hour job failures are logged but don't stop other jobs.
			// Safety: better to log and continue than halt the system.
		}
	}

	return nil
}

// ForceRunMarketHourJobs runs market-hour jobs without checking
// whether the market is currently open. Used in integration tests
// that need to exercise the full pipeline outside of IST 9:15–15:30.
func (s *Scheduler) ForceRunMarketHourJobs(ctx context.Context) error {
	s.logger.Println("[scheduler] force-running market-hour jobs (calendar check skipped)")

	for _, job := range s.jobs {
		if job.Type != JobTypeMarketHour {
			continue
		}

		s.logger.Printf("[scheduler] running market-hour job: %s", job.Name)
		if err := job.RunFunc(ctx); err != nil {
			s.logger.Printf("[scheduler] FAILED market-hour job %s: %v", job.Name, err)
			// Same policy as RunMarketHourJobs: log and continue.
		}
	}

	return nil
}

// RunWeeklyJobs executes weekly maintenance jobs.
// These typically run on weekends.
func (s *Scheduler) RunWeeklyJobs(ctx context.Context) error {
	s.logger.Println("[scheduler] starting weekly job cycle")

	for _, job := range s.jobs {
		if job.Type != JobTypeWeekly {
			continue
		}

		s.logger.Printf("[scheduler] running weekly job: %s", job.Name)
		if err := job.RunFunc(ctx); err != nil {
			s.logger.Printf("[scheduler] FAILED weekly job %s: %v", job.Name, err)
			return fmt.Errorf("weekly job %s failed: %w", job.Name, err)
		}
	}

	s.logger.Println("[scheduler] weekly job cycle complete")
	return nil
}

// Status returns current market state information.
func (s *Scheduler) Status() string {
	now := time.Now()
	isOpen := s.calendar.IsMarketOpen(now)
	isTrading := s.calendar.IsTradingDay(now)
	nextSession := s.calendar.TimeUntilNextSession(now)

	status := fmt.Sprintf(
		"Market Status: open=%v trading_day=%v next_session_in=%v",
		isOpen, isTrading, nextSession.Round(time.Minute),
	)

	if reason := s.calendar.HolidayReason(now); reason != "" {
		status += fmt.Sprintf(" holiday=%s", reason)
	}

	return status
}
