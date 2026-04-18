package main

import (
	"math"
	"testing"
)

func TestComputeHours(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		end       string
		want      float64
		wantError bool
	}{
		{"full hour", "09:00", "17:00", 8.0, false},
		{"half hour", "09:00", "09:30", 0.5, false},
		{"quarter", "08:00", "08:15", 0.25, false},
		{"with minutes", "08:30", "10:15", 1.75, false},
		{"end before start", "10:00", "09:00", 0, true},
		{"equal", "10:00", "10:00", 0, true},
		{"bad format start", "9-00", "10:00", 0, true},
		{"bad format end", "09:00", "10:0Z", 0, true},
		{"minute precision 2dp", "08:00", "08:20", 0.33, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ComputeHours(tc.start, tc.end)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got hours=%v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got-tc.want) > 0.001 {
				t.Errorf("ComputeHours(%q, %q) = %v, want %v", tc.start, tc.end, got, tc.want)
			}
		})
	}
}

func TestParseTimeOfDay(t *testing.T) {
	if _, err := ParseTimeOfDay("09:30"); err != nil {
		t.Errorf("ParseTimeOfDay(09:30) failed: %v", err)
	}
	if _, err := ParseTimeOfDay("25:00"); err == nil {
		t.Errorf("ParseTimeOfDay(25:00) should fail")
	}
	if _, err := ParseTimeOfDay("9:00"); err == nil {
		t.Errorf("ParseTimeOfDay(9:00) without leading zero should fail")
	}
}
