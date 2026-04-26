//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/dongju93/diablo-helper/internal/config"
)

const (
	idStartKey = 100
	idStopKey  = 101
	idPauseKey = 102

	idBulkInterval = 110
	idApplyBulk    = 111

	idMenuInventory  = 120
	idMenuSkills     = 121
	idMenuFollower   = 122
	idMenuMap        = 123
	idMenuTownPortal = 124
	idMenuChat       = 125
	idMenuWorldMap   = 126
	idMenuWhisper    = 127

	idSkillEnabledBase  = 200
	idSkillKeyBase      = 300
	idSkillIntervalBase = 400

	idSave = 500
	idLoad = 501
)

type captureKind int

const (
	captureNone captureKind = iota
	captureStart
	captureStop
	capturePause
	captureMenu
	captureSkill
)

type captureTarget struct {
	kind   captureKind
	index  int
	menuID string
}

func (t captureTarget) valid() bool {
	return t.kind != captureNone
}

type controlRefs struct {
	startButton uintptr
	stopButton  uintptr
	pauseButton uintptr

	menuButtons map[string]uintptr

	skillEnabled  [config.MaxSkills]uintptr
	skillButtons  [config.MaxSkills]uintptr
	skillInterval [config.MaxSkills]uintptr

	bulkInterval uintptr
	status       uintptr
}

type menuControl struct {
	id      string
	label   string
	control int
}

type application struct {
	hwnd        uintptr
	instance    uintptr
	hook        uintptr
	mouseHook   uintptr
	font        uintptr
	titleFont   uintptr
	sectionFont uintptr
	bgBrush     uintptr
	panelBrush  uintptr
	editBrush   uintptr
	configPath  string
	cfg         config.Config
	controls    controlRefs
	capture     captureTarget
	pressed     map[uint16]bool
	runner      *skillRunner
}

var (
	appInstance      *application
	windowProc       = syscall.NewCallback(wndProc)
	keyboardHookProc = syscall.NewCallback(lowLevelKeyboardProc)
	mouseHookProc    = syscall.NewCallback(lowLevelMouseProc)
	uiBackground     = rgb(243, 243, 243)
	uiPanel          = rgb(255, 255, 255)
	uiPanelAlt       = rgb(250, 250, 250)
	uiBorder         = rgb(221, 221, 221)
	uiBorderStrong   = rgb(198, 198, 198)
	uiText           = rgb(32, 32, 32)
	uiTextSubtle     = rgb(96, 96, 96)
	uiAccent         = rgb(0, 103, 192)
	uiAccentPressed  = rgb(0, 90, 158)
	uiAccentSoft     = rgb(232, 241, 252)
	menuControls     = []menuControl{
		{id: "inventory", label: "소지품", control: idMenuInventory},
		{id: "skills", label: "기술", control: idMenuSkills},
		{id: "follower", label: "추종자", control: idMenuFollower},
		{id: "map", label: "지도", control: idMenuMap},
		{id: "world_map", label: "세계지도", control: idMenuWorldMap},
		{id: "town_portal", label: "차원문", control: idMenuTownPortal},
		{id: "chat", label: "채팅", control: idMenuChat},
		{id: "whisper", label: "귓말", control: idMenuWhisper},
	}
)

func main() {
	runtime.LockOSThread()

	app := newApplication()
	if err := app.run(); err != nil {
		messageBox(0, "diablo-helper", err.Error(), mbOK|mbIconError)
		os.Exit(1)
	}
}

func newApplication() *application {
	return &application{
		cfg:      config.Default(),
		pressed:  make(map[uint16]bool),
		runner:   newSkillRunner(sendVirtualKey),
		controls: controlRefs{menuButtons: make(map[string]uintptr)},
	}
}

