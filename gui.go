package main

import (
	"syscall"
)

// Segoe Fluent Icons is built into Windows 11. The touch icon is U+E7C9.
// Rendering it as a font glyph at the *actual DPI* avoids raster scaling artifacts.
const (
	iconTouch    = "\ue7c9" // TouchPointer
	iconSettings = "\ue713" // Settings
	iconEdit     = "\ue70f" // Edit
	iconBack     = "\ue72b" // Back
	iconClose    = "\ue8bb" // ChromeClose
	iconCheck    = "\ue73e" // CheckMark
)

func inMainCircle(x, y int32) bool {
	dx := int64(x - d(176))
	dy := int64(y - d(181))
	r := int64(d(83))
	return dx*dx+dy*dy <= r*r
}
func inClose(x, y int32) bool { return x >= d(315) && x <= d(348) && y >= d(13) && y <= d(56) }
func inGear(x, y int32) bool  { return x >= d(12) && x <= d(145) && y >= d(370) && y <= d(416) }
func inBack(x, y int32) bool  { return x >= d(273) && x <= d(315) && y >= d(13) && y <= d(56) }
func settingsInteractive(x, y int32) bool {
	if !showSettings {
		return false
	}
	return x >= d(16) && x <= d(336) && ((y >= d(119) && y <= d(160)) || (y >= d(210) && y <= d(256)) || (y >= d(286) && y <= d(323)) || (y >= d(341) && y <= d(376)))
}
func handleSettingsClick(x, y int32) {
	if y >= d(119) && y <= d(161) {
		if x < d(124) {
			setTheme("system")
		} else if x < d(231) {
			setTheme("light")
		} else {
			setTheme("dark")
		}
		return
	}
	if y >= d(210) && y <= d(256) {
		beginRecording()
		return
	}
	if y >= d(278) && y <= d(329) {
		enabled := !config.StartWithWindows
		if err := setWindowsStartup(enabled); err != nil {
			showNotice("Couldn't update startup: "+err.Error(), true)
		} else {
			config.StartWithWindows = enabled
			saveSettings()
			notice = ""
			startHoverTimer()
		}
		return
	}
	if y >= d(329) && y <= d(385) {
		config.CloseToTray = !config.CloseToTray
		saveSettings()
		notice = ""
		startHoverTimer()
	}
}
func drawContent(dc uintptr) {
	setBkMode.Call(dc, transparent)
	fillRounded(dc, 0, 0, w, h, d(20), colors.background, colors.stroke)
	drawHeader(dc)
	if showSettings {
		drawSettings(dc)
	} else {
		drawMain(dc)
	}
}

func drawGlyphCentered(dc uintptr, glyph string, font uintptr, color uint32, left, top, right, bottom int32) {
	flushShapes()
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, uintptr(color))
	rc := rect{left, top, right, bottom}
	str := syscall.StringToUTF16(glyph)
	drawText.Call(dc, ptr(&str[0]), ^uintptr(0), ptr(&rc), 0x1|dtVCenter|dtSingleLine|dtNoPrefix)
	selectObject.Call(dc, old)
}

func drawGlyph(dc uintptr, glyph string, font uintptr, color uint32, left, top, right, bottom int32) {
	flushShapes()
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, uintptr(color))
	rc := rect{left, top, right, bottom}
	str := syscall.StringToUTF16(glyph)
	drawText.Call(dc, ptr(&str[0]), ^uintptr(0), ptr(&rc), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	selectObject.Call(dc, old)
}

