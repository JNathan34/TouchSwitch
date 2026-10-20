package main

// A small state animator for native GDI+ controls. Timers are active only while
// visual properties are changing, rather than repainting continually.
import (
	"math"
	"unsafe"
)

const (
	controlNone = iota
	controlMain
	controlSettings
	controlClose
	controlBack
	controlSystem
	controlLight
	controlDark
	controlShortcut
	controlStartup
	controlTrayOption
	controlCount
)

var hoverControl int
var pressedControl int
var mouseTracking bool
var hoverAmounts [controlCount]float64
var startupAmount float64
var trayOptionAmount float64
var hoverTimer bool
var themeSlideAmount float64 // 0 System, 1 Light, 2 Dark

func themeIndex(t string) float64 {
	switch t {
	case "light":
		return 1
	case "dark":
		return 2
	default:
		return 0
	}
}

type trackMouseEventData struct {
	Size      uint32
	Flags     uint32
	Window    uintptr
	HoverTime uint32
}

func initHoverAnimation() {
	themeSlideAmount = themeIndex(config.Theme)
	if config.StartWithWindows {
		startupAmount = 1
	}
	if config.CloseToTray {
		trayOptionAmount = 1
	}
}
func hitControl(x, y int32) int {
	if inClose(x, y) {
		return controlClose
	}
	if showSettings {
		if inBack(x, y) {
			return controlBack
		}
		if x >= d(17) && x <= d(335) {
			if y >= d(121) && y <= d(161) {
				if x < d(124) {
					return controlSystem
				}
				if x < d(231) {
					return controlLight
				}
				return controlDark
			}
			if y >= d(208) && y <= d(259) {
				return controlShortcut
			}
			if y >= d(278) && y <= d(328) {
				return controlStartup
			}
			if y >= d(333) && y <= d(385) {
				return controlTrayOption
			}
		}
	} else {
		if inGear(x, y) {
			return controlSettings
		}
		if inMainCircle(x, y) {
			return controlMain
		}
	}
	return controlNone
}
func updatePointer(x, y int32) {
	if !mouseTracking {
		req := trackMouseEventData{Size: uint32(unsafe.Sizeof(trackMouseEventData{})), Flags: 2, Window: hwnd}
		if ok, _, _ := trackMouseEvent.Call(ptr(&req)); ok != 0 {
			mouseTracking = true
		}
	}
	next := hitControl(x, y)
	setHoverControl(next)
	if next != controlNone {
		setCursor.Call(cursorHand)
	} else {
		setCursor.Call(cursorArrow)
	}
}
func setHoverControl(id int) {
	if hoverControl == id {
		return
	}
	hoverControl = id
	hoverSwitch = id == controlMain
	hoverClose = id == controlClose
	hoverBack = id == controlBack
	hoverSettings = id == controlSettings
	startHoverTimer()
}
func startHoverTimer() {
	if !hoverTimer && hwnd != 0 {
		setTimer.Call(hwnd, 4, 16, 0)
		hoverTimer = true
	}
	redraw()
}
func hoverValue(id int) float64 {
	if id <= 0 || id >= controlCount {
		return 0
	}
	return hoverAmounts[id]
}
func tickHoverAnimation() {
	animating := false
	themeTarget := themeIndex(config.Theme)
	themeSlideAmount += (themeTarget - themeSlideAmount) * .24
	if math.Abs(themeSlideAmount-themeTarget) < .009 {
		themeSlideAmount = themeTarget
	} else {
		animating = true
	}
	for i := 1; i < controlCount; i++ {
		target := 0.0
		if hoverControl == i {
			target = 1
		}
		before := hoverAmounts[i]
		hoverAmounts[i] += (target - hoverAmounts[i]) * 0.29
		if math.Abs(hoverAmounts[i]-target) < 0.016 {
			hoverAmounts[i] = target
		}
		if hoverAmounts[i] != target || hoverAmounts[i] != before {
			animating = true
		}
	}
	for _, v := range []struct {
		a    *float64
		want bool
	}{
		{&startupAmount, config.StartWithWindows},
		{&trayOptionAmount, config.CloseToTray},
	} {
		target := 0.0
		if v.want {
			target = 1
		}
		*v.a += (target - *v.a) * 0.29
		if math.Abs(*v.a-target) < 0.016 {
			*v.a = target
		} else {
			animating = true
		}
	}
	redraw()
	if !animating {
		killTimer.Call(hwnd, 4)
		hoverTimer = false
	}
}
func hoverMix(normal uint32, id int, strength float64) uint32 {
	return mixColor(normal, colors.foreground, hoverValue(id)*strength)
}
