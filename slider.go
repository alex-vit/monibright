//go:build windows

package main

import (
	"fmt"
	"log"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procLoadCursorW         = user32.NewProc("LoadCursorW")
	procMoveWindow          = user32.NewProc("MoveWindow")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
)

var modComctl32 = syscall.NewLazyDLL("comctl32.dll")
var procInitCommonControlsEx = modComctl32.NewProc("InitCommonControlsEx")

var modShell32 = syscall.NewLazyDLL("shell32.dll")
var procSHAppBarMessage = modShell32.NewProc("SHAppBarMessage")

var modGdi32 = syscall.NewLazyDLL("gdi32.dll")
var (
	procCreateSolidBrush = modGdi32.NewProc("CreateSolidBrush")
	procSetBkColor       = modGdi32.NewProc("SetBkColor")
	procSetTextColor     = modGdi32.NewProc("SetTextColor")
)

const (
	WS_POPUP         = 0x80000000
	WS_BORDER        = 0x00800000
	WS_CHILD         = 0x40000000
	WS_VISIBLE       = 0x10000000
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_TOPMOST    = 0x00000008

	SW_SHOW = 5
	SW_HIDE = 0

	WM_ACTIVATE = 0x0006
	WM_DESTROY  = 0x0002
	WM_HSCROLL  = 0x0114

	WM_APP          = 0x8000
	wmShowSlider    = WM_APP + 1
	wmSyncSlider    = WM_APP + 2
	wmSyncColorTemp = WM_APP + 3

	WA_INACTIVE = 0

	SB_THUMBTRACK = 5
	SB_ENDSCROLL  = 8

	TBM_SETRANGE    = 0x0406
	TBM_SETPOS      = 0x0405
	TBM_GETPOS      = 0x0400
	TBM_SETPAGESIZE = 0x0415

	TBS_HORZ    = 0x0000
	TBS_NOTICKS = 0x0010

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	SS_RIGHT  = 0x0002
	SS_NOTIFY = 0x0100

	WM_COMMAND = 0x0111

	ICC_BAR_CLASSES = 0x00000004

	WM_CTLCOLORSTATIC = 0x0138

	wmSyncAutoToggle = WM_APP + 4

	WM_POWERBROADCAST      = 0x0218
	PBT_APMRESUMEAUTOMATIC = 0x0012

	// Win32 colors are 0x00BBGGRR
	sliderBgColor   = 0x00202020 // #202020 dark panel
	sliderTextColor = 0x00DEDEDE // #DEDEDE light text
	autoOffColor    = 0x00888888 // #888888 dimmed gray
	autoOnColor     = 0x004EA5FF // #FFA54E warm amber (BGR)
)

var (
	sliderHWND      uintptr
	sliderTrackHWND uintptr
	sliderPctHWND   uintptr
	sliderReady     = make(chan struct{})
	sliderWndProcCB uintptr
	brightnessReqs  = make(chan int, 1)
	sliderBgBrush   uintptr
	sliderDragging  bool

	tempTrackHWND    uintptr
	tempValueHWND    uintptr
	autoToggleHWND   uintptr
	colorTempReqs    = make(chan int, 1)
	tempDragging     bool
	currentColorTemp = 6500
	lastManualTemp   = 6500
	animateStop      chan struct{}
)

type sliderPoint struct{ X, Y int32 }
type sliderRect struct{ Left, Top, Right, Bottom int32 }

type appBarData struct {
	CbSize           uint32
	HWND             uintptr
	UCallbackMessage uint32
	UEdge            uint32
	Rc               sliderRect
	LParam           int32
}

const ABM_GETTASKBARPOS = 5

type wndClassExW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type initCommonControlsEx struct {
	DwSize uint32
	DwICC  uint32
}

