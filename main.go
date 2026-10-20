// TouchSwitch - an intentionally small, dependency-free native Windows tray app.
// Targets HID touchscreen devices only, never HID touchpads.
// Copyright (c) 2026. MIT licence; see LICENSE.
package main

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

//go:embed touch-on.ico
var iconOnBytes []byte

//go:embed touch-off.ico
var iconOffBytes []byte

//go:embed touch-mark.ico
var iconMarkBytes []byte

//go:embed touch-mark-dark.ico
var iconMarkDarkBytes []byte

const (
	wmDestroy          = 0x0002
	wmPaint            = 0x000F
	wmClose            = 0x0010
	wmActivate         = 0x0006
	wmLButtonUp        = 0x0202
	wmMouseMove        = 0x0200
	wmMouseLeave       = 0x02A3
	wmLButtonDown      = 0x0201
	wmSettingChange    = 0x001A
	wmThemeChanged     = 0x031A
	wmDwmColorChanged  = 0x0320
	wmRButtonUp        = 0x0205
	wmTimer            = 0x0113
	wmApp              = 0x8000
	wmTray             = wmApp + 1
	wmDone             = wmApp + 2
	wmShow             = wmApp + 3
	wmTheme            = wmApp + 4
	wmHotkey           = 0x0312
	wmKeyDown          = 0x0100
	wmSysKeyDown       = 0x0104
	wmDeviceChange     = 0x0219
	wsPopup            = 0x80000000
	wsExToolWindow     = 0x00000080
	wsExTopmost        = 0x00000008
	swHide             = 0
	swShow             = 5
	csVredraw          = 0x0001
	csHredraw          = 0x0002
	nidMessage         = 0x1
	nidIcon            = 0x2
	nidTip             = 0x4
	nidInfo            = 0x10
	nimAdd             = 0x0
	nimModify          = 0x1
	nimDelete          = 0x2
	niifInfo           = 0x1
	mfString           = 0
	mfSeparator        = 0x800
	tpmReturnCmd       = 0x100
	tpmRightButton     = 0x2
	transparent        = 1
	dtLeft             = 0x0
	dtVCenter          = 0x4
	dtSingleLine       = 0x20
	dtNoPrefix         = 0x800
	fwNormal           = 400
	fwSemibold         = 600
	spdrpDevDesc       = 0x0
	spdrpHardwareID    = 0x1
	spdrpCompatibleIDs = 0x2
	spdrpConfigFlags   = 0xA
	spdrpFriendlyName  = 0xC
	cmProbDisabled     = 22
	configFlagDisabled = 0x1
	idMenuShow         = 101
	idMenuOn           = 102
	idMenuOff          = 103
	idMenuExit         = 104
	idMenuSettings     = 105
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}
type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type paintstruct struct {
	HDC                uintptr
	Erase              int32
	RcPaint            rect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type wndclassex struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}
type msg struct {
	HWnd    uintptr
	Message uint32
	Pad     uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}
