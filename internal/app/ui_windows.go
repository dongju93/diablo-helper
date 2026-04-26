//go:build windows

package app

import (
	"unsafe"

	"github.com/dongju93/diablo-helper/internal/config"
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
)

func rgb(red byte, green byte, blue byte) uintptr {
	return uintptr(uint32(red) | uint32(green)<<8 | uint32(blue)<<16)
}

func int32Arg(value int32) uintptr {
	return uintptr(uint32(value))
}

func (a *application) initUIResources() {
	if a.font == 0 {
		a.font = createUIFont("Malgun Gothic", -15, fwNormal)
	}
	if a.titleFont == 0 {
		a.titleFont = createUIFont("Segoe UI Variable Display", -28, fwSemiBold)
	}
	if a.sectionFont == 0 {
		a.sectionFont = createUIFont("Malgun Gothic", -16, fwSemiBold)
	}
	if a.bgBrush == 0 {
		a.bgBrush = createBrush(uiBackground)
	}
	if a.panelBrush == 0 {
		a.panelBrush = createBrush(uiPanel)
	}
	if a.editBrush == 0 {
		a.editBrush = createBrush(uiPanel)
	}
	if a.borderPen == 0 {
		a.borderPen = createPen(uiBorder, 1)
	}
	if a.borderStrongPen == 0 {
		a.borderStrongPen = createPen(uiBorderStrong, 1)
	}
	if a.borderBrush == 0 {
		a.borderBrush = createBrush(uiBorder)
	}
	if a.accentBrush == 0 {
		a.accentBrush = createBrush(uiAccent)
	}
	if a.accentPen == 0 {
		a.accentPen = createPen(uiAccent, 1)
	}
}

func createUIFont(face string, height int32, weight int) uintptr {
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
	brush, _, _ := procCreateSolidBrush.Call(color)
	return brush
}

func createPen(color uintptr, width int) uintptr {
	pen, _, _ := procCreatePen.Call(psSolid, uintptr(width), color)
	return pen
}

func deleteGDIObject(handle uintptr) {
	if handle != 0 {
		procDeleteObject.Call(handle)
	}
}

func (a *application) disposeUIResources() {
	deleteGDIObject(a.font)
	deleteGDIObject(a.titleFont)
	deleteGDIObject(a.sectionFont)
	deleteGDIObject(a.bgBrush)
	deleteGDIObject(a.panelBrush)
	deleteGDIObject(a.editBrush)
	deleteGDIObject(a.borderPen)
	deleteGDIObject(a.borderStrongPen)
	deleteGDIObject(a.borderBrush)
	deleteGDIObject(a.accentBrush)
	deleteGDIObject(a.accentPen)
	a.font = 0
	a.titleFont = 0
	a.sectionFont = 0
	a.bgBrush = 0
	a.panelBrush = 0
	a.editBrush = 0
	a.borderPen = 0
	a.borderStrongPen = 0
	a.borderBrush = 0
	a.accentBrush = 0
	a.accentPen = 0
}

func (a *application) paint(hwnd uintptr) {
	a.initUIResources()

	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var client rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&client)), a.bgBrush)

	a.drawPanel(hdc, 24, 92, 348, 126)
	a.drawPanel(hdc, 24, 234, 348, 386)
	a.drawPanel(hdc, 396, 92, 540, 498)
	a.drawPanel(hdc, 396, 606, 540, 84)
	a.drawPanel(hdc, 24, 700, 912, 40)
	a.drawAccentMark(hdc, 28, 26, 4, 24)

	a.drawDivider(hdc, 44, 174, 308)
	for y := 320; y <= 560; y += 40 {
		a.drawDivider(hdc, 44, y, 308)
	}
	a.drawDivider(hdc, 416, 198, 500)
	for y := 242; y <= 515; y += 39 {
		a.drawDivider(hdc, 416, y, 500)
	}

	a.drawInputFrame(hdc, 678, 124, 86, 32)
	for y := 204; y < 204+config.MaxSkills*39; y += 39 {
		a.drawInputFrame(hdc, 732, y+1, 82, 32)
	}
	a.drawStatusDot(hdc, 92, 719)

	drawText(hdc, "Diablo Helper", a.titleFont, uiText, 40, 18, 300, 40, dtSingleLine|dtNoPrefix)
	drawText(hdc, "시작/종료 키", a.sectionFont, uiText, 44, 108, 210, 28, dtSingleLine|dtNoPrefix)
	drawText(hdc, "게임 메뉴 키", a.sectionFont, uiText, 44, 250, 210, 28, dtSingleLine|dtNoPrefix)
	drawText(hdc, "기술 키", a.sectionFont, uiText, 416, 108, 160, 28, dtSingleLine|dtNoPrefix)
	drawText(hdc, "일시정지 키", a.sectionFont, uiText, 416, 622, 180, 28, dtSingleLine|dtNoPrefix)
}