func (a *application) run() error {
	a.configPath = defaultConfigPath()
	if loaded, err := config.LoadFile(a.configPath); err == nil {
		a.cfg = loaded
	} else if !errors.Is(err, os.ErrNotExist) {
		messageBox(0, "diablo-helper", "Failed to load settings.toml. Defaults will be used.\n\n"+err.Error(), mbOK|mbIconError)
	}
	a.cfg.Normalize()

	instance, _, err := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return fmt.Errorf("GetModuleHandleW failed: %w", err)
	}
	a.instance = instance

	if err := a.registerWindowClass(); err != nil {
		return err
	}

	appInstance = a
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("DiabloHelperWindow"))),
		uintptr(unsafe.Pointer(utf16Ptr("Diablo Helper"))),
		wsOverlappedWindow,
		cwUseDefault,
		cwUseDefault,
		980,
		780,
		0,
		0,
		a.instance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", err)
	}
	a.hwnd = hwnd
	setRoundedWindowCorners(hwnd)

	hook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, keyboardHookProc, a.instance, 0)
	if hook == 0 {
		return fmt.Errorf("SetWindowsHookExW failed: %w", err)
	}
	a.hook = hook
	mouseHook, _, err := procSetWindowsHookExW.Call(whMouseLL, mouseHookProc, a.instance, 0)
	if mouseHook == 0 {
		procUnhookWindowsHook.Call(a.hook)
		a.hook = 0
		return fmt.Errorf("SetWindowsHookExW mouse hook failed: %w", err)
	}
	a.mouseHook = mouseHook

	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)

	var msg message
	for {
		ret, _, callErr := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			return fmt.Errorf("GetMessageW failed: %w", callErr)
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func (a *application) registerWindowClass() error {
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	className := utf16Ptr("DiabloHelperWindow")
	wc := windowClassEx{
		Size:       uint32(unsafe.Sizeof(windowClassEx{})),
		WndProc:    windowProc,
		Instance:   a.instance,
		Cursor:     cursor,
		Background: colorWindow + 1,
		ClassName:  className,
	}
	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", err)
	}
	return nil
}

func wndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	if appInstance == nil {
		return defWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case wmCreate:
		appInstance.hwnd = hwnd
		appInstance.createControls(hwnd)
		return 0
	case wmPaint:
		appInstance.paint(hwnd)
		return 0
	case wmEraseBkgnd:
		return 1
	case wmCtlColorStatic:
		return appInstance.colorStatic(wParam)
	case wmCtlColorBtn:
		return appInstance.colorStatic(wParam)
	case wmCtlColorEdit:
		return appInstance.colorEdit(wParam)
	case wmDrawItem:
		appInstance.drawButton((*drawItemStruct)(unsafe.Pointer(lParam)))
		return 1
	case wmCommand:
		if appInstance.handleCommand(wParam) {
			return 0
		}
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		if appInstance.hook != 0 {
			procUnhookWindowsHook.Call(appInstance.hook)
			appInstance.hook = 0
		}
		if appInstance.mouseHook != 0 {
			procUnhookWindowsHook.Call(appInstance.mouseHook)
			appInstance.mouseHook = 0
		}
		appInstance.runner.Stop()
		appInstance.disposeUIResources()
		procPostQuitMessage.Call(0)
		return 0
	}
	return defWindowProc(hwnd, msg, wParam, lParam)
}

func lowLevelKeyboardProc(code int, wParam uintptr, lParam uintptr) uintptr {
	if code < 0 || appInstance == nil {
		return callNextKeyboardHook(code, wParam, lParam)
	}
	event := (*keyboardHookStruct)(unsafe.Pointer(lParam))
	if event.Flags&llkhfInjected != 0 {
		return callNextKeyboardHook(code, wParam, lParam)
	}

	switch wParam {
	case wmKeyDown, wmSysKeyDown:
		if appInstance.handleKeyEvent(uint16(event.VKCode), true) {
			return 1
		}
	case wmKeyUp, wmSysKeyUp:
		if appInstance.handleKeyEvent(uint16(event.VKCode), false) {
			return 1
		}
	}
	return callNextKeyboardHook(code, wParam, lParam)
}

func lowLevelMouseProc(code int, wParam uintptr, lParam uintptr) uintptr {
	if code < 0 || appInstance == nil {
		return callNextKeyboardHook(code, wParam, lParam)
	}
	event := (*mouseHookStruct)(unsafe.Pointer(lParam))
	if event.Flags&llmhfInjected != 0 {
		return callNextKeyboardHook(code, wParam, lParam)
	}

	vk, down, ok := mouseEventKey(wParam, event)
	if !ok {
		return callNextKeyboardHook(code, wParam, lParam)
	}
	if appInstance.handleKeyEvent(vk, down) {
		return 1
	}
	return callNextKeyboardHook(code, wParam, lParam)
}