type deviceInfo struct {
	Size      uint32
	ClassGuid guid
	DevInst   uint32
	Reserved  uintptr
}
type notifyIconData struct {
	Size            uint32
	HWnd            uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Timeout         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	Guid            guid
	BalloonIcon     uintptr
}
type device struct {
	id      string
	enabled bool
}
type theme struct {
	background, panel, foreground, muted, stroke, accent, inactive, white uint32
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	setupapi = syscall.NewLazyDLL("setupapi.dll")
	cfgmgr32 = syscall.NewLazyDLL("cfgmgr32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	registerClass          = user32.NewProc("RegisterClassExW")
	createWindow           = user32.NewProc("CreateWindowExW")
	defWindowProc          = user32.NewProc("DefWindowProcW")
	showWindow             = user32.NewProc("ShowWindow")
	setForegroundWindow    = user32.NewProc("SetForegroundWindow")
	destroyWindow          = user32.NewProc("DestroyWindow")
	getMessage             = user32.NewProc("GetMessageW")
	translateMessage       = user32.NewProc("TranslateMessage")
	dispatchMessage        = user32.NewProc("DispatchMessageW")
	postQuitMessage        = user32.NewProc("PostQuitMessage")
	postMessage            = user32.NewProc("PostMessageW")
	invalidateRect         = user32.NewProc("InvalidateRect")
	beginPaint             = user32.NewProc("BeginPaint")
	endPaint               = user32.NewProc("EndPaint")
	getCursorPos           = user32.NewProc("GetCursorPos")
	moveWindow             = user32.NewProc("MoveWindow")
	getSystemMetrics       = user32.NewProc("GetSystemMetrics")
	setTimer               = user32.NewProc("SetTimer")
	getDpiForSystem        = user32.NewProc("GetDpiForSystem")
	systemParametersInfo   = user32.NewProc("SystemParametersInfoW")
	setDpiAware            = user32.NewProc("SetProcessDPIAware")
	createPopupMenu        = user32.NewProc("CreatePopupMenu")
	appendMenu             = user32.NewProc("AppendMenuW")
	trackPopupMenu         = user32.NewProc("TrackPopupMenu")
	destroyMenu            = user32.NewProc("DestroyMenu")
	messageBox             = user32.NewProc("MessageBoxW")
	loadCursor             = user32.NewProc("LoadCursorW")
	loadIcon               = user32.NewProc("LoadIconW")
	createRoundRectRgn     = gdi32.NewProc("CreateRoundRectRgn")
	setWindowRgn           = user32.NewProc("SetWindowRgn")
	createCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	bitBlt                 = gdi32.NewProc("BitBlt")
	deleteDC               = gdi32.NewProc("DeleteDC")
	createIconFromResource = user32.NewProc("CreateIconFromResourceEx")
	destroyIcon            = user32.NewProc("DestroyIcon")
	shellNotifyIcon        = shell32.NewProc("Shell_NotifyIconW")
	isUserAnAdmin          = shell32.NewProc("IsUserAnAdmin")
	shellExecute           = shell32.NewProc("ShellExecuteW")
	getClassDevs           = setupapi.NewProc("SetupDiGetClassDevsW")
	enumDeviceInfo         = setupapi.NewProc("SetupDiEnumDeviceInfo")
	getDeviceProperty      = setupapi.NewProc("SetupDiGetDeviceRegistryPropertyW")
	getDeviceID            = setupapi.NewProc("SetupDiGetDeviceInstanceIdW")
	destroyDeviceInfoList  = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
	cmGetDevNodeStatus     = cfgmgr32.NewProc("CM_Get_DevNode_Status")
	createSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	createPen              = gdi32.NewProc("CreatePen")
	selectObject           = gdi32.NewProc("SelectObject")
	deleteObject           = gdi32.NewProc("DeleteObject")
	roundRect              = gdi32.NewProc("RoundRect")
	ellipse                = gdi32.NewProc("Ellipse")
	setBkMode              = gdi32.NewProc("SetBkMode")
	setTextColor           = gdi32.NewProc("SetTextColor")
	drawText               = user32.NewProc("DrawTextW")
	createFont             = gdi32.NewProc("CreateFontW")
	getModuleHandle        = kernel32.NewProc("GetModuleHandleW")
	regGetValue            = syscall.NewLazyDLL("advapi32.dll").NewProc("RegGetValueW")
	killTimer              = user32.NewProc("KillTimer")
	moveToEx               = gdi32.NewProc("MoveToEx")
	lineTo                 = gdi32.NewProc("LineTo")
	setCursor              = user32.NewProc("SetCursor")
	setDwmAttribute        = dwmapi.NewProc("DwmSetWindowAttribute")
	createMutex            = kernel32.NewProc("CreateMutexW")
	closeHandle            = kernel32.NewProc("CloseHandle")
	findWindow             = user32.NewProc("FindWindowW")
	registerHotkey         = user32.NewProc("RegisterHotKey")
	unregisterHotkey       = user32.NewProc("UnregisterHotKey")
	getKeyState            = user32.NewProc("GetKeyState")
	trackMouseEvent        = user32.NewProc("TrackMouseEvent")
	dwmGetColorization     = dwmapi.NewProc("DwmGetColorizationColor")
	drawIconEx             = user32.NewProc("DrawIconEx")
)

var (
	hwnd                                              uintptr
	iconOn                                            uintptr
	iconOff                                           uintptr
	trayData                                          notifyIconData
	current                                           device
	found, busy, hovered, popupVisible                bool
	lastError                                         string
	errorLock                                         sync.Mutex
	scale                                             float64 = 1
	w, h                                              int32
	fontTitle, fontMain, fontSub, fontTiny, fontBadge uintptr
	fontIcon, fontIconSmall, fontLogo, fontTouch      uintptr
	iconMark, iconMarkDark, iconBadge, iconBadgeOff   uintptr
	colors                                            = theme{
		background: rgb(22, 25, 33), panel: rgb(34, 39, 50),
		foreground: rgb(245, 247, 251), muted: rgb(150, 160, 178),
		stroke: rgb(51, 59, 73), accent: rgb(82, 151, 242),
		inactive: rgb(82, 88, 104), white: rgb(255, 255, 255),
	}
	// The Windows HID setup class GUID, NOT a pointer/touchpad class.
	hidClass = guid{0x745A17A0, 0x74D3, 0x11D0, [8]byte{0xB6, 0xFE, 0x00, 0xA0, 0xC9, 0x0F, 0x57, 0xDA}}
)

func utf(s string) *uint16      { return syscall.StringToUTF16Ptr(s) }
func rgb(r, g, b uint32) uint32 { return r | (g << 8) | (b << 16) }
func d(n int32) int32           { return int32(float64(n)*scale + 0.5) }
func ptr[T any](x *T) uintptr   { return uintptr(unsafe.Pointer(x)) }

func main() {
	runtime.LockOSThread()
	config = loadSettings()
	// The existing working PnP implementation runs elevated once per launch.
	if !previewMode() {
		if r, _, _ := isUserAnAdmin.Call(); r == 0 {
			exe, e := os.Executable()
			if e != nil {
				return
			}
			args := strings.Join(os.Args[1:], " ")
			r, _, _ = shellExecute.Call(0, ptr(utf("runas")), ptr(utf(exe)), ptr(utf(args)), 0, 1)
			if r <= 32 {
				messageBox.Call(0, ptr(utf("TouchSwitch needs administrator permission to control your touchscreen.")), ptr(utf("TouchSwitch")), 0x30)
			}
			return
		}
	}
	mutex, _, mutexError := createMutex.Call(0, 0, ptr(utf(`Local\TouchSwitch.SingleInstance.2026`)))
	if mutex != 0 {
		defer closeHandle.Call(mutex)
		if mutexError == syscall.Errno(183) {
			existing, _, _ := findWindow.Call(ptr(utf("TouchSwitch.Tray.Popup.V3")), 0)
			if existing != 0 {
				postMessage.Call(existing, wmShow, 0, 0)
			}
			return
		}
	}
	setDpiAware.Call()
	if dpi, _, _ := getDpiForSystem.Call(); dpi >= 96 && dpi <= 384 {
		scale = float64(dpi) / 96
	}
	updateTheme()
	w, h = d(352), d(416) // fixed size on both pages
	createFonts()
	initGDIPlus()
	defer shutdownGDIPlus()
	iconOn = createIcon(iconOnBytes)
	iconOff = createIcon(iconOffBytes)
	iconMark = createIconSized(iconMarkBytes, 104)
	iconMarkDark = createIconSized(iconMarkDarkBytes, 104)
	iconBadge = createIconSized(iconOnBytes, 48)
	iconBadgeOff = createIconSized(iconOffBytes, 48)
	if iconOn == 0 {
		iconOn, _, _ = loadIcon.Call(0, 32512)
	}
	if iconOff == 0 {
		iconOff = iconOn
	}
	defer func() {
		if iconOn != 0 && iconOn != iconOff {
			destroyIcon.Call(iconOn)
		}
		if iconOff != 0 && iconOff != iconOn {
			destroyIcon.Call(iconOff)
		}
		for _, ico := range []uintptr{iconMark, iconMarkDark, iconBadge, iconBadgeOff} {
			if ico != 0 {
				destroyIcon.Call(ico)
			}
		}
		for _, f := range []uintptr{fontTitle, fontMain, fontSub, fontTiny, fontBadge, fontIcon, fontIconSmall, fontLogo, fontTouch} {
			if f != 0 {
				deleteObject.Call(f)
			}
		}
	}()
	instance, _, _ := getModuleHandle.Call(0)
	cls := utf("TouchSwitch.Tray.Popup.V3")
	wc := wndclassex{Size: uint32(unsafe.Sizeof(wndclassex{})), Style: csHredraw | csVredraw, WndProc: syscall.NewCallback(windowProcedure), Instance: instance, Icon: iconOn, IconSm: iconOn, ClassName: cls}
	cursorArrow, _, _ = loadCursor.Call(0, 32512)
	cursorHand, _, _ = loadCursor.Call(0, 32649)
	wc.Cursor = cursorArrow
	if r, _, _ := registerClass.Call(ptr(&wc)); r == 0 {
		return
	}
	hwnd, _, _ = createWindow.Call(wsExToolWindow|wsExTopmost, ptr(cls), ptr(utf("TouchSwitch")), wsPopup, 0, 0, uintptr(w), uintptr(h), 0, 0, instance, 0)
	if hwnd == 0 {
		return
	}
	setWindowCorners()
	createToastWindow(instance)
	initHoverAnimation()
	refresh()
	position = 0
	if current.enabled {
		position = 1
	}
	trayData.Size = uint32(unsafe.Sizeof(trayData))
	trayData.HWnd = hwnd
	trayData.ID = 1
	trayData.Flags = nidMessage | nidIcon | nidTip
	trayData.CallbackMessage = wmTray
	updateTray(nimAdd)
	if !registerConfiguredHotkey() {
		showNotice("Shortcut is already being used", false)
	}
	setTimer.Call(hwnd, 1, 5000, 0)
	if !isStartupLaunch() {
		displayPopup()
	}
	var m msg
	for {
		ok, _, _ := getMessage.Call(ptr(&m), 0, 0, 0)
		if int32(ok) <= 0 {
			break
		}
		translateMessage.Call(ptr(&m))
		dispatchMessage.Call(ptr(&m))
	}
	unregisterHotkey.Call(hwnd, 1)
	updateTray(nimDelete)
}

func isStartupLaunch() bool {
	for _, s := range os.Args[1:] {
		if s == "-startup" {
			return true
		}
	}
	return false
}

func setWindowCorners() {
	// Native Windows 11 DWM anti-aliased corners. Avoid a hard-edged GDI
	// window region unless the OS doesn't support rounded corners.
	var rounded int32 = 2
	hr, _, _ := setDwmAttribute.Call(hwnd, 33, ptr(&rounded), 4)
	if hr != 0 {
		rgn, _, _ := createRoundRectRgn.Call(0, 0, uintptr(w+1), uintptr(h+1), uintptr(d(20)), uintptr(d(20)))
		if rgn != 0 {
			setWindowRgn.Call(hwnd, rgn, 1)
		}
	}
	setDarkTitlebar()
}

func setDarkTitlebar() {
	var dark int32 = 1
	if useLight {
		dark = 0
	}
	setDwmAttribute.Call(hwnd, 20, ptr(&dark), 4)
}

func createFonts() {
	font := utf("Segoe UI")
	iconFont := utf("Segoe Fluent Icons")
	mk := func(name *uint16, size int32, weight uintptr) uintptr {
		r, _, _ := createFont.Call(uintptr(-d(size)), 0, 0, 0, weight, 0, 0, 0, 1, 0, 0, 6, 0, ptr(name))
		return r
	}
	fontTitle = mk(font, 18, fwSemibold)
	fontMain = mk(font, 15, fwSemibold)
	fontSub = mk(font, 12, fwNormal)
	fontTiny = mk(font, 11, fwNormal)
	fontBadge = mk(font, 10, fwSemibold)
	fontIcon = mk(iconFont, 15, fwNormal)
	fontIconSmall = mk(iconFont, 15, fwNormal)
	fontLogo = mk(iconFont, 26, fwNormal)
	fontTouch = mk(iconFont, 77, fwNormal)
}

// Parse an .ico image directory and pass the PNG/DIB entry to the native API.
func createIcon(data []byte) uintptr {
	if len(data) < 22 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return 0
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	best := -1
	bestDelta := 9999
	for i := 0; i < count; i++ {
		o := 6 + i*16
		if o+16 > len(data) {
			break
		}
		sz := int(data[o])
		if sz == 0 {
			sz = 256
		}
		delta := sz - 32
		if delta < 0 {
			delta = -delta
		}
		if delta < bestDelta {
			best = i
			bestDelta = delta
		}
	}
	if best < 0 {
		return 0
	}
	off := 6 + best*16
	size := int(binary.LittleEndian.Uint32(data[off+8 : off+12]))
	pos := int(binary.LittleEndian.Uint32(data[off+12 : off+16]))
	if pos < 0 || size <= 0 || pos+size > len(data) {
		return 0
	}
	r, _, _ := createIconFromResource.Call(uintptr(unsafe.Pointer(&data[pos])), uintptr(size), 1, 0x30000, uintptr(d(16)), uintptr(d(16)), 0)
	runtime.KeepAlive(data)
	return r
}

func createIconSized(data []byte, size int32) uintptr {
	if len(data) < 22 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return 0
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	best, bestDiff := -1, 1<<30
	for i := 0; i < count; i++ {
		o := 6 + i*16
		if o+16 > len(data) {
			break
		}
		pixels := int(data[o])
		if pixels == 0 {
			pixels = 256
		}
		diff := pixels - int(d(size))
		if diff < 0 {
			diff = -diff + 1000 // prefer downsampling over magnification
		}
		if diff < bestDiff {
			best = i
			bestDiff = diff
		}
	}
	if best < 0 {
		return 0
	}
	o := 6 + best*16
	length := int(binary.LittleEndian.Uint32(data[o+8 : o+12]))
	start := int(binary.LittleEndian.Uint32(data[o+12 : o+16]))
	if length <= 0 || start+length > len(data) {
		return 0
	}
	px := d(size)
	r, _, _ := createIconFromResource.Call(uintptr(unsafe.Pointer(&data[start])), uintptr(length), 1, 0x30000, uintptr(px), uintptr(px), 0)
	runtime.KeepAlive(data)
	return r
}

func windowProcedure(window uintptr, event uint32, wp, lp uintptr) uintptr {
	switch event {
	case wmPaint:
		drawPopup()
		return 0
	case wmTray:
		if uint32(lp) == wmLButtonUp {
			refresh()
			requestChangeWithFeedback(!current.enabled, true)
		}
		if uint32(lp) == wmRButtonUp || uint32(lp) == 0x7B {
			menuPopup()
		}
		return 0
	case wmHotkey:
		if wp == 1 {
			refresh()
			requestChangeWithFeedback(!current.enabled, true)
		}
		return 0
	case wmKeyDown, wmSysKeyDown:
		if recordingShortcut {
			captureShortcut(uint32(wp))
		}
		return 0
	case wmDeviceChange:
		if !busy {
			refresh()
			redraw()
		}
		return 0
	case wmActivate:
		if (wp & 0xffff) == 0 {
			cancelRecording()
			hidePopup()
		}
		return 0
	case wmMouseMove:
		x, y := int32(int16(uint16(lp))), int32(int16(uint16(lp>>16)))
		updatePointer(x, y)
		return 0
	case wmMouseLeave:
		mouseTracking = false
		pressedControl = controlNone
		setHoverControl(controlNone)
		return 0
	case wmLButtonDown:
		x, y := int32(int16(uint16(lp))), int32(int16(uint16(lp>>16)))
		pressedControl = hitControl(x, y)
		startHoverTimer()
		redraw()
		return 0
	case wmLButtonUp:
		x, y := int32(int16(uint16(lp))), int32(int16(uint16(lp>>16)))
		pressedControl = controlNone
		switch {
		case inClose(x, y):
			closeAppWindow()
		case showSettings && inBack(x, y):
			switchPage(false)
		case showSettings:
			handleSettingsClick(x, y)
		case inGear(x, y):
			switchPage(true)
		case inMainCircle(x, y):
			requestChange(!current.enabled)
		}
		return 0
	case wmTimer:
		if wp == 2 {
			tickAnimation()
			return 0
		}
		if wp == 4 {
			tickHoverAnimation()
			return 0
		}
		if !busy {
			refresh()
			redraw()
		}
		previousLight, previousAccent := useLight, colors.accent
		updateTheme()
		if previousLight != useLight {
			setDarkTitlebar()
		}
		if previousLight != useLight || previousAccent != colors.accent {
			redraw()
			if toastAvailable {
				invalidateRect.Call(toastWnd, 0, 0)
			}
		}
		return 0
	case wmSettingChange, wmThemeChanged, wmDwmColorChanged:
		updateTheme()
		setDarkTitlebar()
		redraw()
		return 0
	case wmDone:
		busy = false
		refresh()
		redraw()
		errorLock.Lock()
		failure := lastError
		errorLock.Unlock()
		if failure != "" {
			startAnimation(current.enabled) // restore the button appearance after a failed change
			pendingStateNotification = false
			messageBox.Call(hwnd, ptr(utf(failure)), ptr(utf("TouchSwitch")), 0x30)
		} else if pendingStateNotification {
			if found && current.enabled == pendingDesiredState {
				showStateNotification(current.enabled)
			}
			pendingStateNotification = false
		}
		return 0
	case wmClose:
		closeAppWindow()
		return 0
	case wmDestroy:
		postQuitMessage.Call(0)
		return 0
	case wmShow:
		displayPopup()
		return 0
	}
	r, _, _ := defWindowProc.Call(window, uintptr(event), wp, lp)
	return r
}

// Interaction hit-testing is implemented in gui.go.

func redraw() {
	if hwnd != 0 {
		invalidateRect.Call(hwnd, 0, 0)
	}
}
func hidePopup() {
	if popupVisible {
		showWindow.Call(hwnd, swHide)
		popupVisible = false
	}
}
func displayPopup() {
	refresh()
	var pt point
	getCursorPos.Call(ptr(&pt))
	var area rect
	if r, _, _ := systemParametersInfo.Call(48, 0, ptr(&area), 0); r == 0 {
		area.Right = int32(systemMetric(0))
		area.Bottom = int32(systemMetric(1))
	}
	x := pt.X - w + 20
	y := pt.Y - h - 16
	if pt.Y < area.Top+(area.Bottom-area.Top)/2 {
		y = pt.Y + 18
	}
	if x < area.Left+8 {
		x = area.Left + 8
	}
	if x+w > area.Right-8 {
		x = area.Right - w - 8
	}
	if y < area.Top+8 {
		y = area.Top + 8
	}
	if y+h > area.Bottom-8 {
		y = area.Bottom - h - 8
	}
	popX, popY = x, y
	moveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
	popupVisible = true
	showWindow.Call(hwnd, swShow)
	setForegroundWindow.Call(hwnd)
	redraw()
}
func switchPage(settings bool) {
	cancelRecording()
	showSettings = settings
	pressedControl = controlNone
	setHoverControl(controlNone)
	// Both pages share one 352x416 DIP window; never change the height.
	redraw()
}

func systemMetric(n uintptr) uintptr { v, _, _ := getSystemMetrics.Call(n); return v }

func menuPopup() {
	menu, _, _ := createPopupMenu.Call()
	if menu == 0 {
		return
	}
	appendMenu.Call(menu, mfString, idMenuShow, ptr(utf("Open TouchSwitch")))
	appendMenu.Call(menu, mfSeparator, 0, 0)
	if current.enabled {
		appendMenu.Call(menu, mfString, idMenuOff, ptr(utf("Turn touchscreen off")))
	} else {
		appendMenu.Call(menu, mfString, idMenuOn, ptr(utf("Turn touchscreen on")))
	}
	appendMenu.Call(menu, mfString, idMenuSettings, ptr(utf("Settings")))
	appendMenu.Call(menu, mfSeparator, 0, 0)
	appendMenu.Call(menu, mfString, idMenuExit, ptr(utf("Exit TouchSwitch")))
	var pt point
	getCursorPos.Call(ptr(&pt))
	setForegroundWindow.Call(hwnd)
	id, _, _ := trackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	destroyMenu.Call(menu)
	switch id {
	case idMenuShow:
		displayPopup()
	case idMenuOn:
		requestChange(true)
	case idMenuOff:
		requestChange(false)
	case idMenuSettings:
		switchPage(true)
		displayPopup()
	case idMenuExit:
		destroyWindow.Call(hwnd)
	}
}

func updateTray(action uintptr) {
	if found && current.enabled {
		trayData.Icon = iconOn
	} else {
		trayData.Icon = iconOff
	}
	tooltip := "TouchSwitch • Touchscreen not detected"
	if previewMode() {
		tooltip = "TouchSwitch • UI preview (no hardware changes)"
	}
	if found && !previewMode() {
		if current.enabled {
			tooltip = "TouchSwitch • Touchscreen ON"
		} else {
			tooltip = "TouchSwitch • Touchscreen OFF"
		}
	}
	copy(trayData.Tip[:], syscall.StringToUTF16(tooltip))
	shellNotifyIcon.Call(action, ptr(&trayData))
}

func refresh() {
	if previewMode() {
		current = device{id: "ui-preview", enabled: previewEnabled}
		found = true
	} else {
		d, ok := findTouchscreen()
		current = d
		found = ok
	}
	if !busy && !animating {
		if current.enabled {
			position = 1
		} else {
			position = 0
		}
	}
	if hwnd != 0 && trayData.HWnd != 0 {
		updateTray(nimModify)
	}
}

func requestChangeWithFeedback(enable bool, notify bool) {
	if busy || !found || current.enabled == enable {
		return
	}
	if notify {
		pendingStateNotification = true
		pendingDesiredState = enable
	}
	requestChange(enable)
	if busy == false && previewMode() && notify {
		showStateNotification(current.enabled)
		pendingStateNotification = false
	}
}

func showStateNotification(enabled bool) {
	showTouchToast(enabled) // app-owned 1-second popup, unaffected by Windows DND
}

func requestChange(enable bool) {
	if busy || !found || current.enabled == enable {
		return
	}
	if previewMode() {
		startAnimation(enable)
		previewEnabled = enable
		current.enabled = enable
		updateTray(nimModify)
		redraw()
		return
	}
	startAnimation(enable)
	// Target a single verified touchscreen device instance ID, NOT all HID devices.
	id := current.id
	busy = true
	errorLock.Lock()
	lastError = ""
	errorLock.Unlock()
	redraw()
	go func() {
		cmdName := "/disable-device"
		if enable {
			cmdName = "/enable-device"
		}
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot == "" {
			systemRoot = `C:\Windows`
		}
		cmd := exec.Command(filepath.Join(systemRoot, "System32", "pnputil.exe"), cmdName, id)
		// No console window flashes on the desktop.
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		output, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(output))
			if len(detail) > 350 {
				detail = detail[:350]
			}
			if detail == "" {
				detail = err.Error()
			}
			errorLock.Lock()
			lastError = fmt.Sprintf("Windows couldn't change the touchscreen state.\n\n%s", detail)
			errorLock.Unlock()
		}
		postMessage.Call(hwnd, wmDone, 0, 0)
	}()
}

