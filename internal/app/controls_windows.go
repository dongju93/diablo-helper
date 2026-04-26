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
	idBulkSkillGap = 112

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

const (
	bulkIntervalLabelY = 125
	bulkIntervalEditY  = 126
	bulkSkillGapLabelY = 158
	bulkSkillGapEditY  = 159
	bulkApplyY         = 124
	bulkApplyH         = 64
	skillHeaderY       = 204
	skillFirstRowY     = 234
	skillRowGap        = 39
)

type controlRefs struct {
	// Left column key bindings
	startLabel  uintptr
	startButton uintptr
	stopLabel   uintptr
	stopButton  uintptr
	pauseButton uintptr
	menuLabels  map[string]uintptr
	menuButtons map[string]uintptr

	// Right column – header buttons
	loadButton uintptr
	saveButton uintptr

	// Right column – bulk interval section
	bulkLabel       uintptr
	bulkInterval    uintptr
	bulkMsLabel     uintptr
	bulkSkillGapLbl uintptr
	bulkSkillGap    uintptr
	bulkGapMsLabel  uintptr
	applyBulk       uintptr

	// Right column – skill grid headers
	skillUseHdr uintptr
	skillNumHdr uintptr
	skillKeyHdr uintptr
	skillIntHdr uintptr

	// Right column – skill rows
	skillEnabled  [config.MaxSkills]uintptr
	skillNums     [config.MaxSkills]uintptr
	skillButtons  [config.MaxSkills]uintptr
	skillInterval [config.MaxSkills]uintptr
	skillMsLbls   [config.MaxSkills]uintptr

	// Right column – pause section
	pauseLabel uintptr

	// Status bar
	statusLabel uintptr
	status      uintptr
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

	cw, ch := getClientSize(hwnd)
	lo := computeLayout(cw, ch)

	// Header buttons (right-anchored)
	a.controls.loadButton = a.createButton(hwnd, idLoad, "불러오기", lo.loadX, lo.y(26), lo.w(headerBtnW), lo.h(34))
	a.controls.saveButton = a.createButton(hwnd, idSave, "저장하기", lo.saveX, lo.y(26), lo.w(headerBtnW), lo.h(34))

	// Left column – key bindings
	a.controls.startLabel = a.createStatic(hwnd, "시작 키", lo.x(layoutLX+24), lo.y(139), lo.w(95), lo.h(24))
	a.controls.startButton = a.createButton(hwnd, idStartKey, "", lo.x(layoutLX+130), lo.y(134), lo.w(190), lo.h(34))
	a.controls.stopLabel = a.createStatic(hwnd, "종료 키", lo.x(layoutLX+24), lo.y(181), lo.w(95), lo.h(24))
	a.controls.stopButton = a.createButton(hwnd, idStopKey, "", lo.x(layoutLX+130), lo.y(176), lo.w(190), lo.h(34))

	menuY := 282
	for _, menu := range menuControls {
		a.controls.menuLabels[menu.id] = a.createStatic(hwnd, menu.label, lo.x(layoutLX+24), lo.y(menuY+5), lo.w(120), lo.h(24))
		a.controls.menuButtons[menu.id] = a.createButton(hwnd, menu.control, "", lo.x(layoutLX+150), lo.y(menuY), lo.w(170), lo.h(34))
		menuY += 40
	}

	// Right column – bulk interval section
	a.controls.bulkLabel = a.createStatic(hwnd, "일괄 간격", lo.bulkLabelX, lo.y(bulkIntervalLabelY), lo.w(78), lo.h(24))
	a.controls.bulkInterval = a.createEdit(hwnd, idBulkInterval, strconv.Itoa(config.DefaultIntervalMS), lo.bulkEditX, lo.y(bulkIntervalEditY), lo.w(bulkEditW), lo.h(22))
	a.controls.bulkMsLabel = a.createStatic(hwnd, "ms", lo.bulkMsX, lo.y(bulkIntervalLabelY), lo.w(bulkMsW), lo.h(24))
	a.controls.bulkSkillGapLbl = a.createStatic(hwnd, "키별 간격", lo.bulkLabelX, lo.y(bulkSkillGapLabelY), lo.w(78), lo.h(24))
	a.controls.bulkSkillGap = a.createEdit(hwnd, idBulkSkillGap, strconv.Itoa(config.DefaultSkillGapMS), lo.bulkEditX, lo.y(bulkSkillGapEditY), lo.w(bulkEditW), lo.h(22))
	a.controls.bulkGapMsLabel = a.createStatic(hwnd, "ms", lo.bulkMsX, lo.y(bulkSkillGapLabelY), lo.w(bulkMsW), lo.h(24))
	a.controls.applyBulk = a.createButton(hwnd, idApplyBulk, "일괄 적용", lo.bulkApplyX, lo.y(bulkApplyY), lo.w(bulkApplyW), lo.h(bulkApplyH))

	// Right column – skill grid headers
	a.controls.skillUseHdr = a.createStatic(hwnd, "사용", lo.skillUseHdrX, lo.y(skillHeaderY), lo.w(45), lo.h(24))
	a.controls.skillNumHdr = a.createStatic(hwnd, "기술", lo.skillNumHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24))
	a.controls.skillKeyHdr = a.createStatic(hwnd, "키", lo.skillKeyHdrX, lo.y(skillHeaderY), lo.w(35), lo.h(24))
	a.controls.skillIntHdr = a.createStatic(hwnd, "실행 간격", lo.skillIntHdrX, lo.y(skillHeaderY), lo.w(80), lo.h(24))

	// Right column – skill rows
	y := skillFirstRowY
	for i := range config.MaxSkills {
		a.controls.skillEnabled[i] = a.createCheckbox(hwnd, idSkillEnabledBase+i, "", lo.skillChkX, lo.y(y+6), lo.w(22), lo.h(22))
		a.controls.skillNums[i] = a.createStatic(hwnd, strconv.Itoa(i+1), lo.skillNumX, lo.y(y+7), lo.w(skillNumW), lo.h(22))
		a.controls.skillButtons[i] = a.createButton(hwnd, idSkillKeyBase+i, "", lo.skillBtnX, lo.y(y), lo.skillBtnW, lo.h(34))
		a.controls.skillInterval[i] = a.createEdit(hwnd, idSkillIntervalBase+i, "", lo.skillIntervalX, lo.y(y+7), lo.w(skillEditW), lo.h(22))
		a.controls.skillMsLbls[i] = a.createStatic(hwnd, "ms", lo.skillMsX, lo.y(y+6), lo.w(skillMsW), lo.h(22))
		y += skillRowGap
	}

	// Right column – pause section
	a.controls.pauseLabel = a.createStatic(hwnd, "키", lo.pauseLabelX, lo.y(648), lo.w(45), lo.h(24))
	a.controls.pauseButton = a.createButton(hwnd, idPauseKey, "", lo.pauseBtnX, lo.y(642), lo.pauseBtnW, lo.h(34))

	// Status bar
	a.controls.statusLabel = a.createStatic(hwnd, "상태", lo.x(layoutLX+24), lo.y(711), lo.w(55), lo.h(24))
	a.controls.status = a.createStatic(hwnd, "정지.", lo.statusTextX, lo.y(711), lo.statusTextW, lo.h(24))

	a.updateControlsFromConfig()
}