func mouseEventKey(wParam uintptr, event *mouseHookStruct) (uint16, bool, bool) {
	switch wParam {
	case wmLButtonDown:
		return vkLButton, true, true
	case wmLButtonUp:
		return vkLButton, false, true
	case wmRButtonDown:
		return vkRButton, true, true
	case wmRButtonUp:
		return vkRButton, false, true
	case wmMButtonDown:
		return vkMButton, true, true
	case wmMButtonUp:
		return vkMButton, false, true
	case wmXButtonDown, wmXButtonUp:
		button := event.MouseData >> 16
		switch button {
		case xButton1:
			return vkXButton1, wParam == wmXButtonDown, true
		case xButton2:
			return vkXButton2, wParam == wmXButtonDown, true
		}
	}
	return 0, false, false
}

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
		a.titleFont = createUIFont("Segoe UI", -26, fwSemiBold)
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
	a.font = 0
	a.titleFont = 0
	a.sectionFont = 0
	a.bgBrush = 0
	a.panelBrush = 0
	a.editBrush = 0
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

	drawText(hdc, "Diablo Helper", a.titleFont, uiText, 28, 20, 300, 36, dtSingleLine|dtNoPrefix)
	drawText(hdc, "시작/종료 키", a.sectionFont, uiText, 44, 108, 210, 28, dtSingleLine|dtNoPrefix)
	drawText(hdc, "게임 메뉴 키", a.sectionFont, uiText, 44, 250, 210, 28, dtSingleLine|dtNoPrefix)
	drawText(hdc, "기술 키", a.sectionFont, uiText, 416, 108, 160, 28, dtSingleLine|dtNoPrefix)
	drawText(hdc, "일시정지 키", a.sectionFont, uiText, 416, 622, 180, 28, dtSingleLine|dtNoPrefix)
}

func (a *application) drawPanel(hdc uintptr, x int, y int, width int, height int) {
	brush := createBrush(uiPanel)
	pen := createPen(uiBorder, 1)
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procRoundRect.Call(
		hdc,
		uintptr(x),
		uintptr(y),
		uintptr(x+width),
		uintptr(y+height),
		16,
		16,
	)
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

func (a *application) isPrimaryButton(id int) bool {
	return id == idSave || id == idApplyBulk
}

func (a *application) isBindingButton(id int) bool {
	if id == idStartKey || id == idStopKey || id == idPauseKey {
		return true
	}
	if id >= idSkillKeyBase && id < idSkillKeyBase+config.MaxSkills {
		return true
	}
	for _, menu := range menuControls {
		if id == menu.control {
			return true
		}
	}
	return false
}

func (a *application) captureControlID(target captureTarget) int {
	switch target.kind {
	case captureStart:
		return idStartKey
	case captureStop:
		return idStopKey
	case capturePause:
		return idPauseKey
	case captureSkill:
		if target.index >= 0 && target.index < config.MaxSkills {
			return idSkillKeyBase + target.index
		}
	case captureMenu:
		for _, menu := range menuControls {
			if menu.id == target.menuID {
				return menu.control
			}
		}
	}
	return 0
}

func (a *application) invalidateCaptureControls(targets ...captureTarget) {
	for _, target := range targets {
		id := a.captureControlID(target)
		if id == 0 || a.hwnd == 0 {
			continue
		}
		if hwnd := getDlgItem(a.hwnd, id); hwnd != 0 {
			invalidateRect(hwnd, true)
		}
	}
}

func (a *application) createControls(hwnd uintptr) {
	a.initUIResources()

	a.createButton(hwnd, idLoad, "불러오기", 740, 26, 88, 34)
	a.createButton(hwnd, idSave, "저장하기", 840, 26, 88, 34)

	a.createStatic(hwnd, "시작 키", 48, 139, 95, 24)
	a.controls.startButton = a.createButton(hwnd, idStartKey, "", 154, 134, 190, 34)
	a.createStatic(hwnd, "종료 키", 48, 181, 95, 24)
	a.controls.stopButton = a.createButton(hwnd, idStopKey, "", 154, 176, 190, 34)

	menuY := 282
	for _, menu := range menuControls {
		a.createStatic(hwnd, menu.label, 48, menuY+5, 120, 24)
		a.controls.menuButtons[menu.id] = a.createButton(hwnd, menu.control, "", 174, menuY, 170, 34)
		menuY += 40
	}

	a.createStatic(hwnd, "일괄 간격", 598, 129, 78, 24)
	a.controls.bulkInterval = a.createEdit(hwnd, idBulkInterval, strconv.Itoa(config.DefaultIntervalMS), 678, 124, 78, 32)
	a.createStatic(hwnd, "ms", 766, 129, 30, 24)
	a.createButton(hwnd, idApplyBulk, "적용", 812, 124, 76, 32)

	a.createStatic(hwnd, "사용", 430, 174, 45, 24)
	a.createStatic(hwnd, "기술", 492, 174, 55, 24)
	a.createStatic(hwnd, "키", 592, 174, 35, 24)
	a.createStatic(hwnd, "실행 간격", 730, 174, 80, 24)
	y := 204
	for i := 0; i < config.MaxSkills; i++ {
		a.controls.skillEnabled[i] = a.createCheckbox(hwnd, idSkillEnabledBase+i, "", 438, y+6, 22, 22)
		a.createStatic(hwnd, strconv.Itoa(i+1), 501, y+7, 30, 22)
		a.controls.skillButtons[i] = a.createButton(hwnd, idSkillKeyBase+i, "", 548, y, 160, 34)
		a.controls.skillInterval[i] = a.createEdit(hwnd, idSkillIntervalBase+i, "", 732, y+1, 78, 32)
		a.createStatic(hwnd, "ms", 822, y+6, 32, 22)
		y += 39
	}

	a.createStatic(hwnd, "키", 426, 648, 45, 24)
	a.controls.pauseButton = a.createButton(hwnd, idPauseKey, "", 548, 642, 258, 34)

	a.createStatic(hwnd, "상태", 48, 711, 55, 24)
	a.controls.status = a.createStatic(hwnd, "정지.", 112, 711, 780, 24)
	a.updateControlsFromConfig()
}

func (a *application) createGroupBox(parent uintptr, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "BUTTON", text, wsChild|wsVisible|bsGroupBox, x, y, width, height, 0)
}

