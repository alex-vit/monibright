package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	autoColorActive bool
	autoColorStop   chan struct{}
	autoColorMu     sync.Mutex

	cachedSched    sunSchedule
	cachedSchedDay int // YearDay when last fetched, 0 = never
)

// detectLocation determines latitude/longitude from IP geolocation,
// falling back to a timezone-based lookup.
func detectLocation() (lat, lon float64, err error) {
	lat, lon, err = locationFromIP()
	if err != nil {
		log.Printf("autocolor: IP location failed: %v, trying timezone fallback", err)
		lat, lon, err = locationFromTimezone()
	}
	return
}

func locationFromIP() (lat, lon float64, err error) {
	req, err := http.NewRequest(http.MethodGet, "https://iplocation.info", nil) //nolint:noctx
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, err
	}
	if result.Lat == 0 && result.Lon == 0 {
		return 0, 0, errors.New("got zero coordinates")
	}
	return result.Lat, result.Lon, nil
}

func locationFromTimezone() (lat, lon float64, err error) {
	tz := time.Now().Location().String()
	coords, ok := tzCoords[tz]
	if !ok {
		return 0, 0, fmt.Errorf("unknown timezone %q", tz)
	}
	log.Printf("autocolor: using timezone fallback for %s", tz)
	return coords[0], coords[1], nil
}

// sunSchedule holds the parsed sun times for one day.
type sunSchedule struct {
	Sunrise      time.Time
	Sunset       time.Time
	CivilTwBegin time.Time // civil twilight begin (morning)
	CivilTwEnd   time.Time // civil twilight end (evening)
}