func drawHeader(dc uintptr) {
	// Use the EXACT tray logo, rather than a different font hand glyph.
	if iconBadge != 0 {
		flushShapes()
		drawIconEx.Call(dc, uintptr(d(15)), uintptr(d(14)), iconBadge, uintptr(d(35)), uintptr(d(35)), 0, 0, 3)
	} else {
		fillRounded(dc, d(15), d(15), d(49), d(49), d(10), colors.accent, colors.accent)
	}
	drawString(dc, "TouchSwitch", fontTitle, colors.foreground, d(60), d(17), d(258), d(44))
	if showSettings {
		drawString(dc, "Settings", fontTiny, colors.muted, d(61), d(40), d(260), d(61))
	} else {
		drawString(dc, "Touchscreen control", fontTiny, colors.muted, d(61), d(41), d(265), d(60))
	}
	if showSettings && hoverValue(controlBack) > 0.01 {
		fillRounded(dc, d(275), d(16), d(308), d(51), d(8), hoverMix(colors.background, controlBack, .14), hoverMix(colors.background, controlBack, .14))
	}
	if hoverValue(controlClose) > 0.01 {
		fillRounded(dc, d(314), d(16), d(348), d(51), d(8), hoverMix(colors.background, controlClose, .14), hoverMix(colors.background, controlClose, .14))
	}
	if showSettings {
		drawGlyphCentered(dc, iconBack, fontIcon, colors.foreground, d(274), d(16), d(309), d(51))
	}
	drawGlyphCentered(dc, iconClose, fontIcon, colors.foreground, d(314), d(16), d(348), d(51))
	drawLine(dc, d(17), d(70), d(335), d(70), colors.stroke, d(1))
}
func drawChevron(dc uintptr, x, y int32, color uint32) {
	drawGlyphCentered(dc, iconBack, fontIcon, color, x-d(8), y-d(12), x+d(16), y+d(12))
}
func drawMain(dc uintptr) {
	cx, cy := d(176), d(181)
	rad := d(78) + int32(hoverValue(controlMain)*float64(d(3)))
	if pressedControl == controlMain {
		rad -= d(2)
	}
	base := colors.inactive
	base = mixColor(colors.inactive, colors.accent, position)
	if !found || busy {
		base = mixColor(base, colors.background, .13)
	}
	if pressedControl == controlMain {
		base = mixColor(base, colors.background, .18)
	} else {
		base = hoverMix(base, controlMain, .13)
	}
	fillEllipse(dc, cx-rad, cy-rad, cx+rad, cy+rad, base)
	// The finger artwork is an alpha antialiased vector-generated asset made
	// from the SAME source as the tray icon, not a substitute touch glyph.
	if iconMark != 0 {
		flushShapes()
		markSize := d(105)
		drawIconEx.Call(dc, uintptr(cx-markSize/2), uintptr(cy-markSize/2-d(1)), iconMark, uintptr(markSize), uintptr(markSize), 0, 0, 3)
	}
	if busy {
		drawStringCentered(dc, "Applying…", fontMain, colors.foreground, d(30), d(286), d(322), d(310))
	} else if !found {
		drawStringCentered(dc, "Touchscreen not found", fontMain, colors.foreground, d(30), d(286), d(322), d(310))
	} else if current.enabled {
		drawStringCentered(dc, "Touchscreen On", fontMain, colors.foreground, d(30), d(286), d(322), d(310))
	} else {
		drawStringCentered(dc, "Touchscreen Off", fontMain, colors.foreground, d(30), d(286), d(322), d(310))
	}
	hint := "Click to turn off"
	if !current.enabled {
		hint = "Click to turn on"
	}
	if busy {
		hint = "Changing device state"
	}
	if !found {
		hint = "No compatible HID touchscreen detected"
	}
	drawStringCentered(dc, hint, fontSub, colors.muted, d(20), d(314), d(332), d(340))
	drawLine(dc, d(16), d(370), d(336), d(370), colors.stroke, d(1))
	if hoverValue(controlSettings) > 0.01 {
		fillRounded(dc, d(16), d(376), d(147), d(412), d(9), hoverMix(colors.background, controlSettings, .105), hoverMix(colors.background, controlSettings, .105))
	}
	drawGlyphCentered(dc, iconSettings, fontIconSmall, colors.muted, d(23), d(378), d(45), d(406))
	drawString(dc, "Settings", fontTiny, colors.muted, d(52), d(378), d(142), d(408))
	drawStringRight(dc, shortcutText(), fontTiny, colors.muted, d(145), d(378), d(332), d(408))
}
func drawSettings(dc uintptr) {
	drawString(dc, "APPEARANCE", fontBadge, colors.muted, d(20), d(78), d(250), d(103))
	drawString(dc, "Theme", fontSub, colors.foreground, d(20), d(96), d(170), d(120))
	// Single selected pill slides between the three slots over ~180 ms.
	fillRounded(dc, d(17), d(121), d(335), d(161), d(10), colors.panel, colors.stroke)
	themeNames := []string{"System", "Light", "Dark"}
	themeCenter := float64(d(19)) + themeSlideAmount*float64(d(105))
	px := int32(themeCenter + .5)
	pillColor := colors.accent
	fillRounded(dc, px, d(123), px+d(104), d(159), d(8), pillColor, pillColor)
	for i, name := range themeNames {
		x := d(19 + 105*int32(i))
		rx := x + d(104)
		idx := controlSystem + i
		if hoverValue(idx) > 0.005 && themeIndex(config.Theme) != float64(i) {
			// Border-only hover so the moving indicator stays clearly visible.
			fillRounded(dc, x+d(2), d(124), rx-d(2), d(158), d(8),
				mixColor(colors.panel, colors.foreground, hoverValue(idx)*.09),
				mixColor(colors.panel, colors.foreground, hoverValue(idx)*.09))
		}
		fg := colors.foreground
		if absSlider(themeSlideAmount-float64(i)) < .44 {
			fg = accentInk(colors.accent)
		}
		drawStringCentered(dc, name, fontSub, fg, x, d(121), rx, d(161))
	}
	drawString(dc, "KEYBOARD SHORTCUT", fontBadge, colors.muted, d(20), d(174), d(250), d(196))
	drawString(dc, "Toggle touchscreen", fontSub, colors.foreground, d(20), d(189), d(220), d(213))
	outline := colors.stroke
	if recordingShortcut {
		outline = colors.accent
	}
	fillRounded(dc, d(18), d(216), d(334), d(256), d(10), hoverMix(colors.panel, controlShortcut, .055), outline)
	label := shortcutText()
	if recordingShortcut {
		label = "Press your shortcut…  (Esc to cancel)"
	}
	drawString(dc, label, fontSub, colors.foreground, d(31), d(218), d(306), d(255))
	drawGlyphCentered(dc, iconEdit, fontIconSmall, colors.muted, d(301), d(222), d(329), d(250))
	drawLine(dc, d(18), d(267), d(334), d(267), colors.stroke, d(1))
	if hoverValue(controlStartup) > 0.005 {
		bg := hoverMix(colors.background, controlStartup, .065)
		fillRounded(dc, d(17), d(275), d(334), d(328), d(10), bg, bg)
	}
	drawString(dc, "Start with Windows", fontSub, colors.foreground, d(20), d(282), d(260), d(309))
	drawSwitchAnimated(dc, d(284), d(287), startupAmount, controlStartup)
	drawLine(dc, d(18), d(330), d(334), d(330), colors.stroke, d(1))
	if hoverValue(controlTrayOption) > 0.005 {
		bg := hoverMix(colors.background, controlTrayOption, .065)
		fillRounded(dc, d(17), d(333), d(334), d(385), d(10), bg, bg)
	}
	drawString(dc, "Close to system tray", fontSub, colors.foreground, d(20), d(345), d(263), d(371))
	drawSwitchAnimated(dc, d(284), d(344), trayOptionAmount, controlTrayOption)
	caption := "Changes are saved automatically"
	captionColor := colors.muted
	if notice != "" {
		caption = notice
		if noticeBad {
			captionColor = rgb(210, 104, 98)
		} else {
			captionColor = colors.accent
		}
	}
	drawStringCentered(dc, caption, fontTiny, captionColor, d(12), d(384), d(340), d(413))
}
func drawSwitch(dc uintptr, x, y int32, on bool) {
	amount := 0.0
	if on {
		amount = 1
	}
	drawSwitchAnimated(dc, x, y, amount, controlNone)
}
func drawSwitchAnimated(dc uintptr, x, y int32, amount float64, control int) {
	// Track and thumb are GDI+ anti-aliased at actual display DPI.
	fill := mixColor(colors.inactive, colors.accent, amount)
	fill = hoverMix(fill, control, .075)
	fillRounded(dc, x, y, x+d(45), y+d(24), d(24), fill, fill)
	thumbX := x + d(3) + int32(float64(d(21))*amount+.5)
	fillEllipse(dc, thumbX, y+d(3), thumbX+d(18), y+d(21), colors.white)
}