func (a *application) createStatic(parent uintptr, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "STATIC", text, wsChild|wsVisible|ssLeft, x, y, width, height, 0)
}

func (a *application) createButton(parent uintptr, id int, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "BUTTON", text, wsChild|wsVisible|wsTabStop|bsOwnerDraw, x, y, width, height, id)
}

func (a *application) createCheckbox(parent uintptr, id int, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "BUTTON", text, wsChild|wsVisible|wsTabStop|bsAutoCheckbox, x, y, width, height, id)
}

func (a *application) createEdit(parent uintptr, id int, text string, x int, y int, width int, height int) uintptr {
	hwnd := a.createControl(parent, "EDIT", text, wsChild|wsVisible|wsTabStop|wsBorder|esNumber, x, y, width, height, id)
	sendMessage(hwnd, emSetMargins, ecLeftMargin|ecRightMargin, makeLong(8, 8))
	return hwnd
}

func (a *application) createControl(parent uintptr, class string, text string, style int, x int, y int, width int, height int, id int) uintptr {
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(class))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		parent,
		uintptr(id),
		a.instance,
		0,
	)
	if hwnd != 0 && a.font != 0 {
		sendMessage(hwnd, wmSetFont, a.font, 1)
		setWindowTheme(hwnd, "Explorer")
	}
	return hwnd
}

func (a *application) handleCommand(wParam uintptr) bool {
	if highWord(wParam) != bnClicked {
		return false
	}

	id := lowWord(wParam)
	switch {
	case id == idStartKey:
		a.startCapture(captureTarget{kind: captureStart})
	case id == idStopKey:
		a.startCapture(captureTarget{kind: captureStop})
	case id == idPauseKey:
		a.startCapture(captureTarget{kind: capturePause})
	case id >= idSkillKeyBase && id < idSkillKeyBase+config.MaxSkills:
		a.startCapture(captureTarget{kind: captureSkill, index: id - idSkillKeyBase})
	case id == idApplyBulk:
		a.applyBulkInterval()
	case id == idSave:
		a.saveConfig()
	case id == idLoad:
		a.loadConfig()
	default:
		for _, menu := range menuControls {
			if id == menu.control {
				a.startCapture(captureTarget{kind: captureMenu, menuID: menu.id})
				return true
			}
		}
		return false
	}
	return true
}

func (a *application) startCapture(target captureTarget) {
	previous := a.capture
	a.capture = target
	a.invalidateCaptureControls(previous, target)
	a.setStatus("할당할 키를 입력하세요. Esc는 해제입니다.")
}

