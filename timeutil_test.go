package tuitui

import (
	"testing"
	"time"
)

func TestParseRelativeTime(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 10, 30, 0, 0, time.UTC)
	start, end, ok := parseRelativeTime("last_3_days", now)
	if !ok || !start.Equal(now.AddDate(0, 0, -3)) || !end.Equal(now) {
		t.Fatalf("unexpected range: %v %v %v", start, end, ok)
	}
	start, end, ok = parseRelativeTime("this_month", now)
	if !ok || start.Day() != 1 || end.Month() != time.September {
		t.Fatalf("unexpected month range: %v %v", start, end)
	}
}

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()
	if value := formatTimestamp("0"); value != "" {
		t.Fatalf("invalid timestamp must be empty: %q", value)
	}
	if value := formatTimestamp("1723420800"); value == "" {
		t.Fatal("valid timestamp must be formatted")
	}
}