func runSlider() {
	runtime.LockOSThread()

	icc := initCommonControlsEx{
		DwSize: uint32(unsafe.Sizeof(initCommonControlsEx{})),
		DwICC:  ICC_BAR_CLASSES,
	}
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc))) //nolint:errcheck

	hInst, _, _ := procGetModuleHandleW.Call(0)
	hCursor, _, _ := procLoadCursorW.Call(0, 32512) // IDC_ARROW

	sliderWndProcCB = syscall.NewCallback(sliderWndProc)
	sliderBgBrush, _, _ = procCreateSolidBrush.Call(sliderBgColor)

	className, _ := syscall.UTF16PtrFromString("MoniBrightSlider")
	wc := wndClassExW{
		LpfnWndProc:   sliderWndProcCB,
		HInstance:     hInst,
		HCursor:       hCursor,
		HbrBackground: sliderBgBrush,
		LpszClassName: className,
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))) //nolint:errcheck

	empty, _ := syscall.UTF16PtrFromString("")
	offScreen := int32(-1000)
	sliderHWND, _, _ = procCreateWindowExW.Call(
		WS_EX_TOOLWINDOW|WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(empty)),
		WS_POPUP|WS_BORDER,
		uintptr(offScreen), uintptr(offScreen),
		260, 130,
		0, 0, hInst, 0,
	)
	if sliderHWND == 0 {
		log.Printf("slider: CreateWindowExW failed")
		return
	}

	staticClass, _ := syscall.UTF16PtrFromString("STATIC")
	trackbarClass, _ := syscall.UTF16PtrFromString("msctls_trackbar32")

	// --- Color temperature row (top) ---
	colorTempLabel, _ := syscall.UTF16PtrFromString("Temperature")
	procCreateWindowExW.Call( //nolint:errcheck
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(colorTempLabel)),
		WS_CHILD|WS_VISIBLE,
		8, 8, 85, 16,
		sliderHWND, 0, hInst, 0,
	)

	autoToggleText, _ := syscall.UTF16PtrFromString("Auto")
	autoToggleHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(autoToggleText)),
		WS_CHILD|WS_VISIBLE|SS_RIGHT|SS_NOTIFY,
		95, 8, 80, 16,
		sliderHWND, 0, hInst, 0,
	)

	initTempText, _ := syscall.UTF16PtrFromString("6500K")
	tempValueHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(initTempText)),
		WS_CHILD|WS_VISIBLE|SS_RIGHT,
		192, 8, 60, 16,
		sliderHWND, 0, hInst, 0,
	)

	tempTrackHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(trackbarClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|TBS_HORZ|TBS_NOTICKS,
		8, 28, 244, 30,
		sliderHWND, 0, hInst, 0,
	)

	// Range 3500–6500, page size 500, initial position 6500
	procSendMessageW.Call(tempTrackHWND, TBM_SETRANGE, 1, uintptr(6500<<16|3500)) //nolint:errcheck
	procSendMessageW.Call(tempTrackHWND, TBM_SETPAGESIZE, 0, 500)                 //nolint:errcheck
	procSendMessageW.Call(tempTrackHWND, TBM_SETPOS, 1, 6500)                     //nolint:errcheck

	// --- Brightness row (bottom) ---
	brightnessLabel, _ := syscall.UTF16PtrFromString("Brightness")
	procCreateWindowExW.Call( //nolint:errcheck
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(brightnessLabel)),
		WS_CHILD|WS_VISIBLE,
		8, 66, 120, 16,
		sliderHWND, 0, hInst, 0,
	)

	sliderPctHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|SS_RIGHT,
		190, 66, 62, 16,
		sliderHWND, 0, hInst, 0,
	)

	sliderTrackHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(trackbarClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|TBS_HORZ|TBS_NOTICKS,
		8, 86, 244, 30,
		sliderHWND, 0, hInst, 0,
	)

	// Range 0–100, page size 10
	procSendMessageW.Call(sliderTrackHWND, TBM_SETRANGE, 1, 100<<16) //nolint:errcheck
	procSendMessageW.Call(sliderTrackHWND, TBM_SETPAGESIZE, 0, 10)   //nolint:errcheck

	// Async brightness updater
	go func() {
		for level := range brightnessReqs {
			setBrightness(level)
		}
	}()

	// Async color temp updater
	go func() {
		for kelvin := range colorTempReqs {
			applyColorTemp(kelvin)
		}
	}()

	close(sliderReady)

	var m wmMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m))) //nolint:errcheck
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m))) //nolint:errcheck
	}
}

func sliderWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmSyncSlider:
		if !sliderDragging {
			procSendMessageW.Call(sliderTrackHWND, TBM_SETPOS, 1, wParam) //nolint:errcheck
			updatePctLabel(int(wParam))
		}
		return 0
	case wmSyncColorTemp:
		if !tempDragging {
			procSendMessageW.Call(tempTrackHWND, TBM_SETPOS, 1, wParam) //nolint:errcheck
			updateTempLabel(int(wParam))
		}
		return 0
	case wmSyncAutoToggle:
		updateAutoToggleText()
		return 0
	case wmShowSlider:
		cursorX := int32(int16(lParam & 0xFFFF))
		cursorY := int32(int16((lParam >> 16) & 0xFFFF))
		positionAndShow(hwnd, cursorX, cursorY)
		return 0
	case WM_COMMAND:
		if lParam == autoToggleHWND {
			handleAutoToggleClick()
		}
		return 0
	case WM_CTLCOLORSTATIC:
		if lParam == autoToggleHWND {
			if autoColorActive {
				procSetTextColor.Call(wParam, autoOnColor) //nolint:errcheck
			} else {
				procSetTextColor.Call(wParam, autoOffColor) //nolint:errcheck
			}
			procSetBkColor.Call(wParam, sliderBgColor) //nolint:errcheck
			return sliderBgBrush
		}
		procSetTextColor.Call(wParam, sliderTextColor) //nolint:errcheck
		procSetBkColor.Call(wParam, sliderBgColor)     //nolint:errcheck
		return sliderBgBrush
	case WM_ACTIVATE:
		if wParam&0xFFFF == WA_INACTIVE {
			procShowWindow.Call(hwnd, SW_HIDE) //nolint:errcheck
		}
		return 0
	case WM_HSCROLL:
		// Distinguish trackbars by lParam (child HWND).
		switch lParam {
		case tempTrackHWND:
			pos, _, _ := procSendMessageW.Call(tempTrackHWND, TBM_GETPOS, 0, 0)
			updateTempLabel(int(pos))
			code := wParam & 0xFFFF
			switch code {
			case SB_THUMBTRACK:
				tempDragging = true
				lastManualTemp = int(pos)
				stopAnimation()
				if autoColorActive {
					stopAutoColor()
					cfg.AutoColorEnabled = false
					saveConfig()
					updateAutoToggleText()
					log.Printf("auto color temp disabled (manual override)")
				}
				requestColorTemp(int(pos))
			case SB_ENDSCROLL:
				tempDragging = false
				lastManualTemp = int(pos)
				cfg.ManualTemp = int(pos)
				saveConfig()
				requestColorTemp(int(pos))
			}
		default:
			pos, _, _ := procSendMessageW.Call(sliderTrackHWND, TBM_GETPOS, 0, 0)
			updatePctLabel(int(pos))
			code := wParam & 0xFFFF
			switch code {
			case SB_THUMBTRACK:
				sliderDragging = true
				requestBrightness(int(pos))
			case SB_ENDSCROLL:
				sliderDragging = false
				requestBrightness(int(pos))
			}
		}
		return 0
	case WM_POWERBROADCAST:
		if wParam == PBT_APMRESUMEAUTOMATIC {
			log.Printf("wake detected, reapplying color temp %dK", currentColorTemp)
			go applyColorTemp(currentColorTemp)
		}
		return 1
	case WM_DESTROY:
		procPostQuitMessage.Call(0) //nolint:errcheck
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}
}

// sliderPosition computes the popup position given cursor coordinates,
// an optional taskbar rect, and screen dimensions.
func sliderPosition(cursorX, cursorY int32, taskbar *sliderRect, screenW, screenH int32) (x, y int32) {
	const winW, winH int32 = 260, 130
	const gap int32 = 4

	if taskbar != nil {
		tbH := taskbar.Bottom - taskbar.Top
		tbW := taskbar.Right - taskbar.Left
		if tbH < tbW {
			// Horizontal taskbar (bottom or top).
			x = cursorX - winW/2
			if taskbar.Top == 0 {
				y = taskbar.Bottom + gap
			} else {
				y = taskbar.Top - winH - gap
			}
		} else {
			// Vertical taskbar (left or right).
			y = cursorY - winH/2
			if taskbar.Left == 0 {
				x = taskbar.Right + gap
			} else {
				x = taskbar.Left - winW - gap
			}
		}
	} else {
		// Fallback: position above cursor.
		x = cursorX - winW/2
		y = cursorY - winH - gap
	}

	// Clamp to screen.
	if x < 0 {
		x = 0
	}
	if x+winW > screenW {
		x = screenW - winW
	}
	if y < 0 {
		y = 0
	}
	if y+winH > screenH {
		y = screenH - winH
	}
	return
}

func positionAndShow(hwnd uintptr, cursorX, cursorY int32) {
	_, cur, _, err := allMonitors[0].GetBrightness()
	if err != nil {
		log.Printf("slider: GetBrightness: %v", err)
		cur = 50
	}
	procSendMessageW.Call(sliderTrackHWND, TBM_SETPOS, 1, uintptr(cur)) //nolint:errcheck
	updatePctLabel(cur)

	// Sync color temp trackbar to current value.
	procSendMessageW.Call(tempTrackHWND, TBM_SETPOS, 1, uintptr(currentColorTemp)) //nolint:errcheck
	updateTempLabel(currentColorTemp)
	updateAutoToggleText()

	// Get taskbar position to anchor the slider above it (like volume flyout).
	abd := appBarData{CbSize: uint32(unsafe.Sizeof(appBarData{}))}
	ret, _, _ := procSHAppBarMessage.Call(ABM_GETTASKBARPOS, uintptr(unsafe.Pointer(&abd)))

	var taskbar *sliderRect
	if ret != 0 {
		taskbar = &abd.Rc
	}

	sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	x, y := sliderPosition(cursorX, cursorY, taskbar, int32(sw), int32(sh))

	procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), 260, 130, 1) //nolint:errcheck
	procSetForegroundWindow.Call(hwnd)                             //nolint:errcheck
	procShowWindow.Call(hwnd, SW_SHOW)                             //nolint:errcheck
}