func (a *application) handleKeyEvent(vk uint16, down bool) bool {
	if down {
		if a.pressed[vk] {
			return false
		}
		a.pressed[vk] = true
	} else {
		delete(a.pressed, vk)
		if sameKey(vk, a.cfg.Pause) {
			a.runner.SetPaused(false)
			a.updateRuntimeStatus()
		}
		return false
	}

	if a.capture.valid() {
		if vk == vkEscape {
			target := a.capture
			a.clearCapturedKey()
			a.capture = captureTarget{}
			a.updateBindingControl(target)
			a.invalidateCaptureControls(target)
			a.setStatus("키 할당을 해제했습니다.")
			return true
		}
		if vk == vkLButton && (a.capture.kind == captureStart || a.capture.kind == captureStop) {
			a.setStatus("시작/종료 키에는 Mouse Left를 사용할 수 없습니다.")
			return true
		}
		a.assignCapturedKey(vk)
		return true
	}

	switch {
	case a.runner.Running() && sameKey(vk, a.cfg.Stop):
		a.stopRunner("종료 키 입력으로 정지했습니다.")
	case sameKey(vk, a.cfg.Start):
		a.startRunnerFromHotkey()
	case sameKey(vk, a.cfg.Stop):
		a.stopRunner("종료 키 입력으로 정지했습니다.")
	case a.menuKeyMatches(vk):
		a.stopRunner("게임 메뉴 키 입력으로 정지했습니다.")
	case sameKey(vk, a.cfg.Pause):
		a.runner.SetPaused(true)
		a.updateRuntimeStatus()
	}
	return false
}

func (a *application) assignCapturedKey(vk uint16) {
	target := a.capture
	binding := config.KeyBinding{Name: keyDisplayName(vk), VK: int(vk)}
	switch target.kind {
	case captureStart:
		a.cfg.Start = binding
	case captureStop:
		a.cfg.Stop = binding
	case capturePause:
		a.cfg.Pause = binding
	case captureSkill:
		if target.index >= 0 && target.index < len(a.cfg.Skills) {
			a.cfg.Skills[target.index].Key = binding
		}
	case captureMenu:
		a.setMenuBinding(target.menuID, binding)
	}
	a.capture = captureTarget{}
	a.updateBindingControl(target)
	a.invalidateCaptureControls(target)
	a.setStatus("키 입력 완료.")
}

func (a *application) clearCapturedKey() {
	switch a.capture.kind {
	case captureStart:
		a.cfg.Start = config.KeyBinding{}
	case captureStop:
		a.cfg.Stop = config.KeyBinding{}
	case capturePause:
		a.cfg.Pause = config.KeyBinding{}
	case captureSkill:
		if a.capture.index >= 0 && a.capture.index < len(a.cfg.Skills) {
			a.cfg.Skills[a.capture.index].Key = config.KeyBinding{}
		}
	case captureMenu:
		a.setMenuBinding(a.capture.menuID, config.KeyBinding{})
	}
}

func (a *application) updateBindingControl(target captureTarget) {
	switch target.kind {
	case captureStart:
		setWindowText(a.controls.startButton, bindingText(a.cfg.Start))
	case captureStop:
		setWindowText(a.controls.stopButton, bindingText(a.cfg.Stop))
	case capturePause:
		setWindowText(a.controls.pauseButton, bindingText(a.cfg.Pause))
	case captureSkill:
		if target.index >= 0 && target.index < len(a.cfg.Skills) {
			setWindowText(a.controls.skillButtons[target.index], bindingText(a.cfg.Skills[target.index].Key))
		}
	case captureMenu:
		if hwnd := a.controls.menuButtons[target.menuID]; hwnd != 0 {
			for _, menu := range a.cfg.MenuBindings() {
				if menu.ID == target.menuID {
					setWindowText(hwnd, bindingText(menu.Binding))
					return
				}
			}
		}
	}
}

func (a *application) updateControlsFromConfig() {
	a.cfg.Normalize()
	setWindowText(a.controls.startButton, bindingText(a.cfg.Start))
	setWindowText(a.controls.stopButton, bindingText(a.cfg.Stop))
	setWindowText(a.controls.pauseButton, bindingText(a.cfg.Pause))
	for _, menu := range a.cfg.MenuBindings() {
		if hwnd := a.controls.menuButtons[menu.ID]; hwnd != 0 {
			setWindowText(hwnd, bindingText(menu.Binding))
		}
	}
	for i := 0; i < config.MaxSkills; i++ {
		setChecked(a.controls.skillEnabled[i], a.cfg.Skills[i].Enabled)
		setWindowText(a.controls.skillButtons[i], bindingText(a.cfg.Skills[i].Key))
		setWindowText(a.controls.skillInterval[i], strconv.Itoa(a.cfg.Skills[i].IntervalMS))
	}
	a.updateRuntimeStatus()
}