// fetchSunSchedule queries the sunrise-sunset.io API for the given coordinates.
func fetchSunSchedule(lat, lon float64) (sunSchedule, error) {
	url := fmt.Sprintf("https://api.sunrisesunset.io/json?lat=%f&lng=%f&date=today", lat, lon)
	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
	if err != nil {
		return sunSchedule{}, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return sunSchedule{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return sunSchedule{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct {
		Results struct {
			Sunrise      string `json:"sunrise"`
			Sunset       string `json:"sunset"`
			CivilTwBegin string `json:"civil_twilight_begin"`
			CivilTwEnd   string `json:"civil_twilight_end"`
		} `json:"results"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return sunSchedule{}, err
	}
	if result.Status != "OK" {
		return sunSchedule{}, fmt.Errorf("API status: %s", result.Status)
	}

	today := time.Now()
	parse := func(s string) (time.Time, error) {
		t, err := time.Parse("3:04:05 PM", s)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(today.Year(), today.Month(), today.Day(),
			t.Hour(), t.Minute(), t.Second(), 0, today.Location()), nil
	}

	sched := sunSchedule{}
	if sched.Sunrise, err = parse(result.Results.Sunrise); err != nil {
		return sunSchedule{}, fmt.Errorf("parse sunrise: %w", err)
	}
	if sched.Sunset, err = parse(result.Results.Sunset); err != nil {
		return sunSchedule{}, fmt.Errorf("parse sunset: %w", err)
	}
	if sched.CivilTwBegin, err = parse(result.Results.CivilTwBegin); err != nil {
		return sunSchedule{}, fmt.Errorf("parse civil_twilight_begin: %w", err)
	}
	if sched.CivilTwEnd, err = parse(result.Results.CivilTwEnd); err != nil {
		return sunSchedule{}, fmt.Errorf("parse civil_twilight_end: %w", err)
	}

	return sched, nil
}

// defaultSunSchedule returns a hardcoded fallback (sunrise 06:00, sunset 18:00).
func defaultSunSchedule() sunSchedule {
	today := time.Now()
	loc := today.Location()
	return sunSchedule{
		Sunrise:      time.Date(today.Year(), today.Month(), today.Day(), 6, 0, 0, 0, loc),
		Sunset:       time.Date(today.Year(), today.Month(), today.Day(), 18, 0, 0, 0, loc),
		CivilTwBegin: time.Date(today.Year(), today.Month(), today.Day(), 5, 30, 0, 0, loc),
		CivilTwEnd:   time.Date(today.Year(), today.Month(), today.Day(), 18, 30, 0, 0, loc),
	}
}

// interpolateTemp computes the color temperature for the given time based on
// the sun schedule, linearly blending between dayTemp and nightTemp during
// twilight transitions.
func interpolateTemp(now time.Time, sched sunSchedule, dayTemp, nightTemp int) int {
	// Evening ramp: sunset is the midpoint (~50% warm at sunset).
	// eveningStart = 2*sunset - civilTwEnd  (≈30 min before sunset)
	// eveningEnd   = civilTwEnd             (≈30 min after sunset)
	eveningStart := sched.Sunset.Add(sched.Sunset.Sub(sched.CivilTwEnd)) // 2*sunset - twEnd
	eveningEnd := sched.CivilTwEnd

	// Morning ramp: sunrise is the midpoint (symmetric).
	// morningStart = civilTwBegin              (≈30 min before sunrise)
	// morningEnd   = 2*sunrise - civilTwBegin  (≈30 min after sunrise)
	morningStart := sched.CivilTwBegin
	morningEnd := sched.Sunrise.Add(sched.Sunrise.Sub(sched.CivilTwBegin)) // 2*sunrise - twBegin

	switch {
	case now.Before(morningStart) || now.After(eveningEnd):
		return nightTemp
	case now.After(morningEnd) && now.Before(eveningStart):
		return dayTemp
	case !now.Before(morningStart) && !now.After(morningEnd):
		// Morning transition: night → day
		frac := float64(now.Sub(morningStart)) / float64(morningEnd.Sub(morningStart))
		temp := float64(nightTemp) + frac*float64(dayTemp-nightTemp)
		return roundTo100(int(temp))
	default:
		// Evening transition: day → night
		frac := float64(now.Sub(eveningStart)) / float64(eveningEnd.Sub(eveningStart))
		temp := float64(dayTemp) + frac*float64(nightTemp-dayTemp)
		return roundTo100(int(temp))
	}
}

func roundTo100(k int) int {
	return ((k + 50) / 100) * 100
}

// startAutoColor launches the auto color goroutine. Never blocks on HTTP.
// If animateFrom > 0, the first color temp change is animated from that value.
func startAutoColor(animateFrom int) {
	autoColorMu.Lock()
	defer autoColorMu.Unlock()

	if autoColorActive {
		return
	}

	autoColorStop = make(chan struct{})
	autoColorActive = true
	go runAutoColor(autoColorStop, animateFrom)
}

// stopAutoColor stops the auto color goroutine. Leaves the current color temp as-is.
func stopAutoColor() {
	autoColorMu.Lock()
	defer autoColorMu.Unlock()

	if !autoColorActive {
		return
	}
	close(autoColorStop)
	autoColorActive = false
}

func runAutoColor(stop chan struct{}, animateFrom int) {
	// Use cached or default schedule for immediate response.
	sched := defaultSunSchedule()
	if cachedSchedDay == time.Now().YearDay() {
		sched = cachedSched
	}

	// Animate/apply immediately — no HTTP wait.
	lastTemp := 0
	temp := interpolateTemp(time.Now(), sched, cfg.DayTemp, cfg.NightTemp)
	if animateFrom > 0 && animateFrom != temp {
		log.Printf("autocolor: %dK (animating from %dK)", temp, animateFrom)
		animateColorTempSync(animateFrom, temp, stop)
	} else {
		log.Printf("autocolor: %dK", temp)
		requestColorTemp(temp)
		syncColorTempSlider(temp)
	}
	lastTemp = temp

	// Background: detect location if needed, then refresh schedule.
	if cfg.Latitude == 0 && cfg.Longitude == 0 {
		lat, lon, err := detectLocation()
		if err != nil {
			log.Printf("autocolor: location detection failed: %v", err)
		} else {
			cfg.Latitude = lat
			cfg.Longitude = lon
			saveConfig()
			log.Printf("autocolor: detected location lat=%.2f lon=%.2f", lat, lon)
		}
	}
	if cfg.Latitude != 0 || cfg.Longitude != 0 {
		if freshSched, err := fetchSunSchedule(cfg.Latitude, cfg.Longitude); err != nil {
			log.Printf("autocolor: sun schedule fetch failed: %v", err)
		} else {
			sched = freshSched
			cachedSched = freshSched
			cachedSchedDay = time.Now().YearDay()
			log.Printf("autocolor: sunrise=%s sunset=%s tw_begin=%s tw_end=%s",
				sched.Sunrise.Format("15:04"), sched.Sunset.Format("15:04"),
				sched.CivilTwBegin.Format("15:04"), sched.CivilTwEnd.Format("15:04"))
		}
	}

	// Apply corrected temp if the fresh schedule changed it.
	temp = interpolateTemp(time.Now(), sched, cfg.DayTemp, cfg.NightTemp)
	if temp != lastTemp {
		log.Printf("autocolor: %dK (corrected after refresh)", temp)
		requestColorTemp(temp)
		syncColorTempSlider(temp)
		lastTemp = temp
	}

	lastDate := time.Now().YearDay()

	tick := func() {
		now := time.Now()

		// Re-fetch schedule on date change.
		if now.YearDay() != lastDate {
			newSched, err := fetchSunSchedule(cfg.Latitude, cfg.Longitude)
			if err != nil {
				log.Printf("autocolor: schedule re-fetch failed: %v", err)
			} else {
				sched = newSched
				cachedSched = newSched
				cachedSchedDay = now.YearDay()
				log.Printf("autocolor: new day, sunrise=%s sunset=%s",
					sched.Sunrise.Format("15:04"), sched.Sunset.Format("15:04"))
			}
			lastDate = now.YearDay()
		}

		temp := interpolateTemp(now, sched, cfg.DayTemp, cfg.NightTemp)
		if temp != lastTemp {
			log.Printf("autocolor: %dK", temp)
			requestColorTemp(temp)
			syncColorTempSlider(temp)
			lastTemp = temp
		}
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Printf("autocolor: stopped")
			return
		case <-ticker.C:
			tick()
		}
	}
}

// syncColorTempSlider posts a message to update the slider UI from any goroutine.
func syncColorTempSlider(kelvin int) {
	select {
	case <-sliderReady:
	default:
		return
	}
	if sliderHWND != 0 {
		procPostMessageW.Call(sliderHWND, wmSyncColorTemp, uintptr(kelvin), 0) //nolint:errcheck
	}
}

// tzCoords maps IANA timezone names to approximate city coordinates.
var tzCoords = map[string][2]float64{
	"America/New_York":               {40.71, -74.01},
	"America/Chicago":                {41.88, -87.63},
	"America/Denver":                 {39.74, -104.99},
	"America/Los_Angeles":            {34.05, -118.24},
	"America/Anchorage":              {61.22, -149.90},
	"Pacific/Honolulu":               {21.31, -157.86},
	"America/Phoenix":                {33.45, -112.07},
	"America/Toronto":                {43.65, -79.38},
	"America/Vancouver":              {49.28, -123.12},
	"America/Mexico_City":            {19.43, -99.13},
	"America/Sao_Paulo":              {-23.55, -46.63},
	"America/Argentina/Buenos_Aires": {-34.60, -58.38},
	"America/Bogota":                 {4.71, -74.07},
	"America/Lima":                   {-12.05, -77.04},
	"Europe/London":                  {51.51, -0.13},
	"Europe/Paris":                   {48.86, 2.35},
	"Europe/Berlin":                  {52.52, 13.41},
	"Europe/Madrid":                  {40.42, -3.70},
	"Europe/Rome":                    {41.90, 12.50},
	"Europe/Amsterdam":               {52.37, 4.90},
	"Europe/Brussels":                {50.85, 4.35},
	"Europe/Vienna":                  {48.21, 16.37},
	"Europe/Zurich":                  {47.38, 8.54},
	"Europe/Stockholm":               {59.33, 18.07},
	"Europe/Oslo":                    {59.91, 10.75},
	"Europe/Helsinki":                {60.17, 24.94},
	"Europe/Warsaw":                  {52.23, 21.01},
	"Europe/Moscow":                  {55.76, 37.62},
	"Europe/Istanbul":                {41.01, 28.98},
	"Europe/Athens":                  {37.98, 23.73},
	"Europe/Bucharest":               {44.43, 26.10},
	"Asia/Tokyo":                     {35.68, 139.69},
	"Asia/Shanghai":                  {31.23, 121.47},
	"Asia/Hong_Kong":                 {22.32, 114.17},
	"Asia/Singapore":                 {1.35, 103.82},
	"Asia/Kolkata":                   {28.61, 77.21},
	"Asia/Seoul":                     {37.57, 126.98},
	"Asia/Taipei":                    {25.03, 121.57},
	"Asia/Bangkok":                   {13.76, 100.50},
	"Asia/Dubai":                     {25.20, 55.27},
	"Asia/Jerusalem":                 {31.77, 35.22},
	"Australia/Sydney":               {-33.87, 151.21},
	"Australia/Melbourne":            {-37.81, 144.96},
	"Australia/Perth":                {-31.95, 115.86},
	"Pacific/Auckland":               {-36.85, 174.76},
	"Africa/Cairo":                   {30.04, 31.24},
	"Africa/Johannesburg":            {-26.20, 28.04},
	"Africa/Lagos":                   {6.52, 3.38},
}
