package tuitui

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var relativeTimePattern = regexp.MustCompile(`^last_(\d+)_(minutes|hours|days|months|years)$`)

func parseRelativeTime(value string, now time.Time) (time.Time, time.Time, bool) {
	start, end := now, now
	startOfDay := func(value time.Time) time.Time {
		year, month, day := value.Date()
		return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
	}
	switch value {
	case "today":
		start = startOfDay(now)
		end = start.AddDate(0, 0, 1)
	case "yesterday":
		end = startOfDay(now)
		start = end.AddDate(0, 0, -1)
	case "day_before_yesterday":
		end = startOfDay(now).AddDate(0, 0, -1)
		start = end.AddDate(0, 0, -1)
	case "this_week", "last_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = startOfDay(now).AddDate(0, 0, 1-weekday)
		if value == "last_week" {
			start = start.AddDate(0, 0, -7)
		}
		end = start.AddDate(0, 0, 7)
	case "this_month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	case "last_month":
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		start = end.AddDate(0, -1, 0)
	case "recent_1_hour":
		start = now.Add(-time.Hour)
	case "recent_3_hours":
		start = now.Add(-3 * time.Hour)
	case "recent_24_hours":
		start = now.Add(-24 * time.Hour)
	default:
		match := relativeTimePattern.FindStringSubmatch(value)
		if len(match) != 3 {
			return time.Time{}, time.Time{}, false
		}
		amount, _ := strconv.Atoi(match[1])
		switch match[2] {
		case "minutes":
			start = now.Add(-time.Duration(amount) * time.Minute)
		case "hours":
			start = now.Add(-time.Duration(amount) * time.Hour)
		case "days":
			start = now.AddDate(0, 0, -amount)
		case "months":
			start = now.AddDate(0, -amount, 0)
		case "years":
			start = now.AddDate(-amount, 0, 0)
		}
	}
	return start, end, true
}

func formatTimestamp(value interface{}) string {
	var numeric int64
	switch value := value.(type) {
	case float64:
		numeric = int64(value)
	default:
		numeric, _ = strconv.ParseInt(fmt.Sprint(value), 10, 64)
	}
	if numeric <= 0 {
		return ""
	}
	if numeric < 10000000000 {
		numeric *= 1000
	}
	return time.Unix(numeric/1000, (numeric%1000)*int64(time.Millisecond)).Format("2006-01-02 15:04:05")
}

func timestampString(value interface{}) string {
	if numeric, ok := value.(float64); ok {
		return strconv.FormatInt(int64(numeric), 10)
	}
	return fmt.Sprint(value)
}

func unixMilli(value time.Time) int64 {
	return value.UnixNano() / int64(time.Millisecond)
}
