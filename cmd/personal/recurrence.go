package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type Pattern struct {
	Type string `json:"type"`
	Days []int  `json:"days,omitempty"`
	Day  int    `json:"day,omitempty"`
}

func ParsePattern(s string) (Pattern, error) {
	var p Pattern
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return Pattern{}, fmt.Errorf("invalid recurrence JSON: %w", err)
	}
	switch p.Type {
	case "weekday":
		if len(p.Days) == 0 {
			return Pattern{}, fmt.Errorf("weekday pattern requires non-empty days (1..7, Mon=1)")
		}
		for _, d := range p.Days {
			if d < 1 || d > 7 {
				return Pattern{}, fmt.Errorf("weekday days must be 1..7, got %d", d)
			}
		}
		sort.Ints(p.Days)
	case "monthday":
		if p.Day < 1 || p.Day > 31 {
			return Pattern{}, fmt.Errorf("monthday day must be 1..31, got %d", p.Day)
		}
	default:
		return Pattern{}, fmt.Errorf("pattern type must be 'weekday' or 'monthday', got %q", p.Type)
	}
	return p, nil
}

func (p Pattern) Serialize() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p Pattern) NextOccurrence(from time.Time) (time.Time, error) {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	switch p.Type {
	case "weekday":
		return nextWeekday(p.Days, from), nil
	case "monthday":
		return nextMonthday(p.Day, from), nil
	}
	return time.Time{}, fmt.Errorf("unknown pattern type %q", p.Type)
}

func isoWeekday(t time.Time) int {
	w := int(t.Weekday())
	if w == 0 {
		return 7
	}
	return w
}

func nextWeekday(days []int, from time.Time) time.Time {
	for offset := 1; offset <= 7; offset++ {
		candidate := from.AddDate(0, 0, offset)
		cw := isoWeekday(candidate)
		for _, d := range days {
			if d == cw {
				return candidate
			}
		}
	}
	return from.AddDate(0, 0, 7)
}

func nextMonthday(day int, from time.Time) time.Time {
	year, month := from.Year(), from.Month()
	if from.Day() < day {
		target := clampMonthday(year, month, day, from.Location())
		if target.After(from) {
			return target
		}
	}
	month++
	if month > 12 {
		month = 1
		year++
	}
	return clampMonthday(year, month, day, from.Location())
}

func clampMonthday(year int, month time.Month, day int, loc *time.Location) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}
