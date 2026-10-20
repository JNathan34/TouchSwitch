package main

// Anti-aliased controls using the GDI+ runtime shipped with Windows.
// No runtime downloads, external fonts or DLLs are distributed.

import (
	"syscall"
	"unsafe"
)

var (
	gp                = syscall.NewLazyDLL("gdiplus.dll")
	gpStartup         = gp.NewProc("GdiplusStartup")
	gpShutdown        = gp.NewProc("GdiplusShutdown")
	gpGraphicsFromHDC = gp.NewProc("GdipCreateFromHDC")
	gpDeleteGraphics  = gp.NewProc("GdipDeleteGraphics")
	gpSmoothing       = gp.NewProc("GdipSetSmoothingMode")
	gpPixelOffset     = gp.NewProc("GdipSetPixelOffsetMode")
	gpFlush           = gp.NewProc("GdipFlush")
	gpCreateBrush     = gp.NewProc("GdipCreateSolidFill")
	gpDeleteBrush     = gp.NewProc("GdipDeleteBrush")
	gpFillEllipse     = gp.NewProc("GdipFillEllipseI")
	gpFillRectangle   = gp.NewProc("GdipFillRectangleI")
)

type gdipInitInput struct {
	Version            uint32
	Debug              uintptr
	SuppressBackground uint32
	SuppressExternal   uint32
}

var gdipToken uintptr
var shapeGfx uintptr

func initGDIPlus() {
	inp := gdipInitInput{Version: 1}
	status, _, _ := gpStartup.Call(uintptr(unsafe.Pointer(&gdipToken)), uintptr(unsafe.Pointer(&inp)), 0)
	if status != 0 {
		gdipToken = 0
	}
}
func shutdownGDIPlus() {
	if gdipToken != 0 {
		gpShutdown.Call(gdipToken)
		gdipToken = 0
	}
}
func beginShapeFrame(dc uintptr) {
	if gdipToken == 0 {
		return
	}
	r, _, _ := gpGraphicsFromHDC.Call(dc, uintptr(unsafe.Pointer(&shapeGfx)))
	if r != 0 {
		shapeGfx = 0
		return
	}
	gpSmoothing.Call(shapeGfx, 4)   // anti-aliased paths
	gpPixelOffset.Call(shapeGfx, 4) // half-pixel sampling; crisp round edges
}
func flushShapes() {
	if shapeGfx != 0 {
		gpFlush.Call(shapeGfx, 1)
	}
}
func endShapeFrame() {
	if shapeGfx != 0 {
		gpDeleteGraphics.Call(shapeGfx)
		shapeGfx = 0
	}
}
func toArgb(c uint32) uintptr {
	r := (c & 0x000000ff)
	g := (c & 0x0000ff00)
	b := (c & 0x00ff0000) >> 16
	return uintptr(0xff000000 | (r << 16) | g | b)
}
func withBrush(c uint32, f func(uintptr)) {
	var brush uintptr
	status, _, _ := gpCreateBrush.Call(toArgb(c), uintptr(unsafe.Pointer(&brush)))
	if status != 0 || brush == 0 {
		return
	}
	f(brush)
	gpDeleteBrush.Call(brush)
}
func gEllipse(br uintptr, l, t, r, b int32) {
	if r > l && b > t {
		gpFillEllipse.Call(shapeGfx, br, uintptr(l), uintptr(t), uintptr(r-l), uintptr(b-t))
	}
}
func gRect(br uintptr, l, t, r, b int32) {
	if r > l && b > t {
		gpFillRectangle.Call(shapeGfx, br, uintptr(l), uintptr(t), uintptr(r-l), uintptr(b-t))
	}
}

// Four antialiased quarter-circles seamlessly covered with two solid spans.
func solidRound(l, t, r, b, diam int32, brush uintptr) {
	width, height := r-l, b-t
	if width <= 0 || height <= 0 {
		return
	}
	if diam > width {
		diam = width
	}
	if diam > height {
		diam = height
	}
	if diam < 1 {
		gRect(brush, l, t, r, b)
		return
	}
	rad := diam / 2
	if rad < 1 {
		gRect(brush, l, t, r, b)
		return
	}
	gEllipse(brush, l, t, l+diam, t+diam)
	gEllipse(brush, r-diam, t, r, t+diam)
	gEllipse(brush, l, b-diam, l+diam, b)
	gEllipse(brush, r-diam, b-diam, r, b)
	gRect(brush, l+rad, t, r-rad, b)
	gRect(brush, l, t+rad, r, b-rad)
}
func fillRounded(dc uintptr, l, t, r, b, diam int32, color, border uint32) {
	if shapeGfx == 0 {
		fallbackFillRounded(dc, l, t, r, b, diam, color, border)
		return
	}
	if border != color && r-l >= 5 && b-t >= 5 {
		withBrush(border, func(br uintptr) { solidRound(l, t, r, b, diam, br) })
		withBrush(color, func(br uintptr) { solidRound(l+1, t+1, r-1, b-1, diam-2, br) })
	} else {
		withBrush(color, func(br uintptr) { solidRound(l, t, r, b, diam, br) })
	}
}
func fillEllipse(dc uintptr, l, t, r, b int32, color uint32) {
	if shapeGfx == 0 {
		fallbackFillEllipse(dc, l, t, r, b, color)
		return
	}
	withBrush(color, func(br uintptr) { gEllipse(br, l, t, r, b) })
}