func (a *application) drawPanel(hdc uintptr, x int, y int, width int, height int) {
	oldBrush, _, _ := procSelectObject.Call(hdc, a.panelBrush)
	oldPen, _, _ := procSelectObject.Call(hdc, a.borderPen)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height), 16, 16)
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
}

func (a *application) drawInputFrame(hdc uintptr, x int, y int, width int, height int) {
	oldBrush, _, _ := procSelectObject.Call(hdc, a.panelBrush)
	oldPen, _, _ := procSelectObject.Call(hdc, a.borderStrongPen)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height), 8, 8)
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
}

func (a *application) drawDivider(hdc uintptr, x int, y int, width int) {
	rc := rect{Left: int32(x), Top: int32(y), Right: int32(x + width), Bottom: int32(y + 1)}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), a.borderBrush)
}

func (a *application) drawAccentMark(hdc uintptr, x int, y int, width int, height int) {
	oldBrush, _, _ := procSelectObject.Call(hdc, a.accentBrush)
	oldPen, _, _ := procSelectObject.Call(hdc, a.accentPen)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height), 4, 4)
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
}

func (a *application) drawStatusDot(hdc uintptr, x int, y int) {
	color := uiTextSubtle
	switch {
	case a.capture.valid():
		color = uiAccent
	case a.runner.Paused():
		color = uiWarning
	case a.runner.Running():
		color = uiSuccess
	}
	brush := createBrush(color)
	pen := createPen(color, 1)
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procEllipse.Call(hdc, uintptr(x), uintptr(y), uintptr(x+10), uintptr(y+10))
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

func (a *application) colorStatic(hdc uintptr) uintptr {
	a.initUIResources()
	procSetBkMode.Call(hdc, transparent)
	procSetTextColor.Call(hdc, uiText)
	return a.panelBrush
}

func (a *application) colorEdit(hdc uintptr) uintptr {
	a.initUIResources()
	procSetBkColor.Call(hdc, uiPanel)
	procSetTextColor.Call(hdc, uiText)
	return a.editBrush
}

func (a *application) drawButton(item *drawItemStruct) {
	if item == nil || item.HDC == 0 {
		return
	}
	a.initUIResources()

	text := getWindowText(item.HwndItem)
	id := int(item.CtlID)
	selected := item.ItemState&odsSelected != 0
	disabled := item.ItemState&odsDisabled != 0
	focused := item.ItemState&odsFocus != 0
	hovered := item.ItemState&odsHotLight != 0
	capturing := a.captureControlID(a.capture) == id

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

	baseBrush := a.panelBrush
	if id == idLoad || id == idSave {
		baseBrush = a.bgBrush
	}
	procFillRect.Call(item.HDC, uintptr(unsafe.Pointer(&item.RcItem)), baseBrush)
	a.fillRoundedButton(item.HDC, item.RcItem, fill, border, focused || capturing)
	drawTextInRect(item.HDC, text, a.font, textColor, item.RcItem, dtCenter|dtVCenter|dtSingleLine|dtEndEllipsis|dtNoPrefix)
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
	procRoundRect.Call(
		hdc,
		uintptr(rc.Left),
		uintptr(rc.Top),
		uintptr(rc.Right),
		uintptr(rc.Bottom),
		10,
		10,
	)
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	deleteGDIObject(pen)
	deleteGDIObject(brush)
}

func drawTextInRect(hdc uintptr, text string, font uintptr, color uintptr, rc rect, flags uintptr) {
	rc.Left += 10
	rc.Right -= 10
	drawText(hdc, text, font, color, int(rc.Left), int(rc.Top), int(rc.Right-rc.Left), int(rc.Bottom-rc.Top), flags)
}
