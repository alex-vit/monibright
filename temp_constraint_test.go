package main

import "testing"

func TestTempConstraint(t *testing.T) {
	tests := []struct {
		name       string
		day, night int
		dayChanged bool
		wantDay    int
		wantNight  int
	}{
		// Already valid — no change needed
		{"valid wide gap", 6500, 3500, true, 6500, 3500},
		{"valid exact gap", 5000, 4900, true, 5000, 4900},
		{"valid exact gap night changed", 5000, 4900, false, 5000, 4900},

		// Day lowered into night — night pushed down
		{"day equals night", 4000, 4000, true, 4000, 3900},
		{"day slightly below gap", 4000, 3950, true, 4000, 3900},
		{"day pushed low near floor", 3600, 3600, true, 3600, 3500},

		// Day pushed so low night would go below min — both clamped
		{"day at floor", 3500, 3500, true, 3600, 3500},
		{"day below floor gap", 3550, 3500, true, 3600, 3500},

		// Night raised into day — day pushed up
		{"night equals day", 4000, 4000, false, 4100, 4000},
		{"night slightly above gap", 4050, 4000, false, 4100, 4000},
		{"night raised near ceiling", 6400, 6400, false, 6500, 6400},

		// Night pushed so high day would exceed max — both clamped
		{"night at ceiling", 6500, 6500, false, 6500, 6400},
		{"night above ceiling gap", 6450, 6450, false, 6500, 6400},

		// Out-of-range inputs get clamped
		{"day above max", 7000, 3500, true, 6500, 3500},
		{"night below min", 6500, 2000, false, 6500, 3500},
		{"both out of range", 8000, 1000, true, 6500, 3500},

		// Manual text entry: typed garbage values (focus loss scenarios)
		{"typed day below range", 2999, 3500, true, 3600, 3500},
		{"typed night above range", 6500, 6666, false, 6500, 6400},
		{"typed day below night inverted", 2999, 6666, true, 3600, 3500},
		{"typed both inverted in range", 3800, 5000, true, 3800, 3700},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDay, gotNight := enforceTempConstraint(tt.day, tt.night, tt.dayChanged)
			if gotDay != tt.wantDay || gotNight != tt.wantNight {
				t.Errorf("enforceTempConstraint(%d, %d, dayChanged=%v) = (%d, %d), want (%d, %d)",
					tt.day, tt.night, tt.dayChanged, gotDay, gotNight, tt.wantDay, tt.wantNight)
			}
			// Invariant must always hold
			if gotDay < gotNight+tempGap {
				t.Errorf("invariant violated: day=%d < night=%d + %d", gotDay, gotNight, tempGap)
			}
			if gotDay < tempMin || gotDay > tempMax {
				t.Errorf("day %d out of range [%d, %d]", gotDay, tempMin, tempMax)
			}
			if gotNight < tempMin || gotNight > tempMax {
				t.Errorf("night %d out of range [%d, %d]", gotNight, tempMin, tempMax)
			}
		})
	}
}