func (a *application) applyBulkInterval() {
	interval, err := parseInterval(getWindowText(a.controls.bulkInterval))
	if err != nil {
		messageBox(a.hwnd, "잘못된 간격", err.Error(), mbOK|mbIconError)
		return
	}
	for i := 0; i < config.MaxSkills; i++ {
		setWindowText(a.controls.skillInterval[i], strconv.Itoa(interval))
	}
	a.setStatus("일괄 간격을 적용했습니다.")
}

func (a *application) saveConfig() {
	if err := a.syncConfigFromControls(); err != nil {
		messageBox(a.hwnd, "잘못된 설정", err.Error(), mbOK|mbIconError)
		return
	}
	if err := config.SaveFile(a.configPath, a.cfg); err != nil {
		messageBox(a.hwnd, "저장 실패", err.Error(), mbOK|mbIconError)
		return
	}
	a.setStatus("저장 완료: " + a.configPath)
}

func (a *application) loadConfig() {
	loaded, err := config.LoadFile(a.configPath)
	if err != nil {
		messageBox(a.hwnd, "불러오기 실패", err.Error(), mbOK|mbIconError)
		return
	}
	a.runner.Stop()
	a.cfg = loaded
	previous := a.capture
	a.capture = captureTarget{}
	a.updateControlsFromConfig()
	a.invalidateCaptureControls(previous)
	a.setStatus("불러오기 완료: " + a.configPath)
}

func (a *application) startRunnerFromHotkey() {
	if err := a.syncConfigFromControls(); err != nil {
		messageBox(a.hwnd, "잘못된 설정", err.Error(), mbOK|mbIconError)
		return
	}
	if a.runner.Start(a.cfg) {
		a.setStatus("실행 중.")
		return
	}
	a.updateRuntimeStatus()
}

func (a *application) stopRunner(status string) {
	if a.runner.Stop() {
		a.setStatus(status)
		return
	}
	a.updateRuntimeStatus()
}

func (a *application) syncConfigFromControls() error {
	a.cfg.Normalize()
	for i := 0; i < config.MaxSkills; i++ {
		interval, err := parseInterval(getWindowText(a.controls.skillInterval[i]))
		if err != nil {
			return fmt.Errorf("기술 %d: %w", i+1, err)
		}
		a.cfg.Skills[i].IntervalMS = interval
		a.cfg.Skills[i].Enabled = checked(a.controls.skillEnabled[i])
	}
	a.cfg.Normalize()
	return a.cfg.Validate()
}

func (a *application) updateRuntimeStatus() {
	switch {
	case a.runner.Paused():
		a.setStatus("일시정지 키를 누르고 있어 기술 입력을 중지했습니다.")
	case a.runner.Running():
		a.setStatus("실행 중.")
	default:
		a.setStatus("정지.")
	}
}

func (a *application) setStatus(text string) {
	if a.controls.status != 0 {
		setWindowText(a.controls.status, text)
	}
}

func (a *application) menuKeyMatches(vk uint16) bool {
	for _, menu := range a.cfg.MenuBindings() {
		if sameKey(vk, menu.Binding) {
			return true
		}
	}
	return false
}

func (a *application) setMenuBinding(id string, binding config.KeyBinding) {
	switch id {
	case "inventory":
		a.cfg.Menu.Inventory = binding
	case "skills":
		a.cfg.Menu.Skills = binding
	case "follower":
		a.cfg.Menu.Follower = binding
	case "map":
		a.cfg.Menu.Map = binding
	case "world_map":
		a.cfg.Menu.WorldMap = binding
	case "town_portal":
		a.cfg.Menu.TownPortal = binding
	case "chat":
		a.cfg.Menu.Chat = binding
	case "whisper":
		a.cfg.Menu.Whisper = binding
	}
}

func sameKey(vk uint16, binding config.KeyBinding) bool {
	return binding.Assigned() && uint16(binding.VK) == vk
}

func bindingText(binding config.KeyBinding) string {
	if !binding.Assigned() {
		return "미지정"
	}
	if binding.Name != "" {
		return binding.Name
	}
	return keyDisplayName(uint16(binding.VK))
}

func parseInterval(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("실행 간격은 필수입니다")
	}
	interval, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("실행 간격은 숫자여야 합니다")
	}
	if interval < config.MinimumIntervalMS {
		return 0, fmt.Errorf("실행 간격은 최소 %dms 이상이어야 합니다", config.MinimumIntervalMS)
	}
	return interval, nil
}

func defaultConfigPath() string {
	executable, err := os.Executable()
	if err != nil {
		return "settings.toml"
	}
	return filepath.Join(filepath.Dir(executable), "settings.toml")
}
