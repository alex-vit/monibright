package main

import (
	"testing"
	"time"
)

func TestInterpolateTemp(t *testing.T) {
	loc := time.Local
	today := time.Now()
	d := func(h, m int) time.Time {
		return time.Date(today.Year(), today.Month(), today.Day(), h, m, 0, 0, loc)
	}

	sched := sunSchedule{
		CivilTwBegin: d(5, 30),
		Sunrise:      d(6, 0),
		Sunset:       d(18, 0),
		CivilTwEnd:   d(18, 30),
	}
	// Derived boundaries:
	// morningStart = CivilTwBegin = 5:30
	// morningEnd   = 2*Sunrise - CivilTwBegin = 6:30
	// eveningStart = 2*Sunset - CivilTwEnd = 17:30
	// eveningEnd   = CivilTwEnd = 18:30

	const day = 6500
	const night = 3500

	tests := []struct {
		name    string
		now     time.Time
		wantMin int
		wantMax int
	}{
		{"deep night (3am)", d(3, 0), night, night},
		{"before morning twilight", d(5, 0), night, night},
		{"after evening twilight", d(19, 0), night, night},
		{"mid-day (noon)", d(12, 0), day, day},
		{"mid-day (10am)", d(10, 0), day, day},
		{"mid-day (15:00)", d(15, 0), day, day},
		{"morning midpoint (sunrise)", d(6, 0), 4800, 5200},
		{"morning transition start", d(5, 30), night, night + 100},
		{"morning transition end", d(6, 30), day - 100, day},
		{"evening midpoint (sunset)", d(18, 0), 4800, 5200},
		{"evening transition start", d(17, 30), day - 100, day},
		{"evening transition end", d(18, 30), night, night + 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interpolateTemp(tt.now, sched, day, night)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("interpolateTemp(%s) = %d, want [%d, %d]",
					tt.now.Format("15:04"), got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestRoundTo100(t *testing.T) {
	tests := []struct {
		input, want int
	}{
		{3450, 3500},
		{3449, 3400},
		{3549, 3500},
		{3550, 3600},
		{6500, 6500},
		{6549, 6500},
		{6550, 6600},
		{100, 100},
		{0, 0},
		{50, 100},
		{49, 0},
	}

	for _, tt := range tests {
		got := roundTo100(tt.input)
		if got != tt.want {
			t.Errorf("roundTo100(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
