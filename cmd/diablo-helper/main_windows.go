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
	hwnd       uintptr
	instance   uintptr
	hook       uintptr
	mouseHook  uintptr
	font       uintptr
	configPath string
	cfg        config.Config
	controls   controlRefs
	capture    captureTarget
	pressed    map[uint16]bool
	runner     *skillRunner
}

var (
	appInstance      *application
	windowProc       = syscall.NewCallback(wndProc)
	keyboardHookProc = syscall.NewCallback(lowLevelKeyboardProc)
	mouseHookProc    = syscall.NewCallback(lowLevelMouseProc)
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
		900,
		720,
		0,
		0,
		a.instance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", err)
	}
	a.hwnd = hwnd

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

func (a *application) createControls(hwnd uintptr) {
	font, _, _ := procGetStockObject.Call(defaultGUIFont)
	a.font = font

	a.createButton(hwnd, idLoad, "불러오기", 20, 18, 90, 30)
	a.createButton(hwnd, idSave, "저장하기", 118, 18, 90, 30)

	a.createGroupBox(hwnd, "시작/종료 키", 20, 65, 360, 120)
	a.createStatic(hwnd, "시작 키", 55, 105, 95, 24)
	a.controls.startButton = a.createButton(hwnd, idStartKey, "", 165, 100, 180, 30)
	a.createStatic(hwnd, "종료 키", 55, 145, 95, 24)
	a.controls.stopButton = a.createButton(hwnd, idStopKey, "", 165, 140, 180, 30)

	a.createGroupBox(hwnd, "게임 메뉴 키 (입력 시 기술 반복 중지)", 20, 205, 360, 390)
	menuY := 245
	for _, menu := range menuControls {
		a.createStatic(hwnd, menu.label, 55, menuY+4, 120, 24)
		a.controls.menuButtons[menu.id] = a.createButton(hwnd, menu.control, "", 190, menuY, 155, 30)
		menuY += 43
	}

	a.createGroupBox(hwnd, "기술 키", 405, 65, 455, 430)
	a.createStatic(hwnd, "일괄 간격", 620, 100, 75, 24)
	a.controls.bulkInterval = a.createEdit(hwnd, idBulkInterval, strconv.Itoa(config.DefaultIntervalMS), 700, 96, 70, 30)
	a.createStatic(hwnd, "msec", 778, 100, 40, 24)
	a.createButton(hwnd, idApplyBulk, "적용", 815, 96, 42, 30)

	a.createStatic(hwnd, "사용", 430, 142, 45, 24)
	a.createStatic(hwnd, "기술", 490, 142, 55, 24)
	a.createStatic(hwnd, "키", 590, 142, 35, 24)
	a.createStatic(hwnd, "실행 간격", 720, 142, 80, 24)
	y := 172
	for i := 0; i < config.MaxSkills; i++ {
		a.controls.skillEnabled[i] = a.createCheckbox(hwnd, idSkillEnabledBase+i, "", 438, y+4, 22, 22)
		a.createStatic(hwnd, strconv.Itoa(i+1), 500, y+4, 30, 22)
		a.controls.skillButtons[i] = a.createButton(hwnd, idSkillKeyBase+i, "", 560, y, 150, 30)
		a.controls.skillInterval[i] = a.createEdit(hwnd, idSkillIntervalBase+i, "", 725, y, 70, 30)
		a.createStatic(hwnd, "msec", 803, y+4, 42, 22)
		y += 36
	}

	a.createGroupBox(hwnd, "일시정지 키 (누르고 있는 동안 기술 반복 중지)", 405, 515, 455, 80)
	a.createStatic(hwnd, "키", 455, 550, 45, 24)
	a.controls.pauseButton = a.createButton(hwnd, idPauseKey, "", 560, 545, 250, 30)

	a.createStatic(hwnd, "상태", 20, 650, 55, 24)
	a.controls.status = a.createStatic(hwnd, "Stopped", 80, 650, 780, 24)
	a.updateControlsFromConfig()
}

func (a *application) createGroupBox(parent uintptr, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "BUTTON", text, wsChild|wsVisible|bsGroupBox, x, y, width, height, 0)
}

func (a *application) createStatic(parent uintptr, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "STATIC", text, wsChild|wsVisible|ssLeft, x, y, width, height, 0)
}

func (a *application) createButton(parent uintptr, id int, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "BUTTON", text, wsChild|wsVisible|wsTabStop|bsPushButton, x, y, width, height, id)
}

func (a *application) createCheckbox(parent uintptr, id int, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "BUTTON", text, wsChild|wsVisible|wsTabStop|bsAutoCheckbox, x, y, width, height, id)
}

func (a *application) createEdit(parent uintptr, id int, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "EDIT", text, wsChild|wsVisible|wsTabStop|wsBorder|esNumber, x, y, width, height, id)
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
	a.capture = target
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
			a.clearCapturedKey()
			a.capture = captureTarget{}
			a.updateControlsFromConfig()
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
	binding := config.KeyBinding{Name: keyDisplayName(vk), VK: int(vk)}
	switch a.capture.kind {
	case captureStart:
		a.cfg.Start = binding
	case captureStop:
		a.cfg.Stop = binding
	case capturePause:
		a.cfg.Pause = binding
	case captureSkill:
		if a.capture.index >= 0 && a.capture.index < len(a.cfg.Skills) {
			a.cfg.Skills[a.capture.index].Key = binding
		}
	case captureMenu:
		a.setMenuBinding(a.capture.menuID, binding)
	}
	a.capture = captureTarget{}
	a.updateControlsFromConfig()
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
	a.capture = captureTarget{}
	a.updateControlsFromConfig()
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
		return "Unassigned"
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
