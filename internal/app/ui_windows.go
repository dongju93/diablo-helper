//go:build windows

package app

import (
	"math"
	"unsafe"

	"github.com/dongju93/diablo-helper/internal/config"
)

const (
	uiFontBaseHeight        = -15
	uiTitleFontBaseHeight   = -28
	uiSectionFontBaseHeight = -16
	uiFontMinScale          = 0.8
	uiFontMaxScale          = 4.0
)

var (
	uiBackground    = rgb(243, 243, 243)
	uiPanel         = rgb(255, 255, 255)
	uiPanelAlt      = rgb(251, 251, 251)
	uiBorder        = rgb(229, 229, 229)
	uiBorderStrong  = rgb(199, 199, 199)
	uiText          = rgb(32, 32, 32)
	uiTextSubtle    = rgb(96, 96, 96)
	uiAccent        = rgb(0, 103, 192)
	uiAccentPressed = rgb(0, 90, 158)
	uiAccentHover   = rgb(0, 95, 184)
	uiAccentSoft    = rgb(232, 241, 252)
	uiSuccess       = rgb(16, 124, 16)
	uiWarning       = rgb(159, 98, 0)
	uiStatusRunning = rgb(16, 124, 16) // 초록 – 동작 중
	uiStatusPaused  = rgb(180, 130, 0) // 노랑 – 일시정지
	uiStatusStopped = rgb(196, 43, 28) // 빨강 – 정지
)

type gdiResources struct {
	font            uintptr
	titleFont       uintptr
	sectionFont     uintptr
	bgBrush         uintptr
	panelBrush      uintptr
	editBrush       uintptr
	borderPen       uintptr
	borderStrongPen uintptr
	borderBrush     uintptr
	accentBrush     uintptr
	accentPen       uintptr
	fontScale       float64
}

func rgb(red byte, green byte, blue byte) uintptr {
	return uintptr(uint32(red) | uint32(green)<<8 | uint32(blue)<<16)
}

func int32Arg(value int32) uintptr {
	return uintptr(uint32(value))
}

func (g *gdiResources) init() {
	scale := g.fontScale
	if scale <= 0 {
		scale = 1
	}
	if g.font == 0 {
		g.font = createUIFont("Malgun Gothic", scaledFontHeight(uiFontBaseHeight, scale), fwNormal)
	}
	if g.titleFont == 0 {
		g.titleFont = createUIFont("Segoe UI Variable Display", scaledFontHeight(uiTitleFontBaseHeight, scale), fwSemiBold)
	}
	if g.sectionFont == 0 {
		g.sectionFont = createUIFont("Malgun Gothic", scaledFontHeight(uiSectionFontBaseHeight, scale), fwSemiBold)
	}
	if g.bgBrush == 0 {
		g.bgBrush = createBrush(uiBackground)
	}
	if g.panelBrush == 0 {
		g.panelBrush = createBrush(uiPanel)
	}
	if g.editBrush == 0 {
		g.editBrush = createBrush(uiPanel)
	}
	if g.borderPen == 0 {
		g.borderPen = createPen(uiBorder, 1)
	}
	if g.borderStrongPen == 0 {
		g.borderStrongPen = createPen(uiBorderStrong, 1)
	}
	if g.borderBrush == 0 {
		g.borderBrush = createBrush(uiBorder)
	}
	if g.accentBrush == 0 {
		g.accentBrush = createBrush(uiAccent)
	}
	if g.accentPen == 0 {
		g.accentPen = createPen(uiAccent, 1)
	}
}

func (g *gdiResources) applyScale(scale float64) bool {
	if scale <= 0 {
		scale = 1
	}
	scale = clampFloat(scale, uiFontMinScale, uiFontMaxScale)
	rebuildFonts := g.font == 0 ||
		g.titleFont == 0 ||
		g.sectionFont == 0 ||
		scaledFontHeight(uiFontBaseHeight, g.fontScale) != scaledFontHeight(uiFontBaseHeight, scale) ||
		scaledFontHeight(uiTitleFontBaseHeight, g.fontScale) != scaledFontHeight(uiTitleFontBaseHeight, scale) ||
		scaledFontHeight(uiSectionFontBaseHeight, g.fontScale) != scaledFontHeight(uiSectionFontBaseHeight, scale)

	g.fontScale = scale
	if rebuildFonts {
		g.disposeFonts()
	}
	g.init()
	return rebuildFonts
}

