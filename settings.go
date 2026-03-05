//go:build windows

package main

import (
	"log"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/alex-vit/monibright/icon"
)

var (
	procGetStockObject            = modGdi32.NewProc("GetStockObject")
	procEnableWindow              = user32.NewProc("EnableWindow")
	procSetFocus                  = user32.NewProc("SetFocus")
	procIsWindow                  = user32.NewProc("IsWindow")
	procDestroyWindow             = user32.NewProc("DestroyWindow")
	procIsWindowVisible           = user32.NewProc("IsWindowVisible")
	procCreateIconFromResourceEx  = user32.NewProc("CreateIconFromResourceEx")
	procGetWindowRect             = user32.NewProc("GetWindowRect")
	procSetWindowPos              = user32.NewProc("SetWindowPos")
	procSetWindowLongPtrW         = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW           = user32.NewProc("CallWindowProcW")
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

	WM_NOTIFY      = 0x004E
	UDN_DELTAPOS   = 0xFFFFFD2E // -722 as uint32
	EN_KILLFOCUS   = 0x0200

	WM_KEYDOWN   = 0x0100
	WM_CHAR      = 0x0102
	VK_TAB       = 0x09
	VK_RETURN    = 0x0D
	EM_SETSEL    = 0x00B1
	GWLP_WNDPROC = ^uintptr(3) // -4
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
	btnOK             uintptr
	btnCancel         uintptr
	origDayEditProc   uintptr
	origNightEditProc uintptr
)

type nmhdr struct {
	HwndFrom uintptr
	IdFrom   uintptr
	Code     uint32
	_        uint32 // padding
}

type nmUpDown struct {
	Hdr    nmhdr
	IPos   int32
	IDelta int32
}

// hIconFromICO extracts the first image from raw ICO file bytes and creates an HICON.
func hIconFromICO(icoData []byte, cx, cy int) uintptr {
	if len(icoData) < 22 { // 6-byte header + 16-byte directory entry minimum
		return 0
	}
	// ICO directory entry: bytes 14-17 = image size, bytes 18-21 = data offset
	size := uint32(icoData[14]) | uint32(icoData[15])<<8 | uint32(icoData[16])<<16 | uint32(icoData[17])<<24
	offset := uint32(icoData[18]) | uint32(icoData[19])<<8 | uint32(icoData[20])<<16 | uint32(icoData[21])<<24
	if int(offset+size) > len(icoData) {
		return 0
	}
	h, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&icoData[offset])),
		uintptr(size),
		1,          // fIcon = TRUE
		0x00030000, // version
		uintptr(cx), uintptr(cy),
		0, // flags
	)
	return h
}

const (
	WM_SETICON   = 0x0080
	ICON_SMALL   = 0
	ICON_BIG     = 1
	SWP_NOSIZE   = 0x0001
	SWP_NOZORDER = 0x0004
)

func runSettings() {
	runtime.LockOSThread()

	hInst, _, _ := procGetModuleHandleW.Call(0)
	hCursor, _, _ := procLoadCursorW.Call(0, 32512) // IDC_ARROW

	settingsWndProcCB = syscall.NewCallback(settingsWndProc)

	hIconBig := hIconFromICO(icon.Data, 32, 32)
	hIconSm := hIconFromICO(icon.Data, 16, 16)

	className, _ := syscall.UTF16PtrFromString("MoniBrightSettings")
	wc := wndClassExW{
		LpfnWndProc:   settingsWndProcCB,
		HInstance:     hInst,
		HCursor:       hCursor,
		HbrBackground: COLOR_BTNFACE + 1, // system brush
		LpszClassName: className,
		HIcon:         hIconBig,
		HIconSm:       hIconSm,
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
		370, 290,
		0, 0, hInst, 0,
	)
	if settingsHWND == 0 {
		log.Printf("settings: CreateWindowExW failed")
		return
	}

	// Center on screen
	centerWindow(settingsHWND)

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
		12, 10, 330, 148,
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
	procSendMessageW.Call(udDayTemp, UDM_SETRANGE32, 3500, 6500)      //nolint:errcheck

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
	procSendMessageW.Call(udNightTemp, UDM_SETRANGE32, 3500, 6500)     //nolint:errcheck

	// Hint text
	hintText, _ := syscall.UTF16PtrFromString("3500K warm \u00B7 4000K neutral \u00B7 5000K cool \u00B7 6500K daylight")
	hintHWND, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(hintText)),
		WS_CHILD|WS_VISIBLE,
		28, 125, 300, 16,
		settingsHWND, 0, hInst, 0,
	)
	setFont(hintHWND)

	// --- Checkbox: Start with Windows (outside group) ---
	autostartText, _ := syscall.UTF16PtrFromString("Start with Windows")
	chkAutostart, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(buttonClass)),
		uintptr(unsafe.Pointer(autostartText)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX,
		12, 170, 200, 20,
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
		168, 203, 80, 28,
		settingsHWND, IDOK, hInst, 0,
	)
	setFont(btnOK)

	cancelText, _ := syscall.UTF16PtrFromString("Cancel")
	btnCancel, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(buttonClass)),
		uintptr(unsafe.Pointer(cancelText)),
		WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,
		258, 203, 80, 28,
		settingsHWND, IDCANCEL, hInst, 0,
	)
	setFont(btnCancel)

	// Subclass edit controls to handle Tab and Enter
	editSubProcCB := syscall.NewCallback(editSubProc)
	origDayEditProc, _, _ = procSetWindowLongPtrW.Call(editDayTemp, uintptr(GWLP_WNDPROC), editSubProcCB)
	origNightEditProc, _, _ = procSetWindowLongPtrW.Call(editNightTemp, uintptr(GWLP_WNDPROC), editSubProcCB)
}

func settingsWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_APP_SHOW_SETTINGS:
		loadSettingsValues()
		centerWindow(hwnd)
		procShowWindow.Call(hwnd, SW_SHOW) //nolint:errcheck
		procSetForegroundWindow.Call(hwnd) //nolint:errcheck
		return 0

	case WM_NOTIFY:
		nm := (*nmUpDown)(unsafe.Pointer(lParam))
		if nm.Hdr.Code == UDN_DELTAPOS {
			if nm.Hdr.HwndFrom == udDayTemp || nm.Hdr.HwndFrom == udNightTemp {
				delta := 100
				if nm.IDelta < 0 {
					delta = -100
				}
				proposed := int(nm.IPos) + delta
				dayChanged := nm.Hdr.HwndFrom == udDayTemp

				var day, night int
				if dayChanged {
					day = proposed
					nightPos, _, _ := procSendMessageW.Call(udNightTemp, UDM_GETPOS32, 0, 0)
					night = int(nightPos)
				} else {
					dayPos, _, _ := procSendMessageW.Call(udDayTemp, UDM_GETPOS32, 0, 0)
					day = int(dayPos)
					night = proposed
				}

				day, night = enforceTempConstraint(day, night, dayChanged)
				procSendMessageW.Call(udDayTemp, UDM_SETPOS32, 0, uintptr(day))     //nolint:errcheck
				procSendMessageW.Call(udNightTemp, UDM_SETPOS32, 0, uintptr(night)) //nolint:errcheck
				return 1 // cancel default handling; we set values manually
			}
		}
		return 0

	case WM_COMMAND:
		notifyCode := (wParam >> 16) & 0xFFFF
		if notifyCode == EN_KILLFOCUS && (lParam == editDayTemp || lParam == editNightTemp) {
			enforceSettingsConstraints(lParam == editDayTemp)
			return 0
		}
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
	dayTemp = clamp(dayTemp, 3500, 6500)
	nightTemp = clamp(nightTemp, 3500, 6500)

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

// enforceSettingsConstraints reads both spinners, applies range + gap constraints,
// and writes corrected values back. Called on edit field focus loss.
func enforceSettingsConstraints(dayChanged bool) {
	dayPos, _, _ := procSendMessageW.Call(udDayTemp, UDM_GETPOS32, 0, 0)
	nightPos, _, _ := procSendMessageW.Call(udNightTemp, UDM_GETPOS32, 0, 0)
	day, night := enforceTempConstraint(int(dayPos), int(nightPos), dayChanged)
	procSendMessageW.Call(udDayTemp, UDM_SETPOS32, 0, uintptr(day))     //nolint:errcheck
	procSendMessageW.Call(udNightTemp, UDM_SETPOS32, 0, uintptr(night)) //nolint:errcheck
}

func editSubProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_KEYDOWN:
		switch wParam {
		case VK_TAB:
			if hwnd == editDayTemp {
				procSetFocus.Call(editNightTemp) //nolint:errcheck
			} else {
				procSetFocus.Call(chkAutostart) //nolint:errcheck
			}
			return 0
		case VK_RETURN:
			enforceSettingsConstraints(hwnd == editDayTemp)
			procSendMessageW.Call(hwnd, EM_SETSEL, 0, ^uintptr(0)) //nolint:errcheck
			return 0
		}
	case WM_CHAR:
		if wParam == '\t' || wParam == '\r' || wParam == '\n' {
			return 0 // eat to prevent beep
		}
	}

	origProc := origDayEditProc
	if hwnd == editNightTemp {
		origProc = origNightEditProc
	}
	ret, _, _ := procCallWindowProcW.Call(origProc, hwnd, msg, wParam, lParam)
	return ret
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

func centerWindow(hwnd uintptr) {
	var rc sliderRect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc))) //nolint:errcheck
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top
	sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	x := (int32(sw) - w) / 2
	y := (int32(sh) - h) / 2
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOZORDER) //nolint:errcheck
}