func (a *application) repositionControls() {
	cw, ch := getClientSize(a.hwnd)
	lo := computeLayout(cw, ch)

	moveControl(a.controls.loadButton, lo.loadX, lo.y(26), lo.w(headerBtnW), lo.h(34))
	moveControl(a.controls.saveButton, lo.saveX, lo.y(26), lo.w(headerBtnW), lo.h(34))

	moveControl(a.controls.startLabel, lo.x(layoutLX+24), lo.y(139), lo.w(95), lo.h(24))
	moveControl(a.controls.startButton, lo.x(layoutLX+130), lo.y(134), lo.w(190), lo.h(34))
	moveControl(a.controls.stopLabel, lo.x(layoutLX+24), lo.y(181), lo.w(95), lo.h(24))
	moveControl(a.controls.stopButton, lo.x(layoutLX+130), lo.y(176), lo.w(190), lo.h(34))

	menuY := 282
	for _, menu := range menuControls {
		moveControl(a.controls.menuLabels[menu.id], lo.x(layoutLX+24), lo.y(menuY+5), lo.w(120), lo.h(24))
		moveControl(a.controls.menuButtons[menu.id], lo.x(layoutLX+150), lo.y(menuY), lo.w(170), lo.h(34))
		menuY += 40
	}

	moveControl(a.controls.bulkLabel, lo.bulkLabelX, lo.y(bulkIntervalLabelY), lo.w(78), lo.h(24))
	moveControl(a.controls.bulkInterval, lo.bulkEditX, lo.y(bulkIntervalEditY), lo.w(bulkEditW), lo.h(22))
	moveControl(a.controls.bulkMsLabel, lo.bulkMsX, lo.y(bulkIntervalLabelY), lo.w(bulkMsW), lo.h(24))
	moveControl(a.controls.bulkSkillGapLbl, lo.bulkLabelX, lo.y(bulkSkillGapLabelY), lo.w(78), lo.h(24))
	moveControl(a.controls.bulkSkillGap, lo.bulkEditX, lo.y(bulkSkillGapEditY), lo.w(bulkEditW), lo.h(22))
	moveControl(a.controls.bulkGapMsLabel, lo.bulkMsX, lo.y(bulkSkillGapLabelY), lo.w(bulkMsW), lo.h(24))
	moveControl(a.controls.applyBulk, lo.bulkApplyX, lo.y(bulkApplyY), lo.w(bulkApplyW), lo.h(bulkApplyH))

	moveControl(a.controls.skillUseHdr, lo.skillUseHdrX, lo.y(skillHeaderY), lo.w(45), lo.h(24))
	moveControl(a.controls.skillNumHdr, lo.skillNumHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24))
	moveControl(a.controls.skillKeyHdr, lo.skillKeyHdrX, lo.y(skillHeaderY), lo.w(35), lo.h(24))
	moveControl(a.controls.skillIntHdr, lo.skillIntHdrX, lo.y(skillHeaderY), lo.w(80), lo.h(24))

	y := skillFirstRowY
	for i := range config.MaxSkills {
		moveControl(a.controls.skillEnabled[i], lo.skillChkX, lo.y(y+6), lo.w(22), lo.h(22))
		moveControl(a.controls.skillNums[i], lo.skillNumX, lo.y(y+7), lo.w(skillNumW), lo.h(22))
		moveControl(a.controls.skillButtons[i], lo.skillBtnX, lo.y(y), lo.skillBtnW, lo.h(34))
		moveControl(a.controls.skillInterval[i], lo.skillIntervalX, lo.y(y+7), lo.w(skillEditW), lo.h(22))
		moveControl(a.controls.skillMsLbls[i], lo.skillMsX, lo.y(y+6), lo.w(skillMsW), lo.h(22))
		y += skillRowGap
	}

	moveControl(a.controls.pauseLabel, lo.pauseLabelX, lo.y(648), lo.w(45), lo.h(24))
	moveControl(a.controls.pauseButton, lo.pauseBtnX, lo.y(642), lo.pauseBtnW, lo.h(34))

	moveControl(a.controls.statusLabel, lo.x(layoutLX+24), lo.y(711), lo.w(55), lo.h(24))
	moveControl(a.controls.status, lo.statusTextX, lo.y(711), lo.statusTextW, lo.h(24))

	invalidateRect(a.hwnd, true)
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
	setWindowText(a.controls.bulkSkillGap, strconv.Itoa(a.cfg.SkillGapMS))
	for i := range config.MaxSkills {
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
	skillGap, err := parseSkillGap(getWindowText(a.controls.bulkSkillGap))
	if err != nil {
		messageBox(a.hwnd, "잘못된 키별 간격", err.Error(), mbOK|mbIconError)
		return
	}
	a.cfg.SkillGapMS = skillGap
	setWindowText(a.controls.bulkSkillGap, strconv.Itoa(skillGap))
	for i := range config.MaxSkills {
		setWindowText(a.controls.skillInterval[i], strconv.Itoa(bulkIntervalForSkill(interval, skillGap, i)))
	}
	if skillGap > 0 {
		a.setStatus("일괄 간격을 키별 간격만큼 벌려 적용했습니다.")
		return
	}
	a.setStatus("일괄 간격을 적용했습니다.")
}

func bulkIntervalForSkill(baseInterval int, skillGap int, index int) int {
	return baseInterval + skillGap*index
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
	skillGap, err := parseSkillGap(getWindowText(a.controls.bulkSkillGap))
	if err != nil {
		return fmt.Errorf("키별 간격: %w", err)
	}
	a.cfg.SkillGapMS = skillGap
	for i := range config.MaxSkills {
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