func drawPencil(dc uintptr, x, y int32, color uint32) {
	drawGlyphCentered(dc, iconEdit, fontIconSmall, color, x-d(12), y-d(12), x+d(12), y+d(12))
}
func drawGear(dc uintptr, x, y int32, color uint32) {
	drawGlyphCentered(dc, iconSettings, fontIconSmall, color, x-d(12), y-d(12), x+d(12), y+d(12))
}
func strokeCircle(dc uintptr, l, t, r, b int32, color uint32) {
	pen, _, _ := createPen.Call(0, uintptr(d(2)), uintptr(color))
	getStock := gdi32.NewProc("GetStockObject")
	hollow, _, _ := getStock.Call(5)
	oldB, _, _ := selectObject.Call(dc, hollow)
	oldP, _, _ := selectObject.Call(dc, pen)
	ellipse.Call(dc, uintptr(l), uintptr(t), uintptr(r), uintptr(b))
	selectObject.Call(dc, oldB)
	selectObject.Call(dc, oldP)
	deleteObject.Call(pen)
}
func drawStringCentered(dc uintptr, s string, font uintptr, color uint32, l, t, r, b int32) {
	flushShapes()
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, uintptr(color))
	rc := rect{l, t, r, b}
	utfstr := utf(s)
	drawText.Call(dc, ptr(utfstr), ^uintptr(0), ptr(&rc), dtVCenter|dtSingleLine|dtNoPrefix|0x1)
	selectObject.Call(dc, old)
}
func drawStringRight(dc uintptr, s string, font uintptr, color uint32, l, t, r, b int32) {
	flushShapes()
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, uintptr(color))
	rc := rect{l, t, r, b}
	utfstr := utf(s)
	drawText.Call(dc, ptr(utfstr), ^uintptr(0), ptr(&rc), dtVCenter|dtSingleLine|dtNoPrefix|0x2)
	selectObject.Call(dc, old)
}

func absSlider(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
