//go:build windows

package app

import (
	"fmt"
	"strconv"
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

var (
	menuControls = []menuControl{
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

	a.createButton(hwnd, idLoad, "불러오기", 680, 26, 120, 34)
	a.createButton(hwnd, idSave, "저장하기", 812, 26, 120, 34)

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
	a.controls.bulkInterval = a.createEdit(hwnd, idBulkInterval, strconv.Itoa(config.DefaultIntervalMS), 686, 130, 62, 22)
	a.createStatic(hwnd, "ms", 766, 129, 30, 24)
	a.createButton(hwnd, idApplyBulk, "적용", 812, 124, 92, 32)

	a.createStatic(hwnd, "사용", 430, 174, 45, 24)
	a.createStatic(hwnd, "기술", 492, 174, 55, 24)
	a.createStatic(hwnd, "키", 592, 174, 35, 24)
	a.createStatic(hwnd, "실행 간격", 730, 174, 80, 24)
	y := 204
	for i := 0; i < config.MaxSkills; i++ {
		a.controls.skillEnabled[i] = a.createCheckbox(hwnd, idSkillEnabledBase+i, "", 438, y+6, 22, 22)
		a.createStatic(hwnd, strconv.Itoa(i+1), 501, y+7, 30, 22)
		a.controls.skillButtons[i] = a.createButton(hwnd, idSkillKeyBase+i, "", 548, y, 160, 34)
		a.controls.skillInterval[i] = a.createEdit(hwnd, idSkillIntervalBase+i, "", 740, y+7, 56, 22)
		a.createStatic(hwnd, "ms", 822, y+6, 32, 22)
		y += 39
	}

	a.createStatic(hwnd, "키", 426, 648, 45, 24)
	a.controls.pauseButton = a.createButton(hwnd, idPauseKey, "", 548, 642, 258, 34)

	a.createStatic(hwnd, "상태", 48, 711, 55, 24)
	a.controls.status = a.createStatic(hwnd, "정지.", 112, 711, 780, 24)
	a.updateControlsFromConfig()
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
	hwnd := a.createControl(parent, "EDIT", text, wsChild|wsVisible|wsTabStop|esNumber|esAutoHScroll, x, y, width, height, id)
	sendMessage(hwnd, emSetMargins, ecLeftMargin|ecRightMargin, makeLong(0, 0))
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
	path, ok, err := chooseConfigSavePath(a.hwnd, a.configPath)
	if err != nil {
		messageBox(a.hwnd, "파일 선택 실패", err.Error(), mbOK|mbIconError)
		return
	}
	if !ok {
		a.setStatus("저장을 취소했습니다.")
		return
	}
	if err := config.SaveFile(path, a.cfg); err != nil {
		messageBox(a.hwnd, "저장 실패", err.Error(), mbOK|mbIconError)
		return
	}
	a.configPath = path
	a.setStatus("저장 완료: " + a.configPath)
}

func (a *application) loadConfig() {
	path, ok, err := chooseConfigOpenPath(a.hwnd, a.configPath)
	if err != nil {
		messageBox(a.hwnd, "파일 선택 실패", err.Error(), mbOK|mbIconError)
		return
	}
	if !ok {
		a.setStatus("불러오기를 취소했습니다.")
		return
	}
	loaded, err := config.LoadFile(path)
	if err != nil {
		messageBox(a.hwnd, "설정 파일 경고", "올바른 diablo-helper 설정 파일이 아닙니다.\n\n"+err.Error(), mbOK|mbIconWarning)
		return
	}
	a.runner.Stop()
	a.cfg = loaded
	a.configPath = path
	previous := a.capture
	a.capture = captureTarget{}
	a.updateControlsFromConfig()
	a.invalidateCaptureControls(previous)
	a.setStatus("불러오기 완료: " + a.configPath)
}

func (a *application) startRunnerFromHotkey() {
	if a.runner.Running() {
		a.updateRuntimeStatus()
		return
	}
	if err := a.syncConfigFromControls(); err != nil {
		messageBox(a.hwnd, "잘못된 설정", err.Error(), mbOK|mbIconError)
		return
	}
	if len(runnableSkills(a.cfg)) == 0 {
		a.setStatus("실행할 기술이 없습니다. 기술 사용을 켜고 키를 지정하세요.")
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
	if a.hwnd != 0 {
		invalidateRect(a.hwnd, false)
	}
}
