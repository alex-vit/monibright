// Package hotkey provides a thin wrapper around the Windows RegisterHotKey API.
// It replaces golang.design/x/hotkey for our use case: register global hotkeys
// and get notified on keydown via a callback. No keyup tracking, no polling,
// no per-hotkey threads — just one message loop for all hotkeys.

//go:build windows

package hotkey

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var user32 = syscall.NewLazyDLL("user32.dll")

var (
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessageW      = user32.NewProc("GetMessageW")
)

// Modifier flags for RegisterHotKey.
const (
	ModAlt   = 0x1
	ModCtrl  = 0x2
	ModShift = 0x4
	ModWin   = 0x8
)

const wmHotkey = 0x0312

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      [2]int32
}

// RegisterHotkeys registers each (modifier|vk) combo as a global hotkey and
// runs fn(id) on the calling goroutine's OS thread whenever a hotkey fires.
// The id passed to fn is the index into the hotkeys slice.
//
// This function blocks forever (it runs a Windows message loop). Call it from
// a dedicated goroutine. The goroutine's OS thread is locked automatically.
//
// hotkeys is a slice of [2]int{modifiers, vk}. Returns an error if any
// registration fails (but still registers the rest).
func RegisterHotkeys(hotkeys [][2]int, fn func(id int)) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var firstErr error
	for i, hk := range hotkeys {
		ret, _, err := procRegisterHotKey.Call(0, uintptr(i+1), uintptr(hk[0]), uintptr(hk[1]))
		if ret == 0 {
			if firstErr == nil {
				firstErr = fmt.Errorf("RegisterHotKey(mod=0x%x, vk=0x%x): %w", hk[0], hk[1], err)
			}
		}
	}

	var m msg
	for {
		// GetMessageW blocks until a message is available. Returns 0 on WM_QUIT, -1 on error.
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		if m.message == wmHotkey {
			id := int(m.wParam) - 1 // we registered with id = index+1
			if id >= 0 && id < len(hotkeys) {
				fn(id)
			}
		}
	}

	for i := range hotkeys {
		procUnregisterHotKey.Call(0, uintptr(i+1))
	}
	return firstErr
}
