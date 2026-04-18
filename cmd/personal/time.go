package main

import (
	"fmt"
	"time"
)

// appLoc holds the application timezone, set at bootstrap via PERSONAL_TZ.
var appLoc = time.UTC

func initTimezone(tzName string) error {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", tzName, err)
	}
	appLoc = loc
	return nil
}

func today() string {
	return time.Now().In(appLoc).Format("2006-01-02")
}

func nowString() string {
	return time.Now().In(appLoc).Format("2006-01-02 15:04:05")
}

func currentYear() int {
	return time.Now().In(appLoc).Year()
}

// ParseTimeOfDay parses "HH:MM" strictly (leading zeros required, 00..23 / 00..59).
func ParseTimeOfDay(s string) (time.Time, error) {
	if len(s) != 5 || s[2] != ':' {
		return time.Time{}, fmt.Errorf("time must be in HH:MM format with leading zeros")
	}
	return time.Parse("15:04", s)
}

// ComputeHours returns (end - start) in hours, rounded to 2 decimal places.
// Both inputs must be "HH:MM". end must be strictly after start.
func ComputeHours(start, end string) (float64, error) {
	s, err := ParseTimeOfDay(start)
	if err != nil {
		return 0, fmt.Errorf("start_time must be HH:MM: %w", err)
	}
	e, err := ParseTimeOfDay(end)
	if err != nil {
		return 0, fmt.Errorf("end_time must be HH:MM: %w", err)
	}
	if !e.After(s) {
		return 0, fmt.Errorf("end_time must be after start_time")
	}
	diff := e.Sub(s).Minutes()
	hours := diff / 60.0
	return float64(int(hours*100+0.5)) / 100.0, nil
}
