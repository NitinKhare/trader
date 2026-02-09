// Package market handles market state awareness.
//
// Design rules (from spec):
//   - System must know if today is a trading day.
//   - System must know if the market is currently open.
//   - Do not rely only on time checks.
//   - Use exchange calendar data.
//   - One central MarketCalendar module.
package market

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// IST is the Indian Standard Time location.
var IST *time.Location

func init() {
	var err error
	IST, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		panic(fmt.Sprintf("market: failed to load IST timezone: %v", err))
	}
}

// NSE market hours (IST).
const (
	MarketOpenHour  = 9
	MarketOpenMin   = 15
	MarketCloseHour = 15
	MarketCloseMin  = 30
)

// Calendar provides exchange calendar and market state information.
type Calendar struct {
	// holidays is a set of dates (YYYY-MM-DD) when the exchange is closed.
	holidays map[string]string // date -> reason
}

// HolidayEntry represents a single exchange holiday.
type HolidayEntry struct {
	Date   string `json:"date"`   // YYYY-MM-DD
	Reason string `json:"reason"` // e.g., "Republic Day", "Diwali"
}

// NewCalendar creates a Calendar from a JSON holiday file.
// The file should contain an array of HolidayEntry objects.
func NewCalendar(holidayFilePath string) (*Calendar, error) {
	data, err := os.ReadFile(holidayFilePath)
	if err != nil {
		return nil, fmt.Errorf("market calendar: read holidays file: %w", err)
	}

	var entries []HolidayEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("market calendar: parse holidays: %w", err)
	}

	holidays := make(map[string]string, len(entries))
	for _, e := range entries {
		holidays[e.Date] = e.Reason
	}

	return &Calendar{holidays: holidays}, nil
}

// NewCalendarFromHolidays creates a Calendar directly from a holiday map.
// Useful for testing.
func NewCalendarFromHolidays(holidays map[string]string) *Calendar {
	return &Calendar{holidays: holidays}
}

// IsTradingDay returns true if the given date is a valid trading day.
// A trading day is a weekday that is not an exchange holiday.
func (c *Calendar) IsTradingDay(date time.Time) bool {
	d := date.In(IST)

	// Weekends are not trading days.
	if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		return false
	}

	// Check exchange holidays.
	dateStr := d.Format("2006-01-02")
	if _, isHoliday := c.holidays[dateStr]; isHoliday {
		return false
	}

	return true
}

// HolidayReason returns the reason for a holiday, or empty string if not a holiday.
func (c *Calendar) HolidayReason(date time.Time) string {
	dateStr := date.In(IST).Format("2006-01-02")
	return c.holidays[dateStr]
}

// IsMarketOpen returns true if the NSE is currently in trading hours.
func (c *Calendar) IsMarketOpen(now time.Time) bool {
	t := now.In(IST)

	if !c.IsTradingDay(t) {
		return false
	}

	currentMinutes := t.Hour()*60 + t.Minute()
	openMinutes := MarketOpenHour*60 + MarketOpenMin
	closeMinutes := MarketCloseHour*60 + MarketCloseMin

	return currentMinutes >= openMinutes && currentMinutes < closeMinutes
}

// TimeUntilNextSession returns the duration until the next market open.
// If the market is currently open, returns 0.
func (c *Calendar) TimeUntilNextSession(now time.Time) time.Duration {
	t := now.In(IST)

	if c.IsMarketOpen(t) {
		return 0
	}

	// Find the next trading day.
	candidate := t
	for i := 0; i < 10; i++ { // Look ahead up to 10 days.
		// If we're before market open today and today is a trading day, next open is today.
		if i == 0 && c.IsTradingDay(candidate) {
			todayOpen := time.Date(candidate.Year(), candidate.Month(), candidate.Day(),
				MarketOpenHour, MarketOpenMin, 0, 0, IST)
			if t.Before(todayOpen) {
				return todayOpen.Sub(t)
			}
		}

		// Try next day.
		candidate = candidate.AddDate(0, 0, 1)
		if c.IsTradingDay(candidate) {
			nextOpen := time.Date(candidate.Year(), candidate.Month(), candidate.Day(),
				MarketOpenHour, MarketOpenMin, 0, 0, IST)
			return nextOpen.Sub(t)
		}
	}

	// Fallback: this shouldn't happen with reasonable holiday data.
	return 24 * time.Hour
}

// NextTradingDay returns the next trading day after the given date.
func (c *Calendar) NextTradingDay(date time.Time) time.Time {
	candidate := date.In(IST).AddDate(0, 0, 1)
	for i := 0; i < 10; i++ {
		if c.IsTradingDay(candidate) {
			return candidate
		}
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

// PreviousTradingDay returns the most recent trading day before the given date.
func (c *Calendar) PreviousTradingDay(date time.Time) time.Time {
	candidate := date.In(IST).AddDate(0, 0, -1)
	for i := 0; i < 10; i++ {
		if c.IsTradingDay(candidate) {
			return candidate
		}
		candidate = candidate.AddDate(0, 0, -1)
	}
	return candidate
}