func (a *application) applyUIScale(scale float64) {
	if a.gdi.applyScale(scale) {
		a.updateControlFonts()
	}
}

func scaledFontHeight(base int, scale float64) int32 {
	if scale <= 0 {
		scale = 1
	}
	height := int32(math.Round(math.Abs(float64(base)) * scale))
	if height < 1 {
		height = 1
	}
	if base < 0 {
		return -height
	}
	return height
}

func createUIFont(face string, height int32, weight int) uintptr {
	// GDI reports allocation failure as a zero handle. WM_SETFONT and paint paths
	// already tolerate zero by falling back to system defaults.
	font, _, _ := procCreateFontW.Call(
		int32Arg(height),
		0,
		0,
		0,
		uintptr(weight),
		0,
		0,
		0,
		defaultCharset,
		0,
		0,
		cleartypeQuality,
		0,
		uintptr(unsafe.Pointer(utf16Ptr(face))),
	)
	return font
}

func createBrush(color uintptr) uintptr {
	// GDI reports allocation failure as a zero handle. Drawing with a zero brush
	// is best-effort UI degradation rather than a user-actionable failure.
	brush, _, _ := procCreateSolidBrush.Call(color)
	return brush
}

func createPen(color uintptr, width int) uintptr {
	// GDI reports allocation failure as a zero handle. Drawing with a zero pen is
	// best-effort UI degradation rather than a user-actionable failure.
	pen, _, _ := procCreatePen.Call(psSolid, uintptr(width), color)
	return pen
}

func deleteGDIObject(handle uintptr) {
	if handle != 0 {
		// Cleanup failure is non-actionable; handles are process-local UI resources.
		_, _, _ = procDeleteObject.Call(handle)
	}
}

func (g *gdiResources) disposeFonts() {
	deleteGDIObject(g.font)
	deleteGDIObject(g.titleFont)
	deleteGDIObject(g.sectionFont)
	g.font = 0
	g.titleFont = 0
	g.sectionFont = 0
}

func (g *gdiResources) dispose() {
	g.disposeFonts()
	deleteGDIObject(g.bgBrush)
	deleteGDIObject(g.panelBrush)
	deleteGDIObject(g.editBrush)
	deleteGDIObject(g.borderPen)
	deleteGDIObject(g.borderStrongPen)
	deleteGDIObject(g.borderBrush)
	deleteGDIObject(g.accentBrush)
	deleteGDIObject(g.accentPen)
	g.bgBrush = 0
	g.panelBrush = 0
	g.editBrush = 0
	g.borderPen = 0
	g.borderStrongPen = 0
	g.borderBrush = 0
	g.accentBrush = 0
	g.accentPen = 0
}