func findTouchscreen() (device, bool) {
	h, _, _ := getClassDevs.Call(ptr(&hidClass), 0, 0, 0)
	if h == 0 || h == ^uintptr(0) {
		return device{}, false
	}
	defer destroyDeviceInfoList.Call(h)
	var chosen device
	bestScore := -1
	for i := uint32(0); i < 2048; i++ {
		info := deviceInfo{Size: uint32(unsafe.Sizeof(deviceInfo{}))}
		r, _, _ := enumDeviceInfo.Call(h, uintptr(i), ptr(&info))
		if r == 0 {
			break
		}
		name := strings.ToLower(getProp(h, &info, spdrpFriendlyName))
		if name == "" {
			name = strings.ToLower(getProp(h, &info, spdrpDevDesc))
		}
		ids := strings.ToUpper(getProp(h, &info, spdrpCompatibleIDs) + "\x00" + getProp(h, &info, spdrpHardwareID))
		// Generic touchscreen HID usage is Digitizers page 0x0D, usage 0x04.
		// Touchpad usage 0x05 must never be matched by name fuzziness.
		hardwareTouch := strings.Contains(ids, "HID_DEVICE_UP:000D_U:0004")
		namedTouch := strings.Contains(name, "touch screen") || strings.Contains(name, "touchscreen")
		if !hardwareTouch && !namedTouch {
			continue
		}
		if strings.Contains(name, "touchpad") || strings.Contains(name, "touch pad") {
			continue
		}
		id := getInstanceID(h, &info)
		if id == "" {
			continue
		}
		var devStatus, problem uint32
		cr, _, _ := cmGetDevNodeStatus.Call(ptr(&devStatus), ptr(&problem), uintptr(info.DevInst), 0)
		config := getConfigFlags(h, &info)
		disabled := problem == cmProbDisabled || config&configFlagDisabled != 0
		// Prefer a device with valid current PnP status; when tied, prefer
		// the touchscreen usage identifier over translated friendly names.
		score := 0
		if hardwareTouch {
			score += 5
		}
		if namedTouch {
			score += 2
		}
		if cr == 0 {
			score += 2
		}
		if score > bestScore {
			bestScore = score
			chosen = device{id: id, enabled: !disabled}
		}
	}
	return chosen, bestScore >= 0
}

