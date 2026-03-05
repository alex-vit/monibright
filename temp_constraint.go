package main

const (
	tempMin = 3500
	tempMax = 6500
	tempGap = 100
)

// enforceTempConstraint ensures day >= night + tempGap.
// dayChanged indicates which value the user modified; the other
// value is adjusted to maintain the invariant. Both values are
// clamped to [tempMin, tempMax].
func enforceTempConstraint(day, night int, dayChanged bool) (int, int) {
	day = clamp(day, tempMin, tempMax)
	night = clamp(night, tempMin, tempMax)

	if day >= night+tempGap {
		return day, night
	}

	if dayChanged {
		night = day - tempGap
		if night < tempMin {
			night = tempMin
			day = tempMin + tempGap
		}
	} else {
		day = night + tempGap
		if day > tempMax {
			day = tempMax
			night = tempMax - tempGap
		}
	}

	return day, night
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
