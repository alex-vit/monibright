package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alex-vit/monibright/internal/icon"
	"github.com/energye/systray"
	"github.com/niluan304/ddcci"
	"golang.design/x/hotkey"
	"golang.org/x/sys/windows/registry"
)

var version = "dev"

const (
	VKNumpad0    = 0x60
	registryKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	registryName = "MoniBright"
)

func main() {
	systray.Run(onReady, nil)
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTooltip("MoniBright")

	title := "MoniBright " + version
	mTitle := systray.AddMenuItem(title, "")
	mTitle.Disable()
	systray.AddSeparator()

	monitors, err := ddcci.NewSystemMonitors()
	if err != nil || len(monitors) == 0 {
		mErr := systray.AddMenuItem("No monitors found", "")
		mErr.Disable()
		systray.AddSeparator()
		addQuit()
		return
	}

	var allMonitors []*ddcci.PhysicalMonitor
	for i := range monitors {
		m, err := ddcci.NewPhysicalMonitor(&monitors[i])
		if err != nil {
			log.Printf("monitor %d: %v", i, err)
			continue
		}
		allMonitors = append(allMonitors, m)
	}
	if len(allMonitors) == 0 {
		mErr := systray.AddMenuItem("No usable monitors", "")
		mErr.Disable()
		systray.AddSeparator()
		addQuit()
		return
	}

	// Menu items: 100, 90, ..., 10 (descending)
	items := make(map[int]*systray.MenuItem)
	for i := 10; i >= 1; i-- {
		level := i * 10
		item := systray.AddMenuItem(fmt.Sprintf("%d%%", level), fmt.Sprintf("Set brightness to %d%%", level))
		items[level] = item
	}

	// Read brightness from first monitor for UI state
	refreshCheck := func() {
		_, current, _, err := allMonitors[0].GetBrightness()
		if err != nil {
			return
		}
		checkItem(items, current)
	}

	// Set initial checkmark
	refreshCheck()

	// Set brightness on all monitors
	setBrightness := func(level int) {
		for i, m := range allMonitors {
			if err := m.SetBrightness(level); err != nil {
				log.Printf("monitor %d: failed to set brightness: %v", i, err)
			}
		}
		checkItem(items, level)
	}

	// Wire up click handlers
	for level, item := range items {
		level := level
		item.Click(func() { setBrightness(level) })
	}

	// Re-read brightness before showing menu (both left and right click)
	showMenu := func(menu systray.IMenu) {
		refreshCheck()
		menu.ShowMenu()
	}
	systray.SetOnClick(func(menu systray.IMenu) { showMenu(menu) })
	systray.SetOnRClick(func(menu systray.IMenu) { showMenu(menu) })

	systray.AddSeparator()

	// Autostart toggle
	mAutostart := systray.AddMenuItem("Start with Windows", "Launch MoniBright at login")
	if autostartEnabled() {
		mAutostart.Check()
	}
	mAutostart.Click(func() {
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
	})

	systray.AddSeparator()
	addQuit()

	// Hotkeys: Win+Numpad1=10%, Win+Numpad2=20%, ..., Win+Numpad0=100%
	hotkeyErrs := make(chan error, 10)
	for i := 0; i <= 9; i++ {
		vk := VKNumpad0 + i
		level := i * 10
		if level == 0 {
			level = 100
		}
		go registerHotkey(vk, level, setBrightness, hotkeyErrs)
	}
	go func() {
		for err := range hotkeyErrs {
			log.Fatalf("hotkey conflict (is another instance running?): %v", err)
		}
	}()
}

func checkItem(items map[int]*systray.MenuItem, level int) {
	for l, item := range items {
		if l == level {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

func registerHotkey(vk int, level int, set func(int), errs chan<- error) {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModWin}, hotkey.Key(vk))
	if err := hk.Register(); err != nil {
		errs <- fmt.Errorf("Win+Numpad%d (%d%%): %w", vk-VKNumpad0, level, err)
		return
	}
	for range hk.Keydown() {
		set(level)
	}
}

func addQuit() {
	systray.AddMenuItem("Quit", "Quit MoniBright").Click(func() { systray.Quit() })
}

func autostartEnabled() bool {
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
