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

// TestInterpolateTempStaleSchedule verifies that interpolateTemp produces
// correct results even when the schedule dates don't match "now"'s date.
// This is the exact bug that caused permanent 3500K after midnight on 2026-03-02:
// schedule had March 1 dates, now was March 2 → everything looked "after evening".
func TestInterpolateTempStaleSchedule(t *testing.T) {
	loc := time.Local
	const day = 6500
	const night = 3500

	// Schedule anchored to "yesterday"
	yesterday := time.Now().AddDate(0, 0, -1)
	yd := func(h, m int) time.Time {
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), h, m, 0, 0, loc)
	}
	staleSched := sunSchedule{
		CivilTwBegin: yd(5, 30),
		Sunrise:      yd(6, 0),
		Sunset:       yd(18, 0),
		CivilTwEnd:   yd(18, 30),
	}

	today := time.Now()
	td := func(h, m int) time.Time {
		return time.Date(today.Year(), today.Month(), today.Day(), h, m, 0, 0, loc)
	}

	// With a stale schedule, mid-morning today should still be day temp.
	// This was the bug: 10:00 today > 18:30 yesterday → returned nightTemp.
	got := interpolateTemp(td(10, 0), staleSched, day, night)
	if got != day {
		t.Errorf("stale schedule: interpolateTemp(10:00 today, yesterday sched) = %d, want %d (day)", got, day)
	}

	// 03:00 today should be night — this worked even with the bug, but verify.
	got = interpolateTemp(td(3, 0), staleSched, day, night)
	if got != night {
		t.Errorf("stale schedule: interpolateTemp(03:00 today, yesterday sched) = %d, want %d (night)", got, night)
	}

	// Noon should be full day.
	got = interpolateTemp(td(12, 0), staleSched, day, night)
	if got != day {
		t.Errorf("stale schedule: interpolateTemp(12:00 today, yesterday sched) = %d, want %d (day)", got, day)
	}

	// 08:49 — the exact time the bug hit on 2026-03-02.
	got = interpolateTemp(td(8, 49), staleSched, day, night)
	if got != day {
		t.Errorf("stale schedule: interpolateTemp(08:49 today, yesterday sched) = %d, want %d (day)", got, day)
	}
}

func TestDefaultSunScheduleUsesToday(t *testing.T) {
	sched := defaultSunSchedule()
	today := time.Now()

	if sched.Sunrise.Day() != today.Day() || sched.Sunrise.Month() != today.Month() {
		t.Errorf("defaultSunSchedule sunrise date = %s, want today %s",
			sched.Sunrise.Format("2006-01-02"), today.Format("2006-01-02"))
	}
	if sched.Sunset.Day() != today.Day() || sched.Sunset.Month() != today.Month() {
		t.Errorf("defaultSunSchedule sunset date = %s, want today %s",
			sched.Sunset.Format("2006-01-02"), today.Format("2006-01-02"))
	}
	if sched.CivilTwBegin.Day() != today.Day() {
		t.Errorf("defaultSunSchedule dawn date = %s, want today",
			sched.CivilTwBegin.Format("2006-01-02"))
	}
	if sched.CivilTwEnd.Day() != today.Day() {
		t.Errorf("defaultSunSchedule dusk date = %s, want today",
			sched.CivilTwEnd.Format("2006-01-02"))
	}
}

// TestInterpolateTempMidnight verifies correct behavior around midnight boundary.
func TestInterpolateTempMidnight(t *testing.T) {
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

	const day = 6500
	const night = 3500

	// Just after midnight
	got := interpolateTemp(d(0, 0), sched, day, night)
	if got != night {
		t.Errorf("midnight: got %d, want %d", got, night)
	}

	// Just before midnight
	got = interpolateTemp(d(23, 59), sched, day, night)
	if got != night {
		t.Errorf("23:59: got %d, want %d", got, night)
	}
}

func TestNormalizeSched(t *testing.T) {
	loc := time.Local
	// Schedule from 2026-01-15
	old := sunSchedule{
		CivilTwBegin: time.Date(2026, 1, 15, 5, 30, 0, 0, loc),
		Sunrise:      time.Date(2026, 1, 15, 6, 0, 0, 0, loc),
		Sunset:       time.Date(2026, 1, 15, 18, 0, 0, 0, loc),
		CivilTwEnd:   time.Date(2026, 1, 15, 18, 30, 0, 0, loc),
	}
	now := time.Date(2026, 3, 2, 10, 0, 0, 0, loc)
	got := normalizeSched(now, old)

	// All times should be on March 2
	for _, tt := range []struct {
		name string
		t    time.Time
		h, m int
	}{
		{"dawn", got.CivilTwBegin, 5, 30},
		{"sunrise", got.Sunrise, 6, 0},
		{"sunset", got.Sunset, 18, 0},
		{"dusk", got.CivilTwEnd, 18, 30},
	} {
		if tt.t.Month() != 3 || tt.t.Day() != 2 {
			t.Errorf("%s: date = %s, want 2026-03-02", tt.name, tt.t.Format("2006-01-02"))
		}
		if tt.t.Hour() != tt.h || tt.t.Minute() != tt.m {
			t.Errorf("%s: time = %s, want %02d:%02d", tt.name, tt.t.Format("15:04"), tt.h, tt.m)
		}
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
