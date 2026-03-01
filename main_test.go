package main

import "testing"

func TestNearestPreset(t *testing.T) {
	tests := []struct {
		input, want int
	}{
		{100, 100},
		{50, 50},
		{75, 80},
		{74, 70},
		{1, 10},
		{0, 10},
		{5, 10},
		{4, 10},
		{15, 20},
		{14, 10},
		{95, 100},
		{99, 100},
		{10, 10},
	}

	for _, tt := range tests {
		got := nearestPreset(tt.input)
		if got != tt.want {
			t.Errorf("nearestPreset(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
