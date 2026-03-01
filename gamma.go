//go:build windows

package main

import (
	"log"
	"math"
	"syscall"
	"unsafe"
)

var (
	procGetDC               = user32.NewProc("GetDC")
	procReleaseDC           = user32.NewProc("ReleaseDC")
	procSetDeviceGammaRamp  = modGdi32.NewProc("SetDeviceGammaRamp")
	procGetDeviceGammaRamp  = modGdi32.NewProc("GetDeviceGammaRamp")
)

// gammaRamp is a 3×256 array of uint16 values (R, G, B channels).
type gammaRamp [3][256]uint16

var savedRamp gammaRamp

// kelvinToRGB converts a color temperature (in Kelvin) to RGB multipliers
// in the range 0.0–1.0 using the Tanner Helland algorithm.
// At 6500K the result is (1, 1, 1). At 2700K it's roughly (1, 0.59, 0.20).
func kelvinToRGB(kelvin int) (r, g, b float64) {
	temp := float64(kelvin) / 100.0

	// Red
	if temp <= 66 {
		r = 1.0
	} else {
		r = 329.698727446 * math.Pow(temp-60, -0.1332047592) / 255.0
	}

	// Green
	if temp <= 66 {
		g = (99.4708025861*math.Log(temp) - 161.1195681661) / 255.0
	} else {
		g = 288.1221695283 * math.Pow(temp-60, -0.0755148492) / 255.0
	}

	// Blue
	if temp >= 66 {
		b = 1.0
	} else if temp <= 19 {
		b = 0.0
	} else {
		b = (138.5177312231*math.Log(temp-10) - 305.0447927307) / 255.0
	}

	r = math.Max(0, math.Min(1, r))
	g = math.Max(0, math.Min(1, g))
	b = math.Max(0, math.Min(1, b))
	return
}

// buildGammaRamp constructs a gamma ramp scaled by the RGB multipliers for
// the given color temperature.
func buildGammaRamp(kelvin int) gammaRamp {
	r, g, b := kelvinToRGB(kelvin)
	var ramp gammaRamp
	for i := 0; i < 256; i++ {
		val := uint16(i) << 8
		ramp[0][i] = uint16(float64(val) * r)
		ramp[1][i] = uint16(float64(val) * g)
		ramp[2][i] = uint16(float64(val) * b)
	}
	return ramp
}

// applyColorTemp builds a gamma ramp scaled by the RGB multipliers for the
// given color temperature and applies it via SetDeviceGammaRamp.
func applyColorTemp(kelvin int) {
	ramp := buildGammaRamp(kelvin)

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		log.Printf("gamma: GetDC failed")
		return
	}
	defer procReleaseDC.Call(0, hdc)

	ret, _, err := procSetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&ramp)))
	if ret == 0 {
		log.Printf("gamma: SetDeviceGammaRamp failed: %v", err)
	}
}

// saveGammaRamp captures the current gamma ramp so it can be restored on exit.
func saveGammaRamp() {
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		log.Printf("gamma: GetDC failed (save)")
		return
	}
	defer procReleaseDC.Call(0, hdc)

	ret, _, err := procGetDeviceGammaRamp.Call(hdc, uintptr(unsafe.Pointer(&savedRamp)))
	if ret == 0 {
		log.Printf("gamma: GetDeviceGammaRamp failed: %v", err)
	} else {
		log.Printf("gamma: saved baseline gamma ramp")
	}
}

// restoreGammaRamp restores the gamma ramp captured by saveGammaRamp.
func restoreGammaRamp() {
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		log.Printf("gamma: GetDC failed (restore)")
		return
	}
	defer procReleaseDC.Call(0, hdc)

	ret, _, err := syscall.SyscallN(procSetDeviceGammaRamp.Addr(), hdc, uintptr(unsafe.Pointer(&savedRamp)))
	if ret == 0 {
		log.Printf("gamma: restore SetDeviceGammaRamp failed: %v", err)
	} else {
		log.Printf("gamma: restored baseline gamma ramp")
	}
}