func getProp(h uintptr, dev *deviceInfo, prop uintptr) string {
	var regtype, needed uint32
	buf := make([]uint16, 2048)
	r, _, _ := getDeviceProperty.Call(h, ptr(dev), prop, ptr(&regtype), ptr(&buf[0]), uintptr(len(buf)*2), ptr(&needed))
	if r == 0 {
		return ""
	}
	if needed > uint32(len(buf)*2) {
		return ""
	}
	return string(utf16.Decode(buf[:needed/2]))
}
func getConfigFlags(h uintptr, dev *deviceInfo) uint32 {
	var typ, needed uint32
	var value uint32
	r, _, _ := getDeviceProperty.Call(h, ptr(dev), spdrpConfigFlags, ptr(&typ), ptr(&value), 4, ptr(&needed))
	if r == 0 {
		return 0
	}
	return value
}
func getInstanceID(h uintptr, dev *deviceInfo) string {
	buf := make([]uint16, 2048)
	var required uint32
	r, _, _ := getDeviceID.Call(h, ptr(dev), ptr(&buf[0]), uintptr(len(buf)), ptr(&required))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func drawPopup() {
	var ps paintstruct
	dc, _, _ := beginPaint.Call(hwnd, ptr(&ps))
	if dc == 0 {
		return
	}
	defer endPaint.Call(hwnd, ptr(&ps))
	// Paint to an offscreen buffer to avoid flickering on state changes.
	mem, _, _ := createCompatibleDC.Call(dc)
	if mem != 0 {
		bmp, _, _ := createCompatibleBitmap.Call(dc, uintptr(w), uintptr(h))
		if bmp != 0 {
			old, _, _ := selectObject.Call(mem, bmp)
			beginShapeFrame(mem)
			drawContent(mem)
			endShapeFrame()
			bitBlt.Call(dc, 0, 0, uintptr(w), uintptr(h), mem, 0, 0, 0x00CC0020)
			selectObject.Call(mem, old)
			deleteObject.Call(bmp)
		} else {
			beginShapeFrame(dc)
			drawContent(dc)
			endShapeFrame()
		}
		deleteDC.Call(mem)
	} else {
		beginShapeFrame(dc)
		drawContent(dc)
		endShapeFrame()
	}
}

// Live native build. The optional preview mode remains available for UI testing,
// but production packages use the verified working Windows PnP device controls.
var buildMode = "live"
var previewEnabled = true
var useLight bool
var cursorArrow uintptr
var cursorHand uintptr
var hoverClose bool
var pendingStateNotification bool
var pendingDesiredState bool
var position float64 = 1
var animFrom float64 = 1
var animTarget float64 = 1
var animStart time.Time
var animating bool

func previewMode() bool { return buildMode != "live" }

func checkLightTheme() bool {
	const hkeyCurrentUser = uintptr(0x80000001)
	var value uint32
	size := uint32(4)
	r, _, _ := regGetValue.Call(hkeyCurrentUser,
		ptr(utf(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)),
		ptr(utf("AppsUseLightTheme")), 0x18, 0, ptr(&value), ptr(&size))
	return r == 0 && value != 0
}

func windowsAccent() uint32 {
	// DWM reports 0xAARRGGBB, unlike GDI's 0x00BBGGRR COLORREF.
	var argb, opaque uint32
	if status, _, _ := dwmGetColorization.Call(ptr(&argb), ptr(&opaque)); status == 0 {
		r := (argb >> 16) & 0xff
		g := (argb >> 8) & 0xff
		b := argb & 0xff
		if r+g+b > 60 {
			return rgb(r, g, b)
		}
	}
	var color uint32
	size := uint32(4)
	if r, _, _ := regGetValue.Call(uintptr(0x80000001), ptr(utf(`Software\Microsoft\Windows\DWM`)), ptr(utf("AccentColor")), 0x18, 0, ptr(&color), ptr(&size)); r == 0 {
		return rgb(color&0xff, (color>>8)&0xff, (color>>16)&0xff)
	}
	return rgb(0, 120, 215) // Windows default blue
}
func accentInk(accent uint32) uint32 {
	r, g, b := float64(accent&255), float64((accent>>8)&255), float64((accent>>16)&255)
	if 0.2126*r+0.7152*g+0.0722*b > 164 {
		return rgb(20, 20, 20)
	}
	return rgb(255, 255, 255)
}
func applyTheme() {
	accent := windowsAccent()
	if useLight {
		colors = theme{
			background: rgb(249, 249, 249), panel: rgb(255, 255, 255),
			foreground: rgb(29, 29, 29), muted: rgb(101, 101, 101),
			stroke: rgb(222, 222, 222), accent: accent,
			inactive: rgb(220, 220, 220), white: rgb(255, 255, 255),
		}
	} else {
		colors = theme{
			background: rgb(25, 25, 25), // exact #191919 as requested
			panel:      rgb(37, 37, 37), foreground: rgb(248, 248, 248),
			muted: rgb(172, 172, 172), stroke: rgb(59, 59, 59),
			accent: accent, inactive: rgb(74, 74, 74), white: rgb(255, 255, 255),
		}
	}
}

func mixColor(a, b uint32, t float64) uint32 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	channel := func(n uint) uint32 {
		x := float64((a >> n) & 0xFF)
		y := float64((b >> n) & 0xFF)
		return uint32(x + (y-x)*t + 0.5)
	}
	return rgb(channel(0), channel(8), channel(16))
}