func (a *application) updateControlFonts() {
	if a.gdi.font == 0 {
		return
	}
	for _, hwnd := range []uintptr{
		a.controls.startLabel,
		a.controls.startButton,
		a.controls.stopLabel,
		a.controls.stopButton,
		a.controls.pauseButton,
		a.controls.loadButton,
		a.controls.saveButton,
		a.controls.bulkLabel,
		a.controls.bulkInterval,
		a.controls.bulkMsLabel,
		a.controls.bulkSkillGapLbl,
		a.controls.bulkSkillGap,
		a.controls.bulkGapMsLabel,
		a.controls.inputHoldLabel,
		a.controls.inputHold,
		a.controls.inputHoldMsLbl,
		a.controls.applyBulk,
		a.controls.skillUseHdr,
		a.controls.skillNumHdr,
		a.controls.skillKeyHdr,
		a.controls.skillIntHdr,
		a.controls.skillHoldHdr,
		a.controls.pauseLabel,
		a.controls.clickerStartLabel,
		a.controls.clickerStartButton,
		a.controls.clickerStopLabel,
		a.controls.clickerStopButton,
		a.controls.clickerKeyLabel,
		a.controls.clickerKeyButton,
		a.controls.clickerIntervalLabel,
		a.controls.clickerInterval,
		a.controls.clickerMsLabel,
		a.controls.clickerHoldLabel,
		a.controls.clickerHold,
		a.controls.clickerHoldMsLabel,
		a.controls.statusLabel,
		a.controls.status,
	} {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.menuLabels {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.menuButtons {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.skillEnabled {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.skillNums {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.skillButtons {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.skillInterval {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.skillMsLbls {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.skillHold {
		setControlFont(hwnd, a.gdi.font)
	}
	for _, hwnd := range a.controls.skillHoldMsLbls {
		setControlFont(hwnd, a.gdi.font)
	}
}

func setControlFont(hwnd uintptr, font uintptr) {
	if hwnd != 0 && font != 0 {
		sendMessage(hwnd, wmSetFont, font, 1)
	}
}

func (a *application) paint(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var client rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))

	lo := computeLayout(int(client.Right), int(client.Bottom), a.currentDPI(hwnd))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&client)), a.gdi.bgBrush)

	// Panels
	a.drawPanel(hdc, lo.leftX, lo.y(92), lo.leftW, lo.h(126))
	a.drawPanel(hdc, lo.leftX, lo.y(menuPanelY), lo.leftW, lo.h(menuPanelH))
	a.drawPanel(hdc, lo.rx, lo.y(92), lo.rw, lo.h(498))
	a.drawPanel(hdc, lo.rx, lo.y(clickerPanelY), lo.rw, lo.h(clickerPanelH))
	a.drawPanel(hdc, lo.rx, lo.y(pausePanelY), lo.rw, lo.h(pausePanelH))
	a.drawPanel(hdc, lo.leftX, lo.y(statusBarY), lo.statusBarW, lo.h(40))
	a.drawAccentMark(hdc, lo.x(28), lo.y(26), lo.w(4), lo.h(24))

	// Dividers – left column
	a.drawDivider(hdc, lo.x(layoutLX+20), lo.y(174), lo.w(layoutLW-40))
	for y := menuFirstY + 38; y <= menuFirstY+38+(len(menuControls)-2)*40; y += 40 {
		a.drawDivider(hdc, lo.x(layoutLX+20), lo.y(y), lo.w(layoutLW-40))
	}

	// Dividers – right column (scales with rw)
	a.drawDivider(hdc, lo.rx+lo.w(20), lo.y(skillFirstRowY-6), lo.rw-lo.w(40))
	for y := skillFirstRowY + 38; y <= skillFirstRowY+38+(config.MaxSkills-1)*skillRowGap; y += skillRowGap {
		a.drawDivider(hdc, lo.rx+lo.w(20), lo.y(y), lo.rw-lo.w(40))
	}
	a.drawDivider(hdc, lo.rx+lo.w(20), lo.y(clickerHotkeyY+38), lo.rw-lo.w(40))

	// Input frames
	a.drawInputFrame(hdc, lo.bulkEditX-lo.w(8), lo.y(bulkIntervalEditY-6), lo.w(inputFrameWidth(bulkEditW)), lo.h(32))
	a.drawInputFrame(hdc, lo.bulkEditX-lo.w(8), lo.y(bulkSkillGapEditY-6), lo.w(inputFrameWidth(bulkEditW)), lo.h(32))
	a.drawInputFrame(hdc, lo.bulkEditX-lo.w(8), lo.y(inputHoldEditY-6), lo.w(inputFrameWidth(bulkEditW)), lo.h(32))
	for y := skillFirstRowY; y < skillFirstRowY+config.MaxSkills*skillRowGap; y += skillRowGap {
		a.drawInputFrame(hdc, lo.skillIntervalX-lo.w(8), lo.y(y+1), lo.w(inputFrameWidth(skillEditW)), lo.h(32))
		a.drawInputFrame(hdc, lo.skillHoldX-lo.w(8), lo.y(y+1), lo.w(inputFrameWidth(skillHoldEditW)), lo.h(32))
	}
	a.drawInputFrame(hdc, lo.clickerIntEditX-lo.w(8), lo.y(clickerSettingY+1), lo.w(inputFrameWidth(clickerIntEditW)), lo.h(32))
	a.drawInputFrame(hdc, lo.clickerHoldEditX-lo.w(8), lo.y(clickerSettingY+1), lo.w(inputFrameWidth(clickerHoldEditW)), lo.h(32))

	a.drawStatusDot(hdc, lo.statusDotX, lo.y(statusBarY+19), lo.s(10))

	drawText(hdc, "Diablo Helper", a.gdi.titleFont, uiText, lo.x(40), lo.y(18), lo.w(300), lo.h(40), dtSingleLine|dtNoPrefix)
	drawText(hdc, "시작/종료 키", a.gdi.sectionFont, uiText, lo.x(layoutLX+20), lo.y(108), lo.w(210), lo.h(28), dtSingleLine|dtNoPrefix)
	drawText(hdc, "게임 메뉴 키", a.gdi.sectionFont, uiText, lo.x(layoutLX+20), lo.y(menuTitleY), lo.w(210), lo.h(28), dtSingleLine|dtNoPrefix)
	drawText(hdc, "기술 키", a.gdi.sectionFont, uiText, lo.rx+lo.w(20), lo.y(108), lo.w(160), lo.h(28), dtSingleLine|dtNoPrefix)
	drawText(hdc, "클릭 반복", a.gdi.sectionFont, uiText, lo.rx+lo.w(20), lo.y(clickerTitleY), lo.w(210), lo.h(28), dtSingleLine|dtNoPrefix)
	drawText(hdc, "일시정지 키", a.gdi.sectionFont, uiText, lo.rx+lo.w(20), lo.y(pauseTitleY), lo.w(180), lo.h(28), dtSingleLine|dtNoPrefix)
}

func (a *application) drawPanel(hdc uintptr, x int, y int, width int, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	corner := maxInt(8, height/8)
	oldBrush, _, _ := procSelectObject.Call(hdc, a.gdi.panelBrush)
	oldPen, _, _ := procSelectObject.Call(hdc, a.gdi.borderPen)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height), uintptr(corner), uintptr(corner))
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
}

func (a *application) drawInputFrame(hdc uintptr, x int, y int, width int, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	corner := maxInt(4, height/4)
	oldBrush, _, _ := procSelectObject.Call(hdc, a.gdi.panelBrush)
	oldPen, _, _ := procSelectObject.Call(hdc, a.gdi.borderStrongPen)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height), uintptr(corner), uintptr(corner))
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
}

