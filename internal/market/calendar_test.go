package market

import (
	"testing"
	"time"
)

func makeTestCalendar() *Calendar {
	return NewCalendarFromHolidays(map[string]string{
		"2026-01-26": "Republic Day",
		"2026-08-15": "Independence Day",
		"2026-10-02": "Gandhi Jayanti",
	})
}

func TestCalendar_WeekdayIsTradingDay(t *testing.T) {
	cal := makeTestCalendar()
	// Monday, Feb 2, 2026.
	monday := time.Date(2026, 2, 2, 10, 0, 0, 0, IST)
	if !cal.IsTradingDay(monday) {
		t.Error("expected Monday to be a trading day")
	}
}

func TestCalendar_WeekendIsNotTradingDay(t *testing.T) {
	cal := makeTestCalendar()
	saturday := time.Date(2026, 2, 7, 10, 0, 0, 0, IST)
	sunday := time.Date(2026, 2, 8, 10, 0, 0, 0, IST)

	if cal.IsTradingDay(saturday) {
		t.Error("expected Saturday to not be a trading day")
	}
	if cal.IsTradingDay(sunday) {
		t.Error("expected Sunday to not be a trading day")
	}
}

func TestCalendar_HolidayIsNotTradingDay(t *testing.T) {
	cal := makeTestCalendar()
	republicDay := time.Date(2026, 1, 26, 10, 0, 0, 0, IST)

	if cal.IsTradingDay(republicDay) {
		t.Error("expected Republic Day to not be a trading day")
	}
	if reason := cal.HolidayReason(republicDay); reason != "Republic Day" {
		t.Errorf("expected 'Republic Day', got %q", reason)
	}
}

func TestCalendar_MarketOpenDuringTradingHours(t *testing.T) {
	cal := makeTestCalendar()
	// 10:30 AM IST on a trading day.
	during := time.Date(2026, 2, 2, 10, 30, 0, 0, IST)
	if !cal.IsMarketOpen(during) {
		t.Error("expected market to be open at 10:30 AM IST on trading day")
	}
}

func TestCalendar_MarketClosedBeforeOpen(t *testing.T) {
	cal := makeTestCalendar()
	// 9:00 AM IST (before 9:15 open).
	before := time.Date(2026, 2, 2, 9, 0, 0, 0, IST)
	if cal.IsMarketOpen(before) {
		t.Error("expected market to be closed at 9:00 AM IST")
	}
}

func TestCalendar_MarketClosedAfterClose(t *testing.T) {
	cal := makeTestCalendar()
	// 3:31 PM IST (after 3:30 close).
	after := time.Date(2026, 2, 2, 15, 31, 0, 0, IST)
	if cal.IsMarketOpen(after) {
		t.Error("expected market to be closed at 3:31 PM IST")
	}
}

func TestCalendar_MarketClosedOnWeekend(t *testing.T) {
	cal := makeTestCalendar()
	saturday := time.Date(2026, 2, 7, 10, 30, 0, 0, IST)
	if cal.IsMarketOpen(saturday) {
		t.Error("expected market to be closed on Saturday")
	}
}

func TestCalendar_TimeUntilNextSession(t *testing.T) {
	cal := makeTestCalendar()

	// After market close on Friday → next session is Monday.
	friday := time.Date(2026, 2, 6, 16, 0, 0, 0, IST)
	duration := cal.TimeUntilNextSession(friday)

	if duration <= 0 {
		t.Errorf("expected positive duration, got %v", duration)
	}

	// During market hours → should be 0.
	during := time.Date(2026, 2, 2, 10, 30, 0, 0, IST)
	duration = cal.TimeUntilNextSession(during)
	if duration != 0 {
		t.Errorf("expected 0 during market hours, got %v", duration)
	}
}

func TestCalendar_NextTradingDay(t *testing.T) {
	cal := makeTestCalendar()

	// Friday → next trading day is Monday.
	friday := time.Date(2026, 2, 6, 0, 0, 0, 0, IST)
	next := cal.NextTradingDay(friday)

	if next.Weekday() != time.Monday {
		t.Errorf("expected Monday after Friday, got %s", next.Weekday())
	}
}

func TestCalendar_PreviousTradingDay(t *testing.T) {
	cal := makeTestCalendar()

	// Monday → previous trading day is Friday.
	monday := time.Date(2026, 2, 9, 0, 0, 0, 0, IST)
	prev := cal.PreviousTradingDay(monday)

	if prev.Weekday() != time.Friday {
		t.Errorf("expected Friday before Monday, got %s", prev.Weekday())
	}
}
