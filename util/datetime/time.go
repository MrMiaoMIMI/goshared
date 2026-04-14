package datetime

import (
	"fmt"
	"time"
)

const (
	DateLayout     = "2006-01-02"
	TimeLayout     = "15:04:05"
	DateTimeLayout = "2006-01-02 15:04:05"
	ISO8601Layout  = time.RFC3339
)

// NowMillis returns the current time in milliseconds.
func NowMillis() int64 {
	return time.Now().UnixMilli()
}

// NowSeconds returns the current time in seconds (Unix timestamp).
func NowSeconds() int64 {
	return time.Now().Unix()
}

// NextNDayTime returns the time n days from now.
func NextNDayTime(n int) time.Time {
	return time.Now().AddDate(0, 0, n)
}

// NextNDayMillis returns the time n days from now in milliseconds.
func NextNDayMillis(n int) int64 {
	return NextNDayTime(n).UnixMilli()
}

// StartOfDay returns the start of day (00:00:00) for the given time.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of day (23:59:59.999999999) for the given time.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-1), t.Location())
}

// StartOfMonth returns the start of the month for the given time.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the last moment of the month for the given time.
func EndOfMonth(t time.Time) time.Time {
	return StartOfMonth(t).AddDate(0, 1, 0).Add(-1)
}

// FormatDate formats a time as "2006-01-02".
func FormatDate(t time.Time) string {
	return t.Format(DateLayout)
}

// FormatDateTime formats a time as "2006-01-02 15:04:05".
func FormatDateTime(t time.Time) string {
	return t.Format(DateTimeLayout)
}

// ParseDate parses a date string in "2006-01-02" format.
func ParseDate(s string) (time.Time, error) {
	return time.Parse(DateLayout, s)
}

// ParseDateTime parses a datetime string in "2006-01-02 15:04:05" format.
func ParseDateTime(s string) (time.Time, error) {
	return time.Parse(DateTimeLayout, s)
}

// ParseDateTimeInLocation parses a datetime string in the given location.
func ParseDateTimeInLocation(s string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(DateTimeLayout, s, loc)
}

// MillisToTime converts milliseconds timestamp to time.Time.
func MillisToTime(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// SecondsToTime converts seconds timestamp to time.Time.
func SecondsToTime(sec int64) time.Time {
	return time.Unix(sec, 0)
}

// DaysBetween returns the number of days between two times (absolute value).
func DaysBetween(a, b time.Time) int {
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	return int(diff.Hours() / 24)
}

// IsToday checks if the given time is today.
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}

// IsSameDay checks if two times are on the same calendar day.
func IsSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// AddDays returns the time with n days added.
func AddDays(t time.Time, n int) time.Time {
	return t.AddDate(0, 0, n)
}

// AddHours returns the time with n hours added.
func AddHours(t time.Time, n int) time.Time {
	return t.Add(time.Duration(n) * time.Hour)
}

// AddMinutes returns the time with n minutes added.
func AddMinutes(t time.Time, n int) time.Time {
	return t.Add(time.Duration(n) * time.Minute)
}

// IsWeekend returns true if the given time falls on Saturday or Sunday.
func IsWeekend(t time.Time) bool {
	day := t.Weekday()
	return day == time.Saturday || day == time.Sunday
}

// IsWeekday returns true if the given time falls on Monday through Friday.
func IsWeekday(t time.Time) bool {
	return !IsWeekend(t)
}

// WeekdayName returns the name of the weekday for the given time.
func WeekdayName(t time.Time) string {
	return t.Weekday().String()
}

// TimeAgo returns a human-readable relative time string (e.g. "5 minutes ago").
// For future times, returns "in the future".
func TimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		return "in the future"
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		n := int(d.Minutes())
		if n == 1 {
			return "1 minute ago"
		}
		return formatAgo(n, "minutes")
	case d < 24*time.Hour:
		n := int(d.Hours())
		if n == 1 {
			return "1 hour ago"
		}
		return formatAgo(n, "hours")
	case d < 30*24*time.Hour:
		n := int(d.Hours() / 24)
		if n == 1 {
			return "1 day ago"
		}
		return formatAgo(n, "days")
	case d < 365*24*time.Hour:
		n := int(d.Hours() / 24 / 30)
		if n == 1 {
			return "1 month ago"
		}
		return formatAgo(n, "months")
	default:
		n := int(d.Hours() / 24 / 365)
		if n == 1 {
			return "1 year ago"
		}
		return formatAgo(n, "years")
	}
}

func formatAgo(n int, unit string) string {
	return fmt.Sprintf("%d %s ago", n, unit)
}

// StartOfWeek returns the start of the week (Monday 00:00:00) for the given time.
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return StartOfDay(t.AddDate(0, 0, -(weekday - 1)))
}

// EndOfWeek returns the end of the week (Sunday 23:59:59) for the given time.
func EndOfWeek(t time.Time) time.Time {
	return EndOfDay(StartOfWeek(t).AddDate(0, 0, 6))
}

// IsBetween checks if t is between start and end (inclusive).
func IsBetween(t, start, end time.Time) bool {
	return (t.Equal(start) || t.After(start)) && (t.Equal(end) || t.Before(end))
}

// Age calculates the age in years from birthDate to now.
func Age(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()
	if now.Month() < birthDate.Month() ||
		(now.Month() == birthDate.Month() && now.Day() < birthDate.Day()) {
		years--
	}
	return years
}
