package main

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return d
}

func TestParsePattern(t *testing.T) {
	cases := []struct {
		in        string
		wantType  string
		wantDays  []int
		wantDay   int
		wantError bool
	}{
		{`{"type":"weekday","days":[1,3,5]}`, "weekday", []int{1, 3, 5}, 0, false},
		{`{"type":"monthday","day":15}`, "monthday", nil, 15, false},
		{`{"type":"weekday","days":[]}`, "", nil, 0, true},
		{`{"type":"weekday","days":[8]}`, "", nil, 0, true},
		{`{"type":"monthday","day":32}`, "", nil, 0, true},
		{`{"type":"monthday","day":0}`, "", nil, 0, true},
		{`{"type":"foo"}`, "", nil, 0, true},
		{`{}`, "", nil, 0, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			p, err := ParsePattern(c.in)
			if c.wantError {
				if err == nil {
					t.Fatalf("expected error for %q, got %+v", c.in, p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Type != c.wantType {
				t.Errorf("Type = %q, want %q", p.Type, c.wantType)
			}
			if c.wantType == "weekday" {
				if len(p.Days) != len(c.wantDays) {
					t.Fatalf("Days length %d, want %d", len(p.Days), len(c.wantDays))
				}
				for i, d := range p.Days {
					if d != c.wantDays[i] {
						t.Errorf("Days[%d] = %d, want %d", i, d, c.wantDays[i])
					}
				}
			}
			if c.wantType == "monthday" && p.Day != c.wantDay {
				t.Errorf("Day = %d, want %d", p.Day, c.wantDay)
			}
		})
	}
}

func TestNextOccurrenceWeekday(t *testing.T) {
	p := Pattern{Type: "weekday", Days: []int{1}}

	cases := []struct {
		from, want string
	}{
		{"2026-04-13", "2026-04-20"},
		{"2026-04-15", "2026-04-20"},
		{"2026-04-19", "2026-04-20"},
		{"2026-04-20", "2026-04-27"},
	}
	for _, c := range cases {
		t.Run(c.from, func(t *testing.T) {
			got, err := p.NextOccurrence(mustDate(t, c.from))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			gotS := got.Format("2006-01-02")
			if gotS != c.want {
				t.Errorf("NextOccurrence(%s) = %s, want %s", c.from, gotS, c.want)
			}
		})
	}
}

func TestNextOccurrenceWeekdayMultiple(t *testing.T) {
	p := Pattern{Type: "weekday", Days: []int{1, 3, 5}}
	got, _ := p.NextOccurrence(mustDate(t, "2026-04-13"))
	if got.Format("2006-01-02") != "2026-04-15" {
		t.Errorf("got %s, want 2026-04-15", got.Format("2006-01-02"))
	}
	got, _ = p.NextOccurrence(mustDate(t, "2026-04-18"))
	if got.Format("2006-01-02") != "2026-04-20" {
		t.Errorf("got %s, want 2026-04-20", got.Format("2006-01-02"))
	}
}

func TestNextOccurrenceMonthday(t *testing.T) {
	p := Pattern{Type: "monthday", Day: 15}
	cases := []struct{ from, want string }{
		{"2026-04-10", "2026-04-15"},
		{"2026-04-15", "2026-05-15"},
		{"2026-04-16", "2026-05-15"},
	}
	for _, c := range cases {
		got, _ := p.NextOccurrence(mustDate(t, c.from))
		if got.Format("2006-01-02") != c.want {
			t.Errorf("from %s: got %s, want %s", c.from, got.Format("2006-01-02"), c.want)
		}
	}
}

func TestNextOccurrenceMonthdayOverflow(t *testing.T) {
	p := Pattern{Type: "monthday", Day: 31}
	got, _ := p.NextOccurrence(mustDate(t, "2026-01-31"))
	if got.Format("2006-01-02") != "2026-02-28" {
		t.Errorf("got %s, want 2026-02-28", got.Format("2006-01-02"))
	}
	got, _ = p.NextOccurrence(mustDate(t, "2026-02-28"))
	if got.Format("2006-01-02") != "2026-03-31" {
		t.Errorf("got %s, want 2026-03-31", got.Format("2006-01-02"))
	}
}

func TestPatternSerialize(t *testing.T) {
	p := Pattern{Type: "weekday", Days: []int{1, 3, 5}}
	s, err := p.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	round, _ := ParsePattern(s)
	if round.Type != "weekday" || len(round.Days) != 3 {
		t.Errorf("roundtrip failed: %+v", round)
	}
}