func startAnimation(on bool) {
	animFrom = position
	animTarget = 0
	if on {
		animTarget = 1
	}
	animStart = time.Now()
	animating = true
	setTimer.Call(hwnd, 2, 16, 0)
}
func tickAnimation() {
	if !animating {
		return
	}
	t := float64(time.Since(animStart).Milliseconds()) / 145
	if t >= 1 {
		position = animTarget
		animating = false
		killTimer.Call(hwnd, 2)
	} else {
		if t < 0 {
			t = 0
		}
		// Ease out with an understated, quick travel (145 ms max).
		t = 1 - (1-t)*(1-t)*(1-t)
		position = animFrom + (animTarget-animFrom)*t
	}
	redraw()
}

// Polished compact main screen and settings panel are implemented in gui.go.

func drawString(dc uintptr, s string, font uintptr, color uint32, left, top, right, bottom int32) {
	flushShapes()
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, uintptr(color))
	rc := rect{left, top, right, bottom}
	str := syscall.StringToUTF16(s)
	drawText.Call(dc, ptr(&str[0]), ^uintptr(0), ptr(&rc), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	selectObject.Call(dc, old)
}
func fallbackFillRounded(dc uintptr, left, top, right, bottom, radius int32, color, border uint32) {
	brush, _, _ := createSolidBrush.Call(uintptr(color))
	pen, _, _ := createPen.Call(0, 1, uintptr(border))
	oldB, _, _ := selectObject.Call(dc, brush)
	oldP, _, _ := selectObject.Call(dc, pen)
	roundRect.Call(dc, uintptr(left), uintptr(top), uintptr(right), uintptr(bottom), uintptr(radius), uintptr(radius))
	selectObject.Call(dc, oldB)
	selectObject.Call(dc, oldP)
	deleteObject.Call(brush)
	deleteObject.Call(pen)
}
func strokeRound(dc uintptr, left, top, right, bottom, radius int32, color uint32) {
	getStock := gdi32.NewProc("GetStockObject")
	nilBrush, _, _ := getStock.Call(5)
	pen, _, _ := createPen.Call(0, uintptr(d(2)), uintptr(color))
	oldB, _, _ := selectObject.Call(dc, nilBrush)
	oldP, _, _ := selectObject.Call(dc, pen)
	roundRect.Call(dc, uintptr(left), uintptr(top), uintptr(right), uintptr(bottom), uintptr(radius), uintptr(radius))
	selectObject.Call(dc, oldB)
	selectObject.Call(dc, oldP)
	deleteObject.Call(pen)
}
func fallbackFillEllipse(dc uintptr, left, top, right, bottom int32, color uint32) {
	brush, _, _ := createSolidBrush.Call(uintptr(color))
	pen, _, _ := createPen.Call(5, 1, 0)
	oldB, _, _ := selectObject.Call(dc, brush)
	oldP, _, _ := selectObject.Call(dc, pen)
	ellipse.Call(dc, uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
	selectObject.Call(dc, oldB)
	selectObject.Call(dc, oldP)
	deleteObject.Call(brush)
	deleteObject.Call(pen)
}
func drawLine(dc uintptr, x1, y1, x2, y2 int32, color uint32, thickness int32) {
	flushShapes()
	if thickness < 1 {
		thickness = 1
	}
	pen, _, _ := createPen.Call(0, uintptr(thickness), uintptr(color))
	old, _, _ := selectObject.Call(dc, pen)
	moveToEx.Call(dc, uintptr(x1), uintptr(y1), 0)
	lineTo.Call(dc, uintptr(x2), uintptr(y2))
	selectObject.Call(dc, old)
	deleteObject.Call(pen)
}
