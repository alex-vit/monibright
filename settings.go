//go:build windows

package main

import (
	"log"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	procGetStockObject  = modGdi32.NewProc("GetStockObject")
	procEnableWindow    = user32.NewProc("EnableWindow")
	procSetFocus        = user32.NewProc("SetFocus")
	procIsWindow        = user32.NewProc("IsWindow")
	procDestroyWindow   = user32.NewProc("DestroyWindow")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
)

const (
	WS_OVERLAPPED = 0x00000000
	WS_CAPTION    = 0x00C00000
	WS_SYSMENU    = 0x00080000

	WS_TABSTOP    = 0x00010000
	WS_GROUP      = 0x00020000
	WS_EX_CLIENTEDGE = 0x00000200

	BS_GROUPBOX      = 0x00000007
	BS_AUTOCHECKBOX  = 0x00000003
	BS_DEFPUSHBUTTON = 0x00000001
	BS_PUSHBUTTON    = 0x00000000

	ES_NUMBER = 0x2000

	BM_GETCHECK = 0x00F0
	BM_SETCHECK = 0x00F1
	BST_CHECKED = 1

	DEFAULT_GUI_FONT = 17

	UDS_SETBUDDYINT = 0x0002
	UDS_ALIGNRIGHT  = 0x0004
	UDS_ARROWKEYS   = 0x0020
	UDS_NOTHOUSANDS = 0x0080

	UDM_SETRANGE32 = 0x046F
	UDM_SETPOS32   = 0x0471
	UDM_GETPOS32   = 0x0472
	UDM_SETBUDDY   = 0x0469

	WM_CLOSE      = 0x0010
	WM_SETFONT    = 0x0030
	WM_GETTEXT    = 0x000D
	WM_GETTEXTLENGTH = 0x000E

	COLOR_BTNFACE = 15

	IDOK     = 1
	IDCANCEL = 2

	WM_APP_SHOW_SETTINGS = WM_APP + 10
)

var (
	settingsHWND      uintptr
	settingsReady     = make(chan struct{})
	settingsWndProcCB uintptr

	// Control handles
	chkAutoColor  uintptr
	editDayTemp   uintptr
	editNightTemp uintptr
	udDayTemp     uintptr
	udNightTemp   uintptr
	chkAutostart  uintptr
	btnOK         uintptr
	btnCancel     uintptr
)

func runSettings() {
	runtime.LockOSThread()

	hInst, _, _ := procGetModuleHandleW.Call(0)
	hCursor, _, _ := procLoadCursorW.Call(0, 32512) // IDC_ARROW

	settingsWndProcCB = syscall.NewCallback(settingsWndProc)

	className, _ := syscall.UTF16PtrFromString("MoniBrightSettings")
	wc := wndClassExW{
		LpfnWndProc:   settingsWndProcCB,
		HInstance:     hInst,
		HCursor:       hCursor,
		HbrBackground: COLOR_BTNFACE + 1, // system brush
		LpszClassName: className,
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))) //nolint:errcheck

	windowTitle, _ := syscall.UTF16PtrFromString("Settings")
	settingsHWND, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		uintptr(WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU),
		0x80000000, // CW_USEDEFAULT
		0x80000000,
		370, 250,
		0, 0, hInst, 0,
	)
	if settingsHWND == 0 {
		log.Printf("settings: CreateWindowExW failed")
		return
	}

	hFont, _, _ := procGetStockObject.Call(DEFAULT_GUI_FONT)
	createSettingsControls(hInst, hFont)

	close(settingsReady)

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

