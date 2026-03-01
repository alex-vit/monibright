//go:build windows

package main

import (
	"fmt"
	"log"
	"runtime"
	"syscall"
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

	WM_APP        = 0x8000
	wmShowSlider  = WM_APP + 1
	wmSyncSlider  = WM_APP + 2

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

	SS_RIGHT = 0x0002

	ICC_BAR_CLASSES = 0x00000004

	WM_CTLCOLORSTATIC = 0x0138

	// Win32 colors are 0x00BBGGRR
	sliderBgColor   = 0x00202020 // #202020 dark panel
	sliderTextColor = 0x00DEDEDE // #DEDEDE light text
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
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))

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
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	empty, _ := syscall.UTF16PtrFromString("")
	offScreen := int32(-1000)
	sliderHWND, _, _ = procCreateWindowExW.Call(
		WS_EX_TOOLWINDOW|WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(empty)),
		WS_POPUP|WS_BORDER,
		uintptr(offScreen), uintptr(offScreen),
		260, 72,
		0, 0, hInst, 0,
	)
	if sliderHWND == 0 {
		log.Printf("slider: CreateWindowExW failed")
		return
	}

	staticClass, _ := syscall.UTF16PtrFromString("STATIC")
	brightnessLabel, _ := syscall.UTF16PtrFromString("Brightness")
	trackbarClass, _ := syscall.UTF16PtrFromString("msctls_trackbar32")

	// "Brightness" label
	procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(brightnessLabel)),
		WS_CHILD|WS_VISIBLE,
		8, 8, 120, 16,
		sliderHWND, 0, hInst, 0,
	)

	// Percentage label (right-aligned)
	sliderPctHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|SS_RIGHT,
		190, 8, 62, 16,
		sliderHWND, 0, hInst, 0,
	)

	// Trackbar
	sliderTrackHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(trackbarClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|TBS_HORZ|TBS_NOTICKS,
		8, 28, 244, 30,
		sliderHWND, 0, hInst, 0,
	)

	// Set trackbar range 0–100, page size 10 (click-track jumps by 10)
	procSendMessageW.Call(sliderTrackHWND, TBM_SETRANGE, 1, 100<<16)
	procSendMessageW.Call(sliderTrackHWND, TBM_SETPAGESIZE, 0, 10)

	// Async brightness updater — latest value wins, WndProc never blocks on DDC/CI
	go func() {
		for level := range brightnessReqs {
			setBrightness(level)
		}
	}()

	close(sliderReady)

	var m wmMsg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func sliderWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmSyncSlider:
		if !sliderDragging {
			procSendMessageW.Call(sliderTrackHWND, TBM_SETPOS, 1, wParam)
			updatePctLabel(int(wParam))
		}
		return 0
	case wmShowSlider:
		cursorX := int32(int16(lParam & 0xFFFF))
		cursorY := int32(int16((lParam >> 16) & 0xFFFF))
		positionAndShow(hwnd, cursorX, cursorY)
		return 0
	case WM_CTLCOLORSTATIC:
		procSetTextColor.Call(wParam, sliderTextColor)
		procSetBkColor.Call(wParam, sliderBgColor)
		return sliderBgBrush
	case WM_ACTIVATE:
		if wParam&0xFFFF == WA_INACTIVE {
			procShowWindow.Call(hwnd, SW_HIDE)
		}
		return 0
	case WM_HSCROLL:
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
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}
}

func positionAndShow(hwnd uintptr, cursorX, cursorY int32) {
	_, cur, _, err := allMonitors[0].GetBrightness()
	if err != nil {
		log.Printf("slider: GetBrightness: %v", err)
		cur = 50
	}
	procSendMessageW.Call(sliderTrackHWND, TBM_SETPOS, 1, uintptr(cur))
	updatePctLabel(int(cur))

	const winW, winH int32 = 260, 72
	const gap int32 = 4

	// Get taskbar position to anchor the slider above it (like volume flyout).
	abd := appBarData{CbSize: uint32(unsafe.Sizeof(appBarData{}))}
	ret, _, _ := procSHAppBarMessage.Call(ABM_GETTASKBARPOS, uintptr(unsafe.Pointer(&abd)))

	var x, y int32
	if ret != 0 {
		tbRC := abd.Rc
		tbH := tbRC.Bottom - tbRC.Top
		tbW := tbRC.Right - tbRC.Left
		if tbH < tbW {
			// Horizontal taskbar (bottom or top).
			x = cursorX - winW/2
			if tbRC.Top == 0 {
				// Top taskbar.
				y = tbRC.Bottom + gap
			} else {
				// Bottom taskbar.
				y = tbRC.Top - winH - gap
			}
		} else {
			// Vertical taskbar (left or right).
			y = cursorY - winH/2
			if tbRC.Left == 0 {
				// Left taskbar.
				x = tbRC.Right + gap
			} else {
				// Right taskbar.
				x = tbRC.Left - winW - gap
			}
		}
	} else {
		// Fallback: position above cursor.
		x = cursorX - winW/2
		y = cursorY - winH - gap
	}

	// Clamp to screen.
	sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	screenW, screenH := int32(sw), int32(sh)
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

	procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(winW), uintptr(winH), 1)
	procSetForegroundWindow.Call(hwnd)
	procShowWindow.Call(hwnd, SW_SHOW)
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
	procSetWindowTextW.Call(sliderPctHWND, uintptr(unsafe.Pointer(text)))
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
		procPostMessageW.Call(sliderHWND, wmSyncSlider, uintptr(level), 0)
	}
}

func showSlider() {
	<-sliderReady
	var pt sliderPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	lp := uintptr(uint16(pt.X)) | uintptr(uint16(pt.Y))<<16
	procPostMessageW.Call(sliderHWND, wmShowSlider, 0, lp)
}
