package main

// A native non-activating short popup. Unlike Windows notification balloons,
// the popup is displayed even when Focus / Do Not Disturb is enabled.
// It doesn't enter Windows Notification Center or interrupt the current app.

import (
	"syscall"
	"time"
	"unsafe"
)

const (
	swShowNoActivate = 4
	wsExNoActivate   = 0x08000000
	wmMouseActivate  = 0x0021
	wmEraseBkgnd     = 0x0014
	maNoActivate     = 3
)

var (
	toastWnd          uintptr
	toastW            int32
	toastH            int32
	toastOn           bool
	toastAvailable    bool
	toastStarted      time.Time
	toastTargetX      int32
	toastOutsideX     int32
	toastY            int32
	setWindowPos      = user32.NewProc("SetWindowPos")
	monitorFromWindow = user32.NewProc("MonitorFromWindow")
	getMonitorInfo    = user32.NewProc("GetMonitorInfoW")
)

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

func createToastWindow(instance uintptr) {
	toastW, toastH = d(296), d(74)
	cls := utf("TouchSwitch.StateToast.V32")
	wc := wndclassex{Size: uint32(unsafe.Sizeof(wndclassex{})), WndProc: syscall.NewCallback(toastProcedure), Instance: instance, ClassName: cls}
	wc.Cursor = cursorArrow
	if r, _, _ := registerClass.Call(ptr(&wc)); r == 0 {
		return
	}
	toastWnd, _, _ = createWindow.Call(wsExNoActivate|wsExToolWindow|wsExTopmost, ptr(cls), ptr(utf("TouchSwitch status")), wsPopup, 0, 0, uintptr(toastW), uintptr(toastH), 0, 0, instance, 0)
	if toastWnd == 0 {
		return
	}
	toastAvailable = true
	var rounded int32 = 2
	hr, _, _ := setDwmAttribute.Call(toastWnd, 33, ptr(&rounded), 4)
	if hr != 0 {
		region, _, _ := createRoundRectRgn.Call(0, 0, uintptr(toastW+1), uintptr(toastH+1), uintptr(d(16)), uintptr(d(16)))
		if region != 0 {
			setWindowRgn.Call(toastWnd, region, 1)
		}
	}
}

// The popup animates independently, does not grab keyboard focus, and
// bypasses Notification Center so Do Not Disturb does not suppress it.
func showTouchToast(on bool) {
	if !toastAvailable {
		return
	}
	toastOn = on
	var work rect
	work.Right = int32(systemMetric(0))
	work.Bottom = int32(systemMetric(1))
	m, _, _ := monitorFromWindow.Call(hwnd, 2)
	if m != 0 {
		info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		if ok, _, _ := getMonitorInfo.Call(m, ptr(&info)); ok != 0 {
			work = info.Work
		}
	}
	toastTargetX = work.Right - toastW - d(20)
	toastOutsideX = work.Right + d(14)
	toastY = work.Bottom - toastH - d(18)
	toastStarted = time.Now()
	setWindowPos.Call(toastWnd, ^uintptr(0), uintptr(toastOutsideX), uintptr(toastY), uintptr(toastW), uintptr(toastH), 0x0010)
	invalidateRect.Call(toastWnd, 0, 0)
	showWindow.Call(toastWnd, swShowNoActivate)
	killTimer.Call(toastWnd, 104)
	setTimer.Call(toastWnd, 104, 15, 0)
}

func easeOutCubic(t float64) float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	u := 1 - t
	return 1 - u*u*u
}
func tickToastSlide() {
	// 180 ms fly-in, 740 ms readable, 180 ms fly-out (~1.1s total).
	ms := time.Since(toastStarted).Milliseconds()
	if ms >= 1100 {
		killTimer.Call(toastWnd, 104)
		showWindow.Call(toastWnd, swHide)
		return
	}
	x := toastTargetX
	if ms < 180 {
		t := easeOutCubic(float64(ms) / 180)
		x = toastOutsideX + int32(float64(toastTargetX-toastOutsideX)*t)
	} else if ms >= 920 {
		t := easeOutCubic(float64(ms-920) / 180)
		x = toastTargetX + int32(float64(toastOutsideX-toastTargetX)*t)
	}
	setWindowPos.Call(toastWnd, 0, uintptr(x), uintptr(toastY), 0, 0,
		0x0001|0x0004|0x0010) // NOSIZE | NOZORDER | NOACTIVATE
}
func toastProcedure(window uintptr, event uint32, wp, lp uintptr) uintptr {
	switch event {
	case wmEraseBkgnd:
		return 1
	case wmMouseActivate:
		return maNoActivate
	case wmPaint:
		var ps paintstruct
		dc, _, _ := beginPaint.Call(window, ptr(&ps))
		if dc != 0 {
			mem, _, _ := createCompatibleDC.Call(dc)
			if mem != 0 {
				bmp, _, _ := createCompatibleBitmap.Call(dc, uintptr(toastW), uintptr(toastH))
				if bmp != 0 {
					old, _, _ := selectObject.Call(mem, bmp)
					beginShapeFrame(mem)
					drawToastContent(mem)
					endShapeFrame()
					bitBlt.Call(dc, 0, 0, uintptr(toastW), uintptr(toastH), mem, 0, 0, 0x00CC0020)
					selectObject.Call(mem, old)
					deleteObject.Call(bmp)
				} else {
					beginShapeFrame(dc)
					drawToastContent(dc)
					endShapeFrame()
				}
				deleteDC.Call(mem)
			} else {
				beginShapeFrame(dc)
				drawToastContent(dc)
				endShapeFrame()
			}
			endPaint.Call(window, ptr(&ps))
		}
		return 0
	case wmTimer:
		if wp == 104 {
			tickToastSlide()
			return 0
		}
	case wmDestroy:
		toastAvailable = false
		return 0
	}
	r, _, _ := defWindowProc.Call(window, uintptr(event), wp, lp)
	return r
}
func drawToastContent(dc uintptr) {
	setBkMode.Call(dc, transparent)
	fillRounded(dc, 0, 0, toastW, toastH, d(16), colors.panel, colors.stroke)
	badge := iconBadge
	if !toastOn {
		badge = iconBadgeOff
	}
	if badge != 0 {
		flushShapes()
		drawIconEx.Call(dc, uintptr(d(13)), uintptr(d(14)), badge, uintptr(d(45)), uintptr(d(45)), 0, 0, 3)
	}
	title := "Touchscreen off"
	info := "Touch input disabled"
	if toastOn {
		title = "Touchscreen on"
		info = "Touch input enabled"
	}
	drawString(dc, title, fontMain, colors.foreground, d(73), d(13), toastW-d(16), d(41))
	drawString(dc, info, fontSub, colors.muted, d(73), d(39), toastW-d(16), d(65))
}
