package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/alex-vit/monibright/icon"
	"github.com/energye/systray"
	"github.com/niluan304/ddcci"
	"golang.org/x/sys/windows/registry"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procCreateMutexW = kernel32.NewProc("CreateMutexW")

var version = "dev"

var logPath string

const (
	VKNumpad0    = 0x60
	registryKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	registryName = "MoniBright"
)

var (
	allMonitors []*ddcci.PhysicalMonitor
	brightItems map[int]*systray.MenuItem
	mAutostart  *systray.MenuItem
)

func isDebugBuild() bool { return version == "dev" }

func main() {
	name, _ := syscall.UTF16PtrFromString("MoniBrightMutex")
	procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))

	if isDebugBuild() {
		appData := os.Getenv("APPDATA")
		dir := filepath.Join(appData, "monibright")
		os.MkdirAll(dir, 0o755)
		logPath = filepath.Join(dir, "debug.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			log.SetOutput(f)
		}
	} else {
		log.SetOutput(io.Discard)
	}
	log.Printf("MoniBright %s starting (debug=%v)", version, isDebugBuild())

	systray.Run(onReady, nil)
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTooltip("MoniBright")

	title := "MoniBright " + version
	mTitle := systray.AddMenuItem(title, "")
	mTitle.Disable()
	if isDebugBuild() {
		mLog := systray.AddMenuItem("Open log", "Open debug log file")
		mLog.Click(func() {
			exec.Command("cmd", "/c", "start", "", logPath).Start()
		})
	}
	systray.AddSeparator()

	sysMonitors, err := ddcci.NewSystemMonitors()
	log.Printf("enumerated %d system monitors (err=%v)", len(sysMonitors), err)
	if err != nil || len(sysMonitors) == 0 {
		mErr := systray.AddMenuItem("No monitors found", "")
		mErr.Disable()
		systray.AddSeparator()
		addQuit()
		return
	}

	for i := range sysMonitors {
		m, err := ddcci.NewPhysicalMonitor(&sysMonitors[i])
		if err != nil {
			log.Printf("monitor %d: %v", i, err)
			continue
		}
		allMonitors = append(allMonitors, m)
	}
	log.Printf("initialized %d physical monitors", len(allMonitors))
	if len(allMonitors) == 0 {
		mErr := systray.AddMenuItem("No usable monitors", "")
		mErr.Disable()
		systray.AddSeparator()
		addQuit()
		return
	}

	// Menu items: 100, 90, ..., 10 (descending)
	brightItems = make(map[int]*systray.MenuItem)
	for i := 10; i >= 1; i-- {
		level := i * 10
		item := systray.AddMenuItem(fmt.Sprintf("%d%%", level), fmt.Sprintf("Set brightness to %d%%", level))
		brightItems[level] = item
	}

	// Set initial checkmark
	refreshCheck()

	// Wire up click handlers
	for level, item := range brightItems {
		level := level
		item.Click(func() { setBrightness(level) })
	}

	// Re-read brightness before showing menu (both left and right click)
	systray.SetOnClick(showMenu)
	systray.SetOnRClick(showMenu)

	systray.AddSeparator()

	// Autostart toggle
	mAutostart = systray.AddMenuItem("Start with Windows", "Launch MoniBright at login")
	if isAutostartEnabled() {
		mAutostart.Check()
	}
	mAutostart.Click(toggleAutostart)

	systray.AddSeparator()
	addQuit()

	// Hotkeys: Win+Numpad1=10%, Win+Numpad2=20%, ..., Win+Numpad0=100%
	var hkeys [][2]int
	var levels []int
	for i := 0; i <= 9; i++ {
		hkeys = append(hkeys, [2]int{modWin, VKNumpad0 + i})
		level := i * 10
		if level == 0 {
			level = 100
		}
		levels = append(levels, level)
	}
	go func() {
		if err := registerHotkeys(hkeys, func(id int) {
			setBrightness(levels[id])
		}); err != nil {
			log.Printf("hotkey registration error: %v", err)
		} else {
			log.Printf("registered %d hotkeys", len(hkeys))
		}
	}()
}

func refreshCheck() {
	_, current, _, err := allMonitors[0].GetBrightness()
	if err != nil {
		log.Printf("GetBrightness failed: %v", err)
		return
	}
	log.Printf("GetBrightness: current=%d", current)
	checkItem(brightItems, current)
}

func setBrightness(level int) {
	log.Printf("setting brightness to %d%%", level)
	for i, m := range allMonitors {
		if err := m.SetBrightness(level); err != nil {
			log.Printf("monitor %d: SetBrightness(%d) error: %v", i, level, err)
		} else {
			log.Printf("monitor %d: SetBrightness(%d) ok", i, level)
		}
	}
	checkItem(brightItems, level)
}

func showMenu(menu systray.IMenu) {
	refreshCheck()
	menu.ShowMenu()
}

func toggleAutostart() {
	if mAutostart.Checked() {
		if err := autostartDisable(); err != nil {
			log.Printf("failed to disable autostart: %v", err)
			return
		}
		mAutostart.Uncheck()
	} else {
		if err := autostartEnable(); err != nil {
			log.Printf("failed to enable autostart: %v", err)
			return
		}
		mAutostart.Check()
	}
}

func checkItem(items map[int]*systray.MenuItem, level int) {
	// Round to nearest preset (10, 20, ..., 100) so non-preset values
	// (e.g. 75% set via monitor buttons) still show a checkmark.
	nearest := max(((level+5)/10)*10, 10)
	for l, item := range items {
		if l == nearest {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func addQuit() {
	systray.AddMenuItem("Quit", "Quit MoniBright").Click(func() { systray.Quit() })
}

func isAutostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(registryName)
	return err == nil
}

func autostartEnable() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(registryName, exePath)
}

func autostartDisable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(registryName)
}
