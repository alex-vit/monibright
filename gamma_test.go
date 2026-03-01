//go:build windows

package main

import "testing"

func TestKelvinToRGB(t *testing.T) {
	tests := []struct {
		name          string
		kelvin        int
		wantRMin      float64
		wantRMax      float64
		wantGMin      float64
		wantGMax      float64
		wantBMin      float64
		wantBMax      float64
	}{
		{"6500K neutral white", 6500, 0.95, 1.0, 0.95, 1.0, 0.95, 1.0},
		{"2700K warm", 2700, 0.95, 1.0, 0.50, 0.70, 0.10, 0.40},
		{"1000K very low", 1000, 0.0, 1.0, 0.0, 1.0, 0.0, 0.01},
		{"10000K very high", 10000, 0.5, 1.0, 0.5, 1.0, 0.95, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b := kelvinToRGB(tt.kelvin)
			if r < tt.wantRMin || r > tt.wantRMax {
				t.Errorf("r = %.4f, want [%.2f, %.2f]", r, tt.wantRMin, tt.wantRMax)
			}
			if g < tt.wantGMin || g > tt.wantGMax {
				t.Errorf("g = %.4f, want [%.2f, %.2f]", g, tt.wantGMin, tt.wantGMax)
			}
			if b < tt.wantBMin || b > tt.wantBMax {
				t.Errorf("b = %.4f, want [%.2f, %.2f]", b, tt.wantBMin, tt.wantBMax)
			}
		})
	}
}

func TestKelvinToRGBClamped(t *testing.T) {
	for _, kelvin := range []int{1000, 2000, 3500, 5000, 6500, 8000, 10000} {
		r, g, b := kelvinToRGB(kelvin)
		if r < 0 || r > 1 || g < 0 || g > 1 || b < 0 || b > 1 {
			t.Errorf("kelvinToRGB(%d) = (%.4f, %.4f, %.4f), channels must be [0,1]", kelvin, r, g, b)
		}
	}
}

func TestBuildGammaRamp(t *testing.T) {
	t.Run("6500K identity-ish", func(t *testing.T) {
		ramp := buildGammaRamp(6500)
		// At 6500K, RGB ≈ (1.0, 0.99, 0.98), so ramp should be within 3% of identity.
		for i := 1; i < 256; i++ {
			expected := float64(uint16(i) << 8)
			for ch := 0; ch < 3; ch++ {
				ratio := float64(ramp[ch][i]) / expected
				if ratio < 0.97 || ratio > 1.01 {
					t.Errorf("ramp[%d][%d] = %d, expected ~%.0f (ratio %.4f)",
						ch, i, ramp[ch][i], expected, ratio)
				}
			}
		}
	})

	t.Run("low K red > green > blue", func(t *testing.T) {
		ramp := buildGammaRamp(3500)
		// Check a mid-range index where channels differ clearly.
		i := 128
		if ramp[0][i] <= ramp[1][i] || ramp[1][i] <= ramp[2][i] {
			t.Errorf("at 3500K index %d: R=%d G=%d B=%d, expected R > G > B",
				i, ramp[0][i], ramp[1][i], ramp[2][i])
		}
	})

	t.Run("monotonically increasing", func(t *testing.T) {
		ramp := buildGammaRamp(4500)
		for ch := 0; ch < 3; ch++ {
			for i := 1; i < 256; i++ {
				if ramp[ch][i] < ramp[ch][i-1] {
					t.Errorf("ramp[%d][%d]=%d < ramp[%d][%d]=%d, expected monotonic increase",
						ch, i, ramp[ch][i], ch, i-1, ramp[ch][i-1])
					break
				}
			}
		}
	})
}

func TestKelvinToRGBMonotonic(t *testing.T) {
	// Blue channel should increase as temperature rises from 2000K to 10000K.
	_, _, prevB := kelvinToRGB(2000)
	for k := 2100; k <= 10000; k += 100 {
		_, _, b := kelvinToRGB(k)
		if b < prevB-0.01 { // small tolerance for float rounding
			t.Errorf("blue channel decreased from %d to %dK: %.4f -> %.4f", k-100, k, prevB, b)
		}
		if b > prevB {
			prevB = b
		}
	}

	// Red channel should decrease as temperature rises above 6600K.
	prevR, _, _ := kelvinToRGB(6600)
	for k := 6700; k <= 10000; k += 100 {
		r, _, _ := kelvinToRGB(k)
		if r > prevR+0.01 {
			t.Errorf("red channel increased from %d to %dK: %.4f -> %.4f", k-100, k, prevR, r)
		}
		prevR = r
	}

}
