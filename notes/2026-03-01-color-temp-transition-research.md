# Color Temperature Transition Research

Date: 2026-03-01

Research into how f.lux, Redshift, Gammastep, Windows Night Light, and LightBulb
handle color temperature transitions throughout the day.

## Summary of findings

The established tools strongly support the instinct that the transition should start
well before sunset and be roughly halfway done at twilight. Redshift/Gammastep make
this explicit: their transition band spans from +3 deg to -6 deg solar elevation,
meaning the transition is ~1/3 complete at the moment of sunset (0 deg). f.lux
starts its transition roughly an hour before sunset, so by twilight you're already
significantly warmed.

---

## 1. Redshift (and Gammastep, its Wayland fork)

These two share the same transition model. Gammastep is a fork of Redshift with
Wayland support; the core algorithm is identical.

### Solar-elevation model

Redshift does **not** use clock times by default. It uses the sun's current
elevation angle (degrees above/below horizon) to determine the transition state.

Three periods:
- **Day**: solar elevation > `elevation-high` (default **3 deg**)
- **Transition**: solar elevation between `elevation-low` and `elevation-high`
- **Night**: solar elevation < `elevation-low` (default **-6 deg**)

The default `-6 deg` for `elevation-low` **is civil dusk/dawn** (the end of civil
twilight). The default `+3 deg` for `elevation-high` means the transition starts
while the sun is still above the horizon.

### Interpolation

**Linear.** The source code (`src/redshift.c`) computes alpha as:

```c
alpha = (transition->low - elevation) / (transition->low - transition->high);
// elevation-low = -6, elevation-high = 3 => range of 9 degrees
```

Then the color setting is:
```c
result = (1.0 - alpha) * night + alpha * day;
```

No easing curve, just straight linear interpolation across the elevation range.

### What does this mean in wall-clock time?

The 9-degree band (from +3 to -6) translates to roughly **60-90 minutes** depending
on latitude and time of year (the sun moves ~0.25 deg/min near equinox at mid
latitudes, slower in summer at high latitudes).

At the moment of sunset (0 deg elevation), the transition is **1/3 complete**
(3 out of 9 degrees into the band). At the end of civil twilight (-6 deg), it
reaches 100%.

### Default temperatures

- Day: **5700K**
- Night: **3500K**

### Sunrise behavior

**Symmetric.** The same elevation thresholds apply in reverse. At civil dawn
(-6 deg, rising), the morning transition begins; at +3 deg, it completes. Same
duration as evening.

### Fade option

The `fade=1` setting (default on) adds a **smooth interpolation over a few seconds**
when the calculated target changes between polling intervals. This prevents visible
stepping. It does NOT control the overall transition duration (that's governed by
solar elevation).

### Manual override

Users can replace the solar model with explicit time ranges:
```
dawn-time=6:00-7:45
dusk-time=18:35-20:15
```