func inputFrameWidth(editWidth int) int {
	return editWidth + 24
}

func (a *application) drawDivider(hdc uintptr, x int, y int, width int) {
	if width <= 0 {
		return
	}
	rc := rect{Left: int32(x), Top: int32(y), Right: int32(x + width), Bottom: int32(y + 1)}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), a.gdi.borderBrush)
}

func (a *application) drawAccentMark(hdc uintptr, x int, y int, width int, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	corner := maxInt(2, height/6)
	oldBrush, _, _ := procSelectObject.Call(hdc, a.gdi.accentBrush)
	oldPen, _, _ := procSelectObject.Call(hdc, a.gdi.accentPen)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height), uintptr(corner), uintptr(corner))
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
}

func (a *application) drawStatusDot(hdc uintptr, x int, y int, size int) {
	if size <= 0 {
		return
	}
	color := uiTextSubtle
	switch {
	case a.capture.valid():
		color = uiAccent
	case a.anyPaused():
		color = uiWarning
	case a.runner.Running() || a.clicker.Running():
		color = uiSuccess
	}
	brush := createBrush(color)
	pen := createPen(color, 1)
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procEllipse.Call(hdc, uintptr(x), uintptr(y), uintptr(x+size), uintptr(y+size))
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	deleteGDIObject(pen)
	deleteGDIObject(brush)
}