func createSettingsControls(hInst, hFont uintptr) {
	buttonClass, _ := syscall.UTF16PtrFromString("BUTTON")
	staticClass, _ := syscall.UTF16PtrFromString("STATIC")
	editClass, _ := syscall.UTF16PtrFromString("EDIT")
	updownClass, _ := syscall.UTF16PtrFromString("msctls_updown32")

	setFont := func(hwnd uintptr) {
		procSendMessageW.Call(hwnd, WM_SETFONT, hFont, 1) //nolint:errcheck
	}

	// --- GroupBox: Auto Color Temperature ---
	groupText, _ := syscall.UTF16PtrFromString("Auto Color Temperature")
	groupBox, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(buttonClass)),
		uintptr(unsafe.Pointer(groupText)),
		WS_CHILD|WS_VISIBLE|BS_GROUPBOX,
		12, 10, 330, 130,
		settingsHWND, 0, hInst, 0,
	)
	setFont(groupBox)

	// Checkbox: Enable auto color temperature
	chkText, _ := syscall.UTF16PtrFromString("Enable auto color temperature")
	chkAutoColor, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(buttonClass)),
		uintptr(unsafe.Pointer(chkText)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX,
		28, 35, 220, 20,
		settingsHWND, 0, hInst, 0,
	)
	setFont(chkAutoColor)

	// Label: Day temperature (K):
	dayLabel, _ := syscall.UTF16PtrFromString("Day temperature (K):")
	dayLabelHWND, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(dayLabel)),
		WS_CHILD|WS_VISIBLE,
		28, 68, 150, 20,
		settingsHWND, 0, hInst, 0,
	)
	setFont(dayLabelHWND)

	// Edit: Day temp
	empty, _ := syscall.UTF16PtrFromString("")
	editDayTemp, _, _ = procCreateWindowExW.Call(
		WS_EX_CLIENTEDGE,
		uintptr(unsafe.Pointer(editClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_NUMBER,
		220, 65, 70, 23,
		settingsHWND, 0, hInst, 0,
	)
	setFont(editDayTemp)

	// UpDown: Day temp spinner
	udDayTemp, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(updownClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|UDS_SETBUDDYINT|UDS_ALIGNRIGHT|UDS_ARROWKEYS|UDS_NOTHOUSANDS,
		0, 0, 0, 0, // positioned automatically by buddy
		settingsHWND, 0, hInst, 0,
	)
	procSendMessageW.Call(udDayTemp, UDM_SETBUDDY, editDayTemp, 0)    //nolint:errcheck
	procSendMessageW.Call(udDayTemp, UDM_SETRANGE32, 2000, 6500)      //nolint:errcheck

	// Label: Night temperature (K):
	nightLabel, _ := syscall.UTF16PtrFromString("Night temperature (K):")
	nightLabelHWND, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(nightLabel)),
		WS_CHILD|WS_VISIBLE,
		28, 100, 150, 20,
		settingsHWND, 0, hInst, 0,
	)
	setFont(nightLabelHWND)

	// Edit: Night temp
	editNightTemp, _, _ = procCreateWindowExW.Call(
		WS_EX_CLIENTEDGE,
		uintptr(unsafe.Pointer(editClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_NUMBER,
		220, 97, 70, 23,
		settingsHWND, 0, hInst, 0,
	)
	setFont(editNightTemp)

	// UpDown: Night temp spinner
	udNightTemp, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(updownClass)),
		uintptr(unsafe.Pointer(empty)),
		WS_CHILD|WS_VISIBLE|UDS_SETBUDDYINT|UDS_ALIGNRIGHT|UDS_ARROWKEYS|UDS_NOTHOUSANDS,
		0, 0, 0, 0,
		settingsHWND, 0, hInst, 0,
	)
	procSendMessageW.Call(udNightTemp, UDM_SETBUDDY, editNightTemp, 0) //nolint:errcheck
	procSendMessageW.Call(udNightTemp, UDM_SETRANGE32, 2000, 6500)     //nolint:errcheck

	// --- Checkbox: Start with Windows (outside group) ---
	autostartText, _ := syscall.UTF16PtrFromString("Start with Windows")
	chkAutostart, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(buttonClass)),
		uintptr(unsafe.Pointer(autostartText)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX,
		12, 152, 200, 20,
		settingsHWND, 0, hInst, 0,
	)
	setFont(chkAutostart)

	// --- OK / Cancel buttons ---
	okText, _ := syscall.UTF16PtrFromString("OK")
	btnOK, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(buttonClass)),
		uintptr(unsafe.Pointer(okText)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON,
		168, 185, 80, 28,
		settingsHWND, IDOK, hInst, 0,
	)
	setFont(btnOK)

	cancelText, _ := syscall.UTF16PtrFromString("Cancel")
	btnCancel, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(buttonClass)),
		uintptr(unsafe.Pointer(cancelText)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		258, 185, 80, 28,
		settingsHWND, IDCANCEL, hInst, 0,
	)
	setFont(btnCancel)
}

func settingsWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_APP_SHOW_SETTINGS:
		loadSettingsValues()
		procShowWindow.Call(hwnd, SW_SHOW) //nolint:errcheck
		procSetForegroundWindow.Call(hwnd) //nolint:errcheck
		return 0

	case WM_COMMAND:
		id := wParam & 0xFFFF
		switch id {
		case IDOK:
			settingsOnOK(hwnd)
		case IDCANCEL:
			procShowWindow.Call(hwnd, SW_HIDE) //nolint:errcheck
		}
		return 0

	case WM_CLOSE:
		procShowWindow.Call(hwnd, SW_HIDE) //nolint:errcheck
		return 0

	case WM_DESTROY:
		procPostQuitMessage.Call(0) //nolint:errcheck
		return 0

	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}
}

