package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/alex-vit/monibright/icon"
	"github.com/energye/systray"
	"github.com/niluan304/ddcci"
	"golang.org/x/sys/windows/registry"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procCreateMutexW = kernel32.NewProc("CreateMutexW")

var version = ""

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

type isoLogWriter struct{ w io.Writer }

func (lw isoLogWriter) Write(p []byte) (int, error) {
	return fmt.Fprintf(lw.w, "%s %s", time.Now().Format("2006-01-02 15:04:05"), p)
}

func displayVersion() string {
	if version != "" {
		return version
	}
	return "dev"
}

func main() {
	name, _ := syscall.UTF16PtrFromString("MoniBrightMutex")
	procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))

	log.SetFlags(0)
	dataDir := filepath.Join(os.Getenv("LocalAppData"), "MoniBright")
	os.MkdirAll(dataDir, 0o755)
	logPath = filepath.Join(dataDir, "log.txt")
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		log.SetOutput(isoLogWriter{f})
	}
	log.Printf("MoniBright %s starting", displayVersion())

	cleanOldBinary()

	systray.Run(onReady, nil)
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTooltip("MoniBright")

	title := "MoniBright " + displayVersion()
	mTitle := systray.AddMenuItem(title, "")
	mTitle.Disable()
	systray.AddMenuItem("Open log", "Open log file").Click(func() {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", logPath).Start()
	})
	systray.AddSeparator()

	go autoUpdate()

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
	go runSlider()
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
		item.Click(func() { setBrightness(level) })
	}

	// Left-click: floating slider popup. Right-click: preset menu.
	systray.SetOnClick(func(menu systray.IMenu) { showSlider() })
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
		}
	}()
}

func updateIcon(level int) {
	systray.SetIcon(icon.Generate(level))
	systray.SetTooltip(fmt.Sprintf("MoniBright — %d%%", level))
}

func refreshCheck() {
	_, current, _, err := allMonitors[0].GetBrightness()
	if err != nil {
		log.Printf("GetBrightness failed: %v", err)
		return
	}
	log.Printf("GetBrightness: current=%d", current)

	// DDC/CI handles go stale after monitor sleep/wake and return 0.
	// Re-enumerate monitors and retry once.
	if current == 0 {
		log.Printf("brightness=0 is suspicious, re-enumerating monitors")
		if refreshMonitors() {
			_, current, _, err = allMonitors[0].GetBrightness()
			if err != nil {
				log.Printf("GetBrightness retry failed: %v", err)
				return
			}
			log.Printf("GetBrightness retry: current=%d", current)
		}
	}

	checkItem(brightItems, current)
	updateIcon(current)
}

func refreshMonitors() bool {
	sysMonitors, err := ddcci.NewSystemMonitors()
	if err != nil || len(sysMonitors) == 0 {
		log.Printf("re-enumerate failed: %d monitors, err=%v", len(sysMonitors), err)
		return false
	}
	var monitors []*ddcci.PhysicalMonitor
	for i := range sysMonitors {
		m, err := ddcci.NewPhysicalMonitor(&sysMonitors[i])
		if err != nil {
			log.Printf("re-enumerate monitor %d: %v", i, err)
			continue
		}
		monitors = append(monitors, m)
	}
	if len(monitors) == 0 {
		log.Printf("re-enumerate: no usable monitors")
		return false
	}
	allMonitors = monitors
	log.Printf("re-enumerated %d physical monitors", len(allMonitors))
	return true
}

func setBrightness(level int) {
	log.Printf("setting brightness to %d%%", level)

	setAll := func() {
		for i, m := range allMonitors {
			if err := m.SetBrightness(level); err != nil {
				log.Printf("monitor %d: SetBrightness(%d) error: %v", i, level, err)
			} else {
				log.Printf("monitor %d: SetBrightness(%d) ok", i, level)
			}
		}
	}

	setAll()

	// Verify the write took effect. Stale DDC/CI handles after sleep/wake
	// silently fail: SetBrightness returns nil but the monitor doesn't change.
	// Re-enumerate for fresh handles and retry.
	_, cur, _, err := allMonitors[0].GetBrightness()
	log.Printf("post-set verify: current=%d expected=%d err=%v", cur, level, err)
	diff := cur - level
	if diff < 0 {
		diff = -diff
	}
	if err != nil || diff > 5 {
		log.Printf("stale handle detected, refreshing monitors and retrying")
		if refreshMonitors() {
			setAll()
		}
	}

	checkItem(brightItems, level)
	updateIcon(level)
	syncSlider(level)
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
	return k.SetStringValue(registryName, `"`+exePath+`"`)
}

func autostartDisable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, registryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(registryName)
}