func drawText(hdc uintptr, text string, font uintptr, color uintptr, x int, y int, width int, height int, flags uintptr) {
	if text == "" {
		return
	}
	textPtr := utf16Ptr(text)
	rect := rect{
		Left:   int32(x),
		Top:    int32(y),
		Right:  int32(x + width),
		Bottom: int32(y + height),
	}
	oldFont, _, _ := procSelectObject.Call(hdc, font)
	procSetBkMode.Call(hdc, transparent)
	procSetTextColor.Call(hdc, color)
	procDrawTextW.Call(
		hdc,
		uintptr(unsafe.Pointer(textPtr)),
		^uintptr(0),
		uintptr(unsafe.Pointer(&rect)),
		flags,
	)
	procSelectObject.Call(hdc, oldFont)
}

func (a *application) colorStatic(hdc uintptr, hwndCtl uintptr) uintptr {
	a.gdi.init()
	procSetBkMode.Call(hdc, transparent)
	if hwndCtl != 0 && hwndCtl == a.controls.status {
		color := a.statusTextColor()
		procSetTextColor.Call(hdc, color)
	} else {
		procSetTextColor.Call(hdc, uiText)
	}
	return a.gdi.panelBrush
}

// statusTextColor returns the colour that should be used for the status text
// based on the current runner/clicker state.
func (a *application) statusTextColor() uintptr {
	switch {
	case a.anyPaused():
		return uiStatusPaused
	case a.runner.Running() || a.clicker.Running():
		return uiStatusRunning
	default:
		return uiStatusStopped
	}
}

func (a *application) anyPaused() bool {
	return a.runner.Paused() || a.clicker.Paused()
}

func (a *application) colorEdit(hdc uintptr) uintptr {
	a.gdi.init()
	procSetBkColor.Call(hdc, uiPanel)
	procSetTextColor.Call(hdc, uiText)
	return a.gdi.editBrush
}

func (a *application) drawButton(item *drawItemStruct) {
	if item == nil || item.HDC == 0 {
		return
	}
	a.gdi.init()

	text, _ := getWindowText(item.HwndItem)
	id := int(item.CtlID)
	selected := item.ItemState&odsSelected != 0
	disabled := item.ItemState&odsDisabled != 0
	focused := item.ItemState&odsFocus != 0
	hovered := item.ItemState&odsHotLight != 0
	capturing := a.captureControlID(a.capture) == id

	// Toggle switch (pill track + sliding knob, right=ON blue)
	if a.isToggleButton(id) {
		idx := id - idSkillEnabledBase
		on := idx >= 0 && idx < len(a.skillEnabled) && a.skillEnabled[idx]
		a.drawToggleSwitch(item.HDC, item.RcItem, on, hovered, selected)
		return
	}

	fill := uiPanelAlt
	border := uiBorderStrong
	textColor := uiText
	if a.isPrimaryButton(id) {
		fill = uiAccent
		border = uiAccent
		textColor = rgb(255, 255, 255)
	}
	if a.isBindingButton(id) {
		fill = rgb(255, 255, 255)
		border = uiBorderStrong
		if text == "" || text == "미지정" || text == "Unassigned" {
			textColor = uiTextSubtle
		}
	}
	if capturing {
		fill = uiAccentSoft
		border = uiAccent
		textColor = uiAccentPressed
	}
	if hovered && !capturing && !disabled {
		if a.isPrimaryButton(id) {
			fill = uiAccentHover
		} else {
			fill = rgb(246, 246, 246)
		}
	}
	if selected {
		if a.isPrimaryButton(id) {
			fill = uiAccentPressed
		} else {
			fill = rgb(238, 238, 238)
		}
	}
	if disabled {
		fill = rgb(246, 246, 246)
		border = rgb(230, 230, 230)
		textColor = rgb(150, 150, 150)
	}
	if text == "" {
		text = "미지정"
	}

	baseBrush := a.gdi.panelBrush
	if id == idLoad || id == idSave {
		baseBrush = a.gdi.bgBrush
	}
	procFillRect.Call(item.HDC, uintptr(unsafe.Pointer(&item.RcItem)), baseBrush)
	a.fillRoundedButton(item.HDC, item.RcItem, fill, border, focused || capturing)
	drawTextInRect(item.HDC, text, a.gdi.font, textColor, item.RcItem, dtCenter|dtVCenter|dtSingleLine|dtEndEllipsis|dtNoPrefix)
}

