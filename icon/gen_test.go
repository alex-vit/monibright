package icon

import (
	"image/color"
	"testing"
)

func TestLerpColor(t *testing.T) {
	a := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	b := color.NRGBA{R: 200, G: 100, B: 50, A: 255}

	tests := []struct {
		name string
		t    float64
		want color.NRGBA
	}{
		{"t=0 returns first", 0, color.NRGBA{R: 0, G: 0, B: 0, A: 255}},
		{"t=1 returns second", 1, color.NRGBA{R: 200, G: 100, B: 50, A: 255}},
		{"t=0.5 midpoint", 0.5, color.NRGBA{R: 100, G: 50, B: 25, A: 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lerpColor(a, b, tt.t)
			if got != tt.want {
				t.Errorf("lerpColor(a, b, %v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestSunColor(t *testing.T) {
	tests := []struct {
		name string
		t    float64
		want color.NRGBA
	}{
		{"t=0 returns colorLow", 0, colorLow},
		{"t=0.5 returns colorMid", 0.5, colorMid},
		{"t=1 returns colorHigh", 1, colorHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sunColor(tt.t)
			if got != tt.want {
				t.Errorf("sunColor(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	t.Run("valid ICO header", func(t *testing.T) {
		data := Generate(50)
		if len(data) < 6 {
			t.Fatalf("Generate(50) returned %d bytes, expected at least 6", len(data))
		}
		// ICO header: reserved=0, type=1 (little-endian).
		if data[0] != 0 || data[1] != 0 {
			t.Errorf("reserved bytes = %x %x, want 0 0", data[0], data[1])
		}
		if data[2] != 1 || data[3] != 0 {
			t.Errorf("image type = %x %x, want 1 0 (ICO)", data[2], data[3])
		}
		// 2 images (16px + 32px).
		if data[4] != 2 || data[5] != 0 {
			t.Errorf("image count = %x %x, want 2 0", data[4], data[5])
		}
	})

	t.Run("non-empty output", func(t *testing.T) {
		data := Generate(0)
		if len(data) == 0 {
			t.Error("Generate(0) returned empty")
		}
		data = Generate(100)
		if len(data) == 0 {
			t.Error("Generate(100) returned empty")
		}
	})

	t.Run("clamps out of range", func(t *testing.T) {
		low := Generate(-10)
		zero := Generate(0)
		if len(low) != len(zero) {
			t.Errorf("Generate(-10) len=%d, Generate(0) len=%d, expected same", len(low), len(zero))
		}
		high := Generate(200)
		hundred := Generate(100)
		if len(high) != len(hundred) {
			t.Errorf("Generate(200) len=%d, Generate(100) len=%d, expected same", len(high), len(hundred))
		}
	})
}
