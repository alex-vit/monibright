// Hotkey provides a thin wrapper around the Windows RegisterHotKey API.
// Inspired by golang.design/x/hotkey (https://github.com/nicot/hotkey).
// Simplified for our use case: one message loop, no keyup tracking, no polling.

//go:build windows

package main

import (
	"fmt"
	"log"
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

const (
	modWin = 0x8

	wmHotkey = 0x0312
)

type wmMsg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      [2]int32
}

// registerHotkeys registers each (modifier|vk) combo as a global hotkey and
// runs fn(id) on the calling goroutine's OS thread whenever a hotkey fires.
// The id passed to fn is the index into the hotkeys slice.
//
// This function blocks forever (it runs a Windows message loop). Call it from
// a dedicated goroutine. The goroutine's OS thread is locked automatically.
//
// hotkeys is a slice of [2]int{modifiers, vk}. Returns an error if any
// registration fails (but still registers the rest).
func registerHotkeys(hotkeys [][2]int, fn func(id int)) error {
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

	log.Printf("registered %d hotkeys (firstErr=%v)", len(hotkeys), firstErr)

	var m wmMsg
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