func (a *application) fillRoundedButton(hdc uintptr, rc rect, fill uintptr, border uintptr, strongBorder bool) {
	brush := createBrush(fill)
	borderWidth := 1
	if strongBorder {
		borderWidth = 2
	}
	pen := createPen(border, borderWidth)
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	height := int(rc.Bottom - rc.Top)
	corner := maxInt(6, height/3)
	procRoundRect.Call(
		hdc,
		uintptr(rc.Left),
		uintptr(rc.Top),
		uintptr(rc.Right),
		uintptr(rc.Bottom),
		uintptr(corner),
		uintptr(corner),
	)
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	deleteGDIObject(pen)
	deleteGDIObject(brush)
}

// drawToggleSwitch renders a pill-shaped toggle switch.
// The track fills the full control rect. The knob sits on the left (OFF) or
// right (ON) side of the track with a small margin.
func (a *application) drawToggleSwitch(hdc uintptr, rc rect, on bool, hovered bool, pressed bool) {
	// Clear background with panel colour so the control blends in.
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), a.gdi.panelBrush)

	// Track colours
	var trackColor uintptr
	if on {
		trackColor = uiAccent
		if hovered {
			trackColor = uiAccentHover
		}
		if pressed {
			trackColor = uiAccentPressed
		}
	} else {
		trackColor = uiBorderStrong
		if hovered {
			trackColor = rgb(170, 170, 170)
		}
		if pressed {
			trackColor = rgb(150, 150, 150)
		}
	}

	// Draw the pill track (fully rounded ends).
	trackH := rc.Bottom - rc.Top
	trackW := rc.Right - rc.Left
	corner := trackH // corner radius = full height → pill shape
	trackBrush := createBrush(trackColor)
	trackPen := createPen(trackColor, 1)
	oldBrush, _, _ := procSelectObject.Call(hdc, trackBrush)
	oldPen, _, _ := procSelectObject.Call(hdc, trackPen)
	procRoundRect.Call(
		hdc,
		uintptr(rc.Left), uintptr(rc.Top),
		uintptr(rc.Right), uintptr(rc.Bottom),
		uintptr(corner), uintptr(corner),
	)
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	deleteGDIObject(trackPen)
	deleteGDIObject(trackBrush)

	// Draw the knob (white circle) inside the track.
	margin := maxInt32(2, trackH/8)
	knobSize := trackH - 2*margin // diameter
	var knobLeft int32
	if on {
		knobLeft = rc.Left + trackW - margin - knobSize
	} else {
		knobLeft = rc.Left + margin
	}
	knobTop := rc.Top + margin
	knobRight := knobLeft + knobSize
	knobBottom := knobTop + knobSize

	knobColor := rgb(255, 255, 255)
	knobBrush := createBrush(knobColor)
	knobPen := createPen(knobColor, 1)
	oldBrush, _, _ = procSelectObject.Call(hdc, knobBrush)
	oldPen, _, _ = procSelectObject.Call(hdc, knobPen)
	procEllipse.Call(
		hdc,
		uintptr(knobLeft), uintptr(knobTop),
		uintptr(knobRight), uintptr(knobBottom),
	)
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	deleteGDIObject(knobPen)
	deleteGDIObject(knobBrush)
}

func drawTextInRect(hdc uintptr, text string, font uintptr, color uintptr, rc rect, flags uintptr) {
	rc = buttonTextContentRect(rc)
	if rc.Right <= rc.Left || rc.Bottom <= rc.Top {
		return
	}
	drawText(hdc, text, font, color, int(rc.Left), int(rc.Top), int(rc.Right-rc.Left), int(rc.Bottom-rc.Top), flags)
}

func buttonTextContentRect(rc rect) rect {
	width := int(rc.Right - rc.Left)
	height := int(rc.Bottom - rc.Top)
	pad := buttonTextHorizontalPadding(width, height)
	rc.Left += int32(pad)
	rc.Right -= int32(pad)
	return rc
}

func buttonTextHorizontalPadding(width, height int) int {
	if width <= 0 || height <= 0 {
		return 0
	}
	pad := maxInt(4, minInt(width/12, height/8))
	if pad*2 >= width {
		return maxInt(0, (width-1)/2)
	}
	return pad
}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