func loadSettingsValues() {
	// Auto color checkbox
	if cfg.AutoColorEnabled {
		procSendMessageW.Call(chkAutoColor, BM_SETCHECK, BST_CHECKED, 0) //nolint:errcheck
	} else {
		procSendMessageW.Call(chkAutoColor, BM_SETCHECK, 0, 0) //nolint:errcheck
	}

	// Day/Night temp spinners
	procSendMessageW.Call(udDayTemp, UDM_SETPOS32, 0, uintptr(cfg.DayTemp))     //nolint:errcheck
	procSendMessageW.Call(udNightTemp, UDM_SETPOS32, 0, uintptr(cfg.NightTemp)) //nolint:errcheck

	// Autostart checkbox
	if isAutostartEnabled() {
		procSendMessageW.Call(chkAutostart, BM_SETCHECK, BST_CHECKED, 0) //nolint:errcheck
	} else {
		procSendMessageW.Call(chkAutostart, BM_SETCHECK, 0, 0) //nolint:errcheck
	}
}

func settingsOnOK(hwnd uintptr) {
	// Read values from controls
	dayResult, _, _ := procSendMessageW.Call(udDayTemp, UDM_GETPOS32, 0, 0)
	nightResult, _, _ := procSendMessageW.Call(udNightTemp, UDM_GETPOS32, 0, 0)
	dayTemp := int(dayResult)
	nightTemp := int(nightResult)

	// Validate: day temp >= night temp
	if dayTemp < nightTemp {
		log.Printf("settings: day temp (%d) < night temp (%d), swapping", dayTemp, nightTemp)
		dayTemp, nightTemp = nightTemp, dayTemp
	}

	// Clamp to valid range
	dayTemp = clamp(dayTemp, 2000, 6500)
	nightTemp = clamp(nightTemp, 2000, 6500)

	autoColorChecked, _, _ := procSendMessageW.Call(chkAutoColor, BM_GETCHECK, 0, 0)
	newAutoColor := autoColorChecked == BST_CHECKED

	autostartChecked, _, _ := procSendMessageW.Call(chkAutostart, BM_GETCHECK, 0, 0)
	newAutostart := autostartChecked == BST_CHECKED

	// Apply autostart change
	wasAutostart := isAutostartEnabled()
	if newAutostart != wasAutostart {
		if newAutostart {
			if err := autostartEnable(); err != nil {
				log.Printf("settings: failed to enable autostart: %v", err)
			}
		} else {
			if err := autostartDisable(); err != nil {
				log.Printf("settings: failed to disable autostart: %v", err)
			}
		}
		// Sync tray menu checkbox
		if mAutostart != nil {
			if newAutostart {
				mAutostart.Check()
			} else {
				mAutostart.Uncheck()
			}
		}
	}

	// Apply auto color changes
	wasAutoColor := cfg.AutoColorEnabled
	oldDayTemp := cfg.DayTemp
	oldNightTemp := cfg.NightTemp
	cfg.DayTemp = dayTemp
	cfg.NightTemp = nightTemp
	cfg.AutoColorEnabled = newAutoColor
	saveConfig()

	log.Printf("settings: saved (auto_color=%v day=%dK night=%dK autostart=%v)",
		newAutoColor, dayTemp, nightTemp, newAutostart)

	if newAutoColor && !wasAutoColor {
		// Turning on: start auto color
		go startAutoColor(currentColorTemp)
		syncAutoToggle()
	} else if !newAutoColor && wasAutoColor {
		// Turning off: stop auto color, restore manual temp
		from := currentColorTemp
		stopAutoColor()
		syncAutoToggle()
		go func() {
			animateColorTempSync(from, lastManualTemp, make(chan struct{}))
		}()
	} else if newAutoColor && (dayTemp != oldDayTemp || nightTemp != oldNightTemp) {
		// Temps changed while auto color is on: restart to pick up new values
		stopAutoColor()
		go startAutoColor(currentColorTemp)
		syncAutoToggle()
	}

	procShowWindow.Call(hwnd, SW_HIDE) //nolint:errcheck
}

// syncAutoToggle posts a message to the slider window to update the auto toggle text.
func syncAutoToggle() {
	select {
	case <-sliderReady:
	default:
		return
	}
	if sliderHWND != 0 {
		procPostMessageW.Call(sliderHWND, wmSyncAutoToggle, 0, 0) //nolint:errcheck
	}
}

func showSettings() {
	<-settingsReady
	procPostMessageW.Call(settingsHWND, WM_APP_SHOW_SETTINGS, 0, 0) //nolint:errcheck
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