// requestBrightness enqueues a brightness update, dropping any pending
// value so the goroutine always processes the latest position.
func requestBrightness(level int) {
	select {
	case <-brightnessReqs:
	default:
	}
	brightnessReqs <- level
}

func updatePctLabel(pct int) {
	text, _ := syscall.UTF16PtrFromString(fmt.Sprintf("%d%%", pct))
	procSetWindowTextW.Call(sliderPctHWND, uintptr(unsafe.Pointer(text))) //nolint:errcheck
}

// syncSlider posts the current brightness level to the slider window so it
// updates while on screen. Safe to call from any goroutine.
func syncSlider(level int) {
	select {
	case <-sliderReady:
	default:
		return // not yet initialized
	}
	if sliderHWND != 0 {
		procPostMessageW.Call(sliderHWND, wmSyncSlider, uintptr(level), 0) //nolint:errcheck
	}
}

func showSlider() {
	<-sliderReady
	var pt sliderPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))) //nolint:errcheck
	lp := uintptr(uint16(pt.X)) | uintptr(uint16(pt.Y))<<16
	procPostMessageW.Call(sliderHWND, wmShowSlider, 0, lp) //nolint:errcheck
}

func updateTempLabel(kelvin int) {
	text, _ := syscall.UTF16PtrFromString(fmt.Sprintf("%dK", kelvin))
	procSetWindowTextW.Call(tempValueHWND, uintptr(unsafe.Pointer(text))) //nolint:errcheck
}

// requestColorTemp enqueues a color temperature update, dropping any pending
// value so the goroutine always processes the latest position.
func requestColorTemp(kelvin int) {
	currentColorTemp = kelvin
	select {
	case <-colorTempReqs:
	default:
	}
	colorTempReqs <- kelvin
}

func updateAutoToggleText() {
	var label string
	if autoColorActive {
		label = "\u25CF Auto"
	} else {
		label = "Auto"
	}
	text, _ := syscall.UTF16PtrFromString(label)
	procSetWindowTextW.Call(autoToggleHWND, uintptr(unsafe.Pointer(text))) //nolint:errcheck
}

func handleAutoToggleClick() {
	if autoColorActive {
		from := currentColorTemp
		stopAutoColor()
		cfg.AutoColorEnabled = false
		saveConfig()
		updateAutoToggleText()
		animateColorTemp(from, lastManualTemp)
	} else {
		stopAnimation()
		from := currentColorTemp
		cfg.AutoColorEnabled = true
		saveConfig()
		updateAutoToggleText()
		go func() {
			startAutoColor(from)
			procPostMessageW.Call(sliderHWND, wmSyncAutoToggle, 0, 0) //nolint:errcheck
		}()
	}
}

func stopAnimation() {
	if animateStop != nil {
		close(animateStop)
		animateStop = nil
	}
}

// animateColorTemp starts a non-blocking animated transition (for disable path).
func animateColorTemp(from, to int) {
	stopAnimation()
	stop := make(chan struct{})
	animateStop = stop
	go func() {
		animateColorTempSync(from, to, stop)
	}()
}

// animationFrames computes the number of transition frames for a color temp
// change from one value to another.
func animationFrames(from, to int) int {
	const maxFrames = 50
	const minFrames = 5
	const fullRange = 3000

	dist := from - to
	if dist < 0 {
		dist = -dist
	}
	frames := maxFrames * dist / fullRange
	if frames < minFrames {
		frames = minFrames
	}
	return frames
}

// animateColorTempSync runs an eased color temp transition, blocking until
// complete or the stop channel is closed. Duration scales with distance:
// full range (3000K) = 1s, smaller distances proportionally less.
func animateColorTempSync(from, to int, stop <-chan struct{}) {
	const frameDur = 20 * time.Millisecond

	frames := animationFrames(from, to)
	for i := 1; i <= frames; i++ {
		select {
		case <-stop:
			return
		default:
		}
		t := float64(i) / float64(frames)
		e := easeInOutCubic(t)
		temp := from + int(e*float64(to-from))
		requestColorTemp(temp)
		syncColorTempSlider(temp)
		time.Sleep(frameDur)
	}
	requestColorTemp(to)
	syncColorTempSlider(to)
}

func easeInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := -2*t + 2
	return 1 - u*u*u/2
}
