package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type appSettings struct {
	Theme            string `json:"theme"`
	Modifiers        uint32 `json:"modifiers"`
	Key              uint32 `json:"key"`
	StartWithWindows bool   `json:"start_with_windows"`
	StartupMode      string `json:"startup_mode,omitempty"`
	CloseToTray      bool   `json:"close_to_tray"`
}

var config appSettings
var showSettings bool
var recordingShortcut bool
var hoverSwitch, hoverSettings, hoverBack bool
var popX, popY int32
var notice string
var noticeBad bool

const modAlt = 0x1
const modControl = 0x2
const modShift = 0x4
const modWin = 0x8
const modNoRepeat = 0x4000

func settingsPath() string {
	p, e := os.UserConfigDir()
	if e != nil {
		p = os.TempDir()
	}
	return filepath.Join(p, "TouchSwitch", "settings.json")
}
func loadSettings() appSettings {
	c := appSettings{Theme: "system", Modifiers: modControl | modAlt, Key: 0x54, CloseToTray: true}
	b, e := os.ReadFile(settingsPath())
	if e == nil {
		_ = json.Unmarshal(b, &c)
	}
	if c.Theme != "system" && c.Theme != "light" && c.Theme != "dark" {
		c.Theme = "system"
	}
	if c.Key == 0 {
		c.Key = 0x54
		c.Modifiers = modControl | modAlt
	}
	return c
}
func saveSettings() {
	p := settingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0700)
	b, e := json.MarshalIndent(config, "", "  ")
	if e == nil {
		_ = os.WriteFile(p, b, 0600)
	}
}
func updateTheme() {
	useLight = config.Theme == "light" || (config.Theme == "system" && checkLightTheme())
	applyTheme()
}
func setTheme(t string) {
	config.Theme = t
	saveSettings()
	updateTheme()
	setDarkTitlebar()
	startHoverTimer() // Animate the selection indicator to its new slot.
}
func registerConfiguredHotkey() bool {
	if hwnd == 0 {
		return false
	}
	r, _, _ := registerHotkey.Call(hwnd, 1, uintptr(config.Modifiers|modNoRepeat), uintptr(config.Key))
	return r != 0
}
func validShortcutKey(v uint32) bool {
	return v >= 0x30 && v <= 0x39 || v >= 0x41 && v <= 0x5a || v >= 0x70 && v <= 0x7b || v >= 0x20 && v <= 0x28 || v == 0x2d || v == 0x2e
}
func keyName(v uint32) string {
	if v >= 0x41 && v <= 0x5a || v >= 0x30 && v <= 0x39 {
		return string(rune(v))
	}
	if v >= 0x70 && v <= 0x7b {
		return fmt.Sprintf("F%d", v-0x6f)
	}
	switch v {
	case 0x20:
		return "Space"
	case 0x25:
		return "←"
	case 0x26:
		return "↑"
	case 0x27:
		return "→"
	case 0x28:
		return "↓"
	case 0x2d:
		return "Insert"
	case 0x2e:
		return "Delete"
	}
	return "Unknown"
}
func shortcutText() string {
	var parts []string
	if config.Modifiers&modControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if config.Modifiers&modAlt != 0 {
		parts = append(parts, "Alt")
	}
	if config.Modifiers&modShift != 0 {
		parts = append(parts, "Shift")
	}
	if config.Modifiers&modWin != 0 {
		parts = append(parts, "Win")
	}
	parts = append(parts, keyName(config.Key))
	return strings.Join(parts, " + ")
}
func keyDown(v uintptr) bool { return int16(uint16(v)) < 0 }
func captureShortcut(v uint32) {
	if v == 0x1b {
		recordingShortcut = false
		registerConfiguredHotkey()
		redraw()
		return
	}
	if !validShortcutKey(v) {
		return
	}
	var mods uint32
	if r, _, _ := getKeyState.Call(0x11); keyDown(r) {
		mods |= modControl
	}
	if r, _, _ := getKeyState.Call(0x12); keyDown(r) {
		mods |= modAlt
	}
	if r, _, _ := getKeyState.Call(0x10); keyDown(r) {
		mods |= modShift
	}
	if r, _, _ := getKeyState.Call(0x5b); keyDown(r) {
		mods |= modWin
	}
	if r, _, _ := getKeyState.Call(0x5c); keyDown(r) {
		mods |= modWin
	}
	if mods == 0 {
		showNotice("Use Ctrl, Alt, Shift or Win with a key", true)
		return
	}
	oldMods, oldKey := config.Modifiers, config.Key
	config.Modifiers, config.Key = mods, v
	if !registerConfiguredHotkey() {
		config.Modifiers, config.Key = oldMods, oldKey
		registerConfiguredHotkey()
		showNotice("That shortcut is already in use", true)
	} else {
		saveSettings()
		showNotice("Shortcut saved", false)
	}
	recordingShortcut = false
	redraw()
}
func beginRecording() {
	unregisterHotkey.Call(hwnd, 1)
	recordingShortcut = true
	notice = ""
	redraw()
}
func showNotice(s string, bad bool) { notice = s; noticeBad = bad; redraw() }
func cancelRecording() {
	if recordingShortcut {
		recordingShortcut = false
		registerConfiguredHotkey()
	}
}
func closeAppWindow() {
	cancelRecording()
	if config.CloseToTray {
		hidePopup()
	} else {
		destroyWindow.Call(hwnd)
	}
}

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	regCreateKeyEx = advapi32.NewProc("RegCreateKeyExW")
	regSetValueEx  = advapi32.NewProc("RegSetValueExW")
	regDeleteValue = advapi32.NewProc("RegDeleteValueW")
	regCloseKey    = advapi32.NewProc("RegCloseKey")
)

// Removes the older HKCU Run entry so the task never causes duplicate launches.
func removeLegacyStartup() error {
	const hkcu = uintptr(0x80000001)
	var key uintptr
	r, _, _ := regCreateKeyEx.Call(hkcu, ptr(utf(`Software\Microsoft\Windows\CurrentVersion\Run`)), 0, 0, 0, 0x2, 0, ptr(&key), 0)
	if r != 0 {
		return fmt.Errorf("couldn't open Windows startup settings (%d)", r)
	}
	defer regCloseKey.Call(key)
	r, _, _ = regDeleteValue.Call(key, ptr(utf("TouchSwitch")))
	if r != 0 && r != 2 {
		return fmt.Errorf("couldn't remove older startup entry (%d)", r)
	}
	return nil
}
