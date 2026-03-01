//go:build windows

package main

import (
	"math"
	"testing"
)

func TestEaseInOutCubic(t *testing.T) {
	tests := []struct {
		name string
		t    float64
		want float64
	}{
		{"t=0", 0, 0},
		{"t=1", 1, 1},
		{"t=0.5 midpoint", 0.5, 0.5},
		{"t=0.25", 0.25, 0.0625},
		{"t=0.75", 0.75, 0.9375},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := easeInOutCubic(tt.t)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("easeInOutCubic(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestEaseInOutCubicMonotonic(t *testing.T) {
	prev := 0.0
	for i := 1; i <= 100; i++ {
		x := float64(i) / 100
		got := easeInOutCubic(x)
		if got < prev-1e-12 {
			t.Errorf("easeInOutCubic(%v) = %v < previous %v, not monotonic", x, got, prev)
		}
		prev = got
	}
}

func TestSliderPosition(t *testing.T) {
	const screenW, screenH int32 = 1920, 1080

	tests := []struct {
		name    string
		cursorX int32
		cursorY int32
		taskbar *sliderRect
		wantX   int32
		wantY   int32
	}{
		{
			name:    "bottom taskbar, cursor centered",
			cursorX: 960, cursorY: 1040,
			taskbar: &sliderRect{Left: 0, Top: 1040, Right: 1920, Bottom: 1080},
			wantX:   830, wantY: 906,
		},
		{
			name:    "top taskbar",
			cursorX: 960, cursorY: 20,
			taskbar: &sliderRect{Left: 0, Top: 0, Right: 1920, Bottom: 40},
			wantX:   830, wantY: 44,
		},
		{
			name:    "left taskbar",
			cursorX: 30, cursorY: 540,
			taskbar: &sliderRect{Left: 0, Top: 0, Right: 60, Bottom: 1080},
			wantX:   64, wantY: 475,
		},
		{
			name:    "right taskbar",
			cursorX: 1890, cursorY: 540,
			taskbar: &sliderRect{Left: 1860, Top: 0, Right: 1920, Bottom: 1080},
			wantX:   1596, wantY: 475,
		},
		{
			name:    "no taskbar fallback",
			cursorX: 960, cursorY: 540,
			taskbar: nil,
			wantX:   830, wantY: 406,
		},
		{
			name:    "clamp left edge",
			cursorX: 50, cursorY: 1040,
			taskbar: &sliderRect{Left: 0, Top: 1040, Right: 1920, Bottom: 1080},
			wantX:   0, wantY: 906,
		},
		{
			name:    "clamp right edge",
			cursorX: 1900, cursorY: 1040,
			taskbar: &sliderRect{Left: 0, Top: 1040, Right: 1920, Bottom: 1080},
			wantX:   1660, wantY: 906,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := sliderPosition(tt.cursorX, tt.cursorY, tt.taskbar, screenW, screenH)
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("sliderPosition(%d, %d, ...) = (%d, %d), want (%d, %d)",
					tt.cursorX, tt.cursorY, x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestSliderPositionClamped(t *testing.T) {
	const winW, winH int32 = 260, 130
	const screenW, screenH int32 = 1920, 1080

	// Test that result is always within screen bounds regardless of input.
	positions := [][2]int32{{0, 0}, {-100, -100}, {2000, 2000}, {960, 540}}
	taskbars := []*sliderRect{
		nil,
		{Left: 0, Top: 1040, Right: 1920, Bottom: 1080},
		{Left: 0, Top: 0, Right: 1920, Bottom: 40},
	}

	for _, pos := range positions {
		for _, tb := range taskbars {
			x, y := sliderPosition(pos[0], pos[1], tb, screenW, screenH)
			if x < 0 || x+winW > screenW || y < 0 || y+winH > screenH {
				t.Errorf("sliderPosition(%d, %d) = (%d, %d) out of screen bounds",
					pos[0], pos[1], x, y)
			}
		}
	}
}

func TestAnimationFrames(t *testing.T) {
	tests := []struct {
		name     string
		from, to int
		want     int
	}{
		{"full range", 6500, 3500, 50},
		{"half range", 6500, 5000, 25},
		{"small distance clamps to min", 6500, 6400, 5},
		{"zero distance clamps to min", 5000, 5000, 5},
		{"reverse direction", 3500, 6500, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := animationFrames(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("animationFrames(%d, %d) = %d, want %d", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