Sources:
- [Redshift man page](https://www.mankier.com/1/redshift)
- [Redshift config sample](https://github.com/jonls/redshift/blob/master/redshift.conf.sample)
- [Redshift source (src/redshift.c)](https://github.com/jonls/redshift/blob/master/src/redshift.c)
- [Redshift wiki: Configuration file](https://github.com/jonls/redshift/wiki/Configuration-file)
- [Gammastep man page](https://man.archlinux.org/man/extra/gammastep/gammastep.1.en)
- [Gammastep GitLab](https://gitlab.com/chinstrap/gammastep)

---

## 2. f.lux

### Three-period model

f.lux v4 uses three temperature settings instead of two:

| Period    | Default temp | When                                   |
|-----------|-------------|----------------------------------------|
| Daytime   | **6500K**   | Sun is up                              |
| Sunset    | **3400K**   | After sunset, before bedtime           |
| Bedtime   | **1900K**   | Before wake time (sleep hours)         |

The "Recommended Colors" preset ships with 6500K / 3400K / 1900K.

### Transition timing

f.lux uses solar position (location-based) but the details are closed-source.
Observable behavior from user reports:

- The sunset transition **starts roughly 1 hour before actual sunset** and completes
  around sunset or shortly after. During the twilight hours, the display is already
  at 3400K.
- After full dark, a second transition takes the display from 3400K down to 1900K
  over **30-45 minutes**, timed relative to the user's configured wake time
  (bedtime = wake time minus ~8 hours).
- Morning transition: from 1900K back up to 6500K at sunrise (or wake time if the
  user is awake before sunrise), duration approximately 1 hour.

### Transition speed options

| Mode              | Duration             | Notes                         |
|-------------------|---------------------|-------------------------------|
| **Fast**          | ~20 seconds         | Abrupt snap, at sunset/sunrise |
| **Classic f.lux** | ~60 minutes         | Fades to 3400K at sunset only  |
| **Natural Timing**| Several hours       | Follows solar curve, rounded/smooth transitions |
| **Very Fast**     | ~1-2 seconds        | Instant, for gamers            |

### Transition curve

"Natural Timing" mode uses a **rounded/smooth curve** (likely ease-in/ease-out or
sinusoidal — the f.lux graph shows rounded corners rather than sharp linear
transitions). Other modes appear to use simple linear fades over their respective
durations.

### Civil twilight

f.lux does not expose elevation thresholds to the user, but observable behavior
suggests it starts transitioning while the sun is still above the horizon (similar
to Redshift's +3 deg start). Reports of the transition beginning "about an hour
before twilight" at high latitudes confirm it uses solar elevation rather than
exact sunset time.

Sources:
- [f.lux FAQ](https://justgetflux.com/faq.html)
- [f.lux macOS quickstart](https://justgetflux.com/news/pages/macquickstart/)
- [f.lux v4 welcome](https://justgetflux.com/news/pages/v4/welcome/)
- [IrisTech f.lux review](https://iristech.co/f-lux-4-beta-review-windows/)
- [f.lux forum: transitions hour before/after twilight](https://forum.justgetflux.com/topic/1915/f-lux-transitions-hour-before-after-twilight-at-sunrise-sunset)
- [f.lux forum: custom transition duration](https://forum.justgetflux.com/topic/4624/suggestion-custom-transition-duration)
- [f.lux forum: preserve natural timing](https://forum.justgetflux.com/topic/8198/preserve-slow-natural-timing-sunset-transitions-regardless-of-wake-time)

---

## 3. Windows Night Light

### Scheduling

Two modes:
- **Sunset to sunrise**: uses location (latitude/longitude) to compute local
  sunset/sunrise times. Activates at sunset, deactivates at sunrise.
- **Custom schedule**: user picks explicit on/off times (HH:MM).

### Transition behavior

The transition is **near-instant** (~1-2 seconds) when toggling manually or when the
scheduled time fires. There is no gradual hour-long fade. Night Light snaps to the
target temperature with only a brief visual fade.

This is widely considered a design weakness compared to f.lux/Redshift. Multiple
reviews note the lack of a gradual transition.

### Temperature control

A single slider from "less warm" to "more warm" (no Kelvin values exposed).
Internally, this maps to a CCT range via the gamma ramp. Only one night temperature;
no bedtime/sunset split.

### No twilight awareness

Night Light triggers at the computed sunset instant (0 deg solar elevation). It does
**not** start early or use civil twilight. There is no configurable transition
duration or offset.

Sources:
- [Microsoft Support: Night Light](https://support.microsoft.com/en-us/windows/set-your-display-for-night-time-in-windows-18fe903a-e0a1-8326-4c68-fd23d7aaf136)
- [IrisTech Night Light review](https://iristech.co/night-light-review-windows/)
- [MakeUseOf: f.lux vs Night Light](https://www.makeuseof.com/tag/flux-vs-windows-10-night-light/)
- [PCWorld: Night Light in Creators Update](https://www.pcworld.com/article/406422/how-to-use-night-light-in-the-windows-10-creators-update.html)

---

## 4. LightBulb (bonus — open-source Windows alternative)

### Transition model

LightBulb uses clock-based transitions anchored to sunrise/sunset. The transition
period defaults to **40 minutes**.

### Transition offset

A percentage (0-100%) controls where sunset falls within the transition window:
- **0% (default)**: Night transition starts at sunset and extends 40 min after
- **50%**: Transition is centered on sunset (20 min before, 20 min after)
- **100%**: Transition ends at sunset (starts 40 min before)

The 50% offset matches the "halfway done at twilight" instinct.

### Sunrise behavior

Symmetric with sunset. Same duration, same offset model.

Sources:
- [LightBulb wiki: transition offset](https://github.com/Tyrrrz/LightBulb/wiki/How-the-transition-offset-works)
- [LightBulb GitHub](https://github.com/Tyrrrz/LightBulb)

---

## 5. Comparison table

| Feature | Redshift/Gammastep | f.lux v4 | Windows Night Light | LightBulb |
|---|---|---|---|---|
| **Transition trigger** | Solar elevation | Solar position (closed-source) | Sunset instant | Sunset + offset |
| **Transition starts** | +3 deg (before sunset) | ~1 hr before sunset | At sunset | Configurable offset |
| **Transition ends** | -6 deg (civil dusk) | At/shortly after sunset | At sunset (+1-2s) | Sunset + duration |
| **Evening duration** | ~60-90 min | ~60 min (natural timing) | ~1-2 seconds | ~40 min (default) |
| **Morning duration** | Same as evening | ~60 min | ~1-2 seconds | Same as evening |
| **Interpolation** | Linear (on elevation) | Smooth/rounded (natural) or linear | Step (near-instant) | Not documented |
| **Day temp default** | 5700K | 6500K | Not exposed | 6500K |
| **Night temp default** | 3500K | 3400K (sunset) / 1900K (bed) | Single slider | 3900K |
| **Twilight-aware** | Yes (civil twilight = -6 deg) | Yes (starts before sunset) | No | Optional via offset |
| **Open source** | Yes | No | No | Yes |

---

## 6. Civil / nautical / astronomical twilight

Standard solar elevation thresholds:

| Type | Elevation | Practical meaning |
|---|---|---|
| Sunset/sunrise | 0 deg | Sun touches horizon |
| **Civil twilight** | 0 to -6 deg | Enough light to see without artificial lighting |
| Nautical twilight | -6 to -12 deg | Horizon still visible at sea |
| Astronomical twilight | -12 to -18 deg | Sky too bright for faintest stars |

Redshift/Gammastep use **civil twilight** as their night threshold (-6 deg) by
default. No tool in this survey uses nautical or astronomical twilight. This makes
sense: civil twilight is the point where you'd naturally turn on room lights, which
is exactly when your screen's color temperature should match those room lights.

---

## 7. Implications for MoniBright

### Recommended model

Follow the Redshift/Gammastep approach — it's well-tested, open-source, and simple:

1. **Use solar elevation as the input**, not wall-clock time
2. **Default thresholds**: `elevation-high = 3`, `elevation-low = -6` (same as Redshift)
3. **Linear interpolation** between day temp and night temp based on position in
   the elevation band (proven simple and effective; no complaints about the curve)
4. The transition naturally starts before sunset and completes at civil dusk
5. Morning is symmetric — same thresholds, same duration

### Temperature defaults

Given MoniBright already has the gamma ramp infrastructure:
- **Day**: 6500K (identity — no ramp modification)
- **Night**: 3500K (matches Redshift default; f.lux's 3400K is close)
- Skip f.lux's three-period model for v1 (adds complexity with wake time, bedtime
  calculation). Two periods (day/night) is sufficient and matches Redshift.

### Answering the original instinct

> The transition should start early enough to be ~halfway done at twilight.

This **exactly** matches what Redshift does. With a +3 to -6 range, at sunset
(0 deg) you're 3/9 = 33% of the way through. At mid-civil-twilight (-3 deg) you're
6/9 = 67% through. The established tools validate this approach.

If you want to be literally 50% at sunset, you'd use symmetric thresholds like
`elevation-high = +3, elevation-low = -3` (a 6-degree band). But the standard -6
low is better because it gives a longer tail into civil twilight, which is when the
light is actually changing most rapidly outside.

### Solar elevation calculation

Need a Go library or algorithm for solar position. Options:
- Port the algorithm from Redshift's `solar.c` (public domain, ~200 lines)
- Use an existing Go package (search for `solar position` or `sun elevation`)
- Need only: `func SolarElevation(lat, lon float64, t time.Time) float64`

### Location input

- Simplest: hardcoded or config-file lat/lon (like Redshift)
- Better: Windows Location API or IP geolocation at startup
- f.lux asks for zip code; Redshift takes lat:lon on command line
