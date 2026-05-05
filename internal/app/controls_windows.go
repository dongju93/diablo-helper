//go:build windows

package app

import (
	"fmt"
	"math"
	"strconv"
	"unsafe"

	"github.com/dongju93/diablo-helper/internal/config"
)

const (
	idStartKey = 100
	idStopKey  = 101
	idPauseKey = 102

	idClickerStartKey = 103
	idClickerStopKey  = 104
	idClickerKey      = 105
	idClickerInterval = 106

	idBulkInterval = 110
	idApplyBulk    = 111
	idBulkSkillGap = 112

	idMenuCharacter   = 120
	idMenuSkillAssign = 121
	idMenuTalents     = 122
	idMenuMap         = 123
	idMenuJournal     = 124
	idMenuSocial      = 125
	idMenuClan        = 126
	idMenuTownPortal  = 127
	idMenuCollection  = 128
	idMenuShop        = 129

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
	menuPanelY         = 234
	menuPanelH         = 466
	menuTitleY         = 250
	menuFirstY         = 282
	clickerPanelY      = 600
	clickerPanelH      = 144
	clickerTitleY      = 616
	clickerHotkeyY     = 648
	clickerSettingY    = 688
	pausePanelY        = 760
	pausePanelH        = 84
	pauseTitleY        = 776
	pauseRowY          = 796
	statusBarY         = 860
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

	// Left column – single-key clicker section
	clickerStartLabel    uintptr
	clickerStartButton   uintptr
	clickerStopLabel     uintptr
	clickerStopButton    uintptr
	clickerKeyLabel      uintptr
	clickerKeyButton     uintptr
	clickerIntervalLabel uintptr
	clickerInterval      uintptr
	clickerMsLabel       uintptr

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
		{id: "character", label: "캐릭터", control: idMenuCharacter},
		{id: "skill_assign", label: "스킬 배치", control: idMenuSkillAssign},
		{id: "talents", label: "능력치", control: idMenuTalents},
		{id: "map", label: "지도", control: idMenuMap},
		{id: "journal", label: "일지", control: idMenuJournal},
		{id: "social", label: "소셜", control: idMenuSocial},
		{id: "clan", label: "클랜", control: idMenuClan},
		{id: "town_portal", label: "차원문", control: idMenuTownPortal},
		{id: "collection", label: "컬렉션", control: idMenuCollection},
		{id: "shop", label: "상점", control: idMenuShop},
	}
)

func (a *application) isPrimaryButton(id int) bool {
	return id == idSave || id == idApplyBulk
}

func (a *application) isToggleButton(id int) bool {
	return id >= idSkillEnabledBase && id < idSkillEnabledBase+config.MaxSkills
}

func (a *application) isBindingButton(id int) bool {
	if id == idStartKey || id == idStopKey || id == idPauseKey ||
		id == idClickerStartKey || id == idClickerStopKey || id == idClickerKey {
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
	case captureClickerStart:
		return idClickerStartKey
	case captureClickerStop:
		return idClickerStopKey
	case captureClickerKey:
		return idClickerKey
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
	cw, ch := getClientSize(hwnd)
	lo := computeLayout(cw, ch, a.currentDPI(hwnd))
	a.applyUIScale(lo.uiScale())

	// Header buttons (right-anchored)
	a.controls.loadButton = a.createButton(hwnd, idLoad, "불러오기", lo.loadX, lo.y(26), lo.w(headerBtnW), lo.h(34))
	a.controls.saveButton = a.createButton(hwnd, idSave, "저장하기", lo.saveX, lo.y(26), lo.w(headerBtnW), lo.h(34))

	// Left column – key bindings
	a.controls.startLabel = a.createStatic(hwnd, "시작 키", lo.x(layoutLX+24), lo.y(139), lo.w(95), lo.h(24))
	a.controls.startButton = a.createButton(hwnd, idStartKey, "", lo.x(layoutLX+130), lo.y(134), lo.w(190), lo.h(34))
	a.controls.stopLabel = a.createStatic(hwnd, "종료 키", lo.x(layoutLX+24), lo.y(181), lo.w(95), lo.h(24))
	a.controls.stopButton = a.createButton(hwnd, idStopKey, "", lo.x(layoutLX+130), lo.y(176), lo.w(190), lo.h(34))

	menuY := menuFirstY
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
	a.controls.skillUseHdr = a.createStatic(hwnd, "사용", lo.skillUseHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24))
	a.controls.skillNumHdr = a.createStatic(hwnd, "기술", lo.skillNumHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24))
	a.controls.skillKeyHdr = a.createStatic(hwnd, "키", lo.skillKeyHdrX, lo.y(skillHeaderY), lo.w(35), lo.h(24))
	a.controls.skillIntHdr = a.createStatic(hwnd, "실행 간격", lo.skillIntHdrX, lo.y(skillHeaderY), lo.w(80), lo.h(24))

	// Right column – skill rows
	y := skillFirstRowY
	for i := range config.MaxSkills {
		a.controls.skillEnabled[i] = a.createButton(hwnd, idSkillEnabledBase+i, "", lo.skillChkX, lo.y(y+4), lo.w(52), lo.h(26))
		a.controls.skillNums[i] = a.createStatic(hwnd, strconv.Itoa(i+1), lo.skillNumX, lo.y(y+7), lo.w(skillNumW), lo.h(22))
		a.controls.skillButtons[i] = a.createButton(hwnd, idSkillKeyBase+i, "", lo.skillBtnX, lo.y(y), lo.skillBtnW, lo.h(34))
		a.controls.skillInterval[i] = a.createEdit(hwnd, idSkillIntervalBase+i, "", lo.skillIntervalX, lo.y(y+7), lo.w(skillEditW), lo.h(22))
		a.controls.skillMsLbls[i] = a.createStatic(hwnd, "ms", lo.skillMsX, lo.y(y+6), lo.w(skillMsW), lo.h(22))
		y += skillRowGap
	}

	// Right column – pause section
	a.controls.pauseLabel = a.createStatic(hwnd, "키", lo.pauseLabelX, lo.y(pauseRowY+6), lo.w(45), lo.h(24))
	a.controls.pauseButton = a.createButton(hwnd, idPauseKey, "", lo.pauseBtnX, lo.y(pauseRowY), lo.pauseBtnW, lo.h(34))

	// Right column – single-key clicker section
	a.controls.clickerStartLabel = a.createStatic(hwnd, "시작", lo.clickerStartLabelX, lo.y(clickerHotkeyY+6), lo.w(44), lo.h(24))
	a.controls.clickerStartButton = a.createButton(hwnd, idClickerStartKey, "", lo.clickerStartBtnX, lo.y(clickerHotkeyY), lo.w(clickerStartBtnW), lo.h(34))
	a.controls.clickerStopLabel = a.createStatic(hwnd, "종료", lo.clickerStopLabelX, lo.y(clickerHotkeyY+6), lo.w(44), lo.h(24))
	a.controls.clickerStopButton = a.createButton(hwnd, idClickerStopKey, "", lo.clickerStopBtnX, lo.y(clickerHotkeyY), lo.w(clickerStopBtnW), lo.h(34))
	a.controls.clickerKeyLabel = a.createStatic(hwnd, "입력", lo.clickerKeyLabelX, lo.y(clickerSettingY+6), lo.w(44), lo.h(24))
	a.controls.clickerKeyButton = a.createButton(hwnd, idClickerKey, "", lo.clickerKeyBtnX, lo.y(clickerSettingY), lo.w(clickerKeyBtnW), lo.h(34))
	a.controls.clickerIntervalLabel = a.createStatic(hwnd, "간격", lo.clickerIntLabelX, lo.y(clickerSettingY+6), lo.w(44), lo.h(24))
	a.controls.clickerInterval = a.createEdit(hwnd, idClickerInterval, strconv.Itoa(config.DefaultClickerIntervalMS), lo.clickerIntEditX, lo.y(clickerSettingY+7), lo.w(clickerIntEditW), lo.h(22))
	a.controls.clickerMsLabel = a.createStatic(hwnd, "ms", lo.clickerMsLabelX, lo.y(clickerSettingY+6), lo.w(32), lo.h(24))

	// Status bar
	a.controls.statusLabel = a.createStatic(hwnd, "상태", lo.x(layoutLX+24), lo.y(statusBarY+11), lo.w(55), lo.h(24))
	a.controls.status = a.createStatic(hwnd, "■ 정지.", lo.statusTextX, lo.y(statusBarY+11), lo.statusTextW, lo.h(24))

	a.updateControlsFromConfig()
}

func (a *application) repositionControls() {
	cw, ch := getClientSize(a.hwnd)
	lo := computeLayout(cw, ch, a.currentDPI(a.hwnd))
	a.applyUIScale(lo.uiScale())

	moveControl(a.controls.loadButton, lo.loadX, lo.y(26), lo.w(headerBtnW), lo.h(34))
	moveControl(a.controls.saveButton, lo.saveX, lo.y(26), lo.w(headerBtnW), lo.h(34))

	moveControl(a.controls.startLabel, lo.x(layoutLX+24), lo.y(139), lo.w(95), lo.h(24))
	moveControl(a.controls.startButton, lo.x(layoutLX+130), lo.y(134), lo.w(190), lo.h(34))
	moveControl(a.controls.stopLabel, lo.x(layoutLX+24), lo.y(181), lo.w(95), lo.h(24))
	moveControl(a.controls.stopButton, lo.x(layoutLX+130), lo.y(176), lo.w(190), lo.h(34))

	menuY := menuFirstY
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

	moveControl(a.controls.skillUseHdr, lo.skillUseHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24))
	moveControl(a.controls.skillNumHdr, lo.skillNumHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24))
	moveControl(a.controls.skillKeyHdr, lo.skillKeyHdrX, lo.y(skillHeaderY), lo.w(35), lo.h(24))
	moveControl(a.controls.skillIntHdr, lo.skillIntHdrX, lo.y(skillHeaderY), lo.w(80), lo.h(24))

	y := skillFirstRowY
	for i := range config.MaxSkills {
		moveControl(a.controls.skillEnabled[i], lo.skillChkX, lo.y(y+4), lo.w(52), lo.h(26))
		moveControl(a.controls.skillNums[i], lo.skillNumX, lo.y(y+7), lo.w(skillNumW), lo.h(22))
		moveControl(a.controls.skillButtons[i], lo.skillBtnX, lo.y(y), lo.skillBtnW, lo.h(34))
		moveControl(a.controls.skillInterval[i], lo.skillIntervalX, lo.y(y+7), lo.w(skillEditW), lo.h(22))
		moveControl(a.controls.skillMsLbls[i], lo.skillMsX, lo.y(y+6), lo.w(skillMsW), lo.h(22))
		y += skillRowGap
	}

	moveControl(a.controls.pauseLabel, lo.pauseLabelX, lo.y(pauseRowY+6), lo.w(45), lo.h(24))
	moveControl(a.controls.pauseButton, lo.pauseBtnX, lo.y(pauseRowY), lo.pauseBtnW, lo.h(34))

	moveControl(a.controls.clickerStartLabel, lo.clickerStartLabelX, lo.y(clickerHotkeyY+6), lo.w(44), lo.h(24))
	moveControl(a.controls.clickerStartButton, lo.clickerStartBtnX, lo.y(clickerHotkeyY), lo.w(clickerStartBtnW), lo.h(34))
	moveControl(a.controls.clickerStopLabel, lo.clickerStopLabelX, lo.y(clickerHotkeyY+6), lo.w(44), lo.h(24))
	moveControl(a.controls.clickerStopButton, lo.clickerStopBtnX, lo.y(clickerHotkeyY), lo.w(clickerStopBtnW), lo.h(34))
	moveControl(a.controls.clickerKeyLabel, lo.clickerKeyLabelX, lo.y(clickerSettingY+6), lo.w(44), lo.h(24))
	moveControl(a.controls.clickerKeyButton, lo.clickerKeyBtnX, lo.y(clickerSettingY), lo.w(clickerKeyBtnW), lo.h(34))
	moveControl(a.controls.clickerIntervalLabel, lo.clickerIntLabelX, lo.y(clickerSettingY+6), lo.w(44), lo.h(24))
	moveControl(a.controls.clickerInterval, lo.clickerIntEditX, lo.y(clickerSettingY+7), lo.w(clickerIntEditW), lo.h(22))
	moveControl(a.controls.clickerMsLabel, lo.clickerMsLabelX, lo.y(clickerSettingY+6), lo.w(32), lo.h(24))

	moveControl(a.controls.statusLabel, lo.x(layoutLX+24), lo.y(statusBarY+11), lo.w(55), lo.h(24))
	moveControl(a.controls.status, lo.statusTextX, lo.y(statusBarY+11), lo.statusTextW, lo.h(24))

	invalidateRect(a.hwnd, false)
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
	sendMessage(hwnd, emLimitText, maxEditTextLen, 0)
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
	case id == idClickerStartKey:
		a.startCapture(captureTarget{kind: captureClickerStart})
	case id == idClickerStopKey:
		a.startCapture(captureTarget{kind: captureClickerStop})
	case id == idClickerKey:
		a.startCapture(captureTarget{kind: captureClickerKey})
	case id >= idSkillEnabledBase && id < idSkillEnabledBase+config.MaxSkills:
		idx := id - idSkillEnabledBase
		a.skillEnabled[idx] = !a.skillEnabled[idx]
		if hwnd := a.controls.skillEnabled[idx]; hwnd != 0 {
			invalidateRect(hwnd, true)
		}
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
	a.cfg.NormalizeForUI()
	setWindowText(a.controls.startButton, bindingText(a.cfg.Start))
	setWindowText(a.controls.stopButton, bindingText(a.cfg.Stop))
	setWindowText(a.controls.pauseButton, bindingText(a.cfg.Pause))
	setWindowText(a.controls.clickerStartButton, bindingText(a.cfg.Clicker.Start))
	setWindowText(a.controls.clickerStopButton, bindingText(a.cfg.Clicker.Stop))
	setWindowText(a.controls.clickerKeyButton, bindingText(a.cfg.Clicker.Key))
	setWindowText(a.controls.clickerInterval, strconv.Itoa(a.cfg.Clicker.IntervalMS))
	for _, menu := range a.cfg.MenuBindings() {
		if hwnd := a.controls.menuButtons[menu.ID]; hwnd != 0 {
			setWindowText(hwnd, bindingText(menu.Binding))
		}
	}
	setWindowText(a.controls.bulkSkillGap, strconv.Itoa(a.cfg.SkillGapMS))
	for i := range config.MaxSkills {
		a.skillEnabled[i] = a.cfg.Skills[i].Enabled
		if hwnd := a.controls.skillEnabled[i]; hwnd != 0 {
			invalidateRect(hwnd, true)
		}
		setWindowText(a.controls.skillButtons[i], bindingText(a.cfg.Skills[i].Key))
		setWindowText(a.controls.skillInterval[i], strconv.Itoa(a.cfg.Skills[i].IntervalMS))
	}
	a.updateRuntimeStatus()
}

func (a *application) applyBulkInterval() {
	bulkText, err := getWindowText(a.controls.bulkInterval)
	if err != nil {
		messageBox(a.hwnd, "잘못된 간격", err.Error(), mbOK|mbIconError)
		return
	}
	interval, err := parseInterval(bulkText)
	if err != nil {
		messageBox(a.hwnd, "잘못된 간격", err.Error(), mbOK|mbIconError)
		return
	}
	gapText, err := getWindowText(a.controls.bulkSkillGap)
	if err != nil {
		messageBox(a.hwnd, "잘못된 키별 간격", err.Error(), mbOK|mbIconError)
		return
	}
	skillGap, err := parseSkillGap(gapText)
	if err != nil {
		messageBox(a.hwnd, "잘못된 키별 간격", err.Error(), mbOK|mbIconError)
		return
	}
	a.cfg.SkillGapMS = skillGap
	setWindowText(a.controls.bulkSkillGap, strconv.Itoa(skillGap))
	for i := range config.MaxSkills {
		skillInterval, err := bulkIntervalForSkill(interval, skillGap, i)
		if err != nil {
			messageBox(a.hwnd, "잘못된 간격", err.Error(), mbOK|mbIconError)
			return
		}
		setWindowText(a.controls.skillInterval[i], strconv.Itoa(skillInterval))
	}
	if skillGap > 0 {
		a.setStatus("일괄 간격을 키별 간격만큼 벌려 적용했습니다.")
		return
	}
	a.setStatus("일괄 간격을 적용했습니다.")
}

func bulkIntervalForSkill(baseInterval int, skillGap int, index int) (int, error) {
	if baseInterval < config.MinimumIntervalMS {
		return 0, fmt.Errorf("실행 간격은 최소 %dms 이상이어야 합니다", config.MinimumIntervalMS)
	}
	if baseInterval > config.MaximumIntervalMS {
		return 0, fmt.Errorf("실행 간격은 최대 %dms 이하여야 합니다", config.MaximumIntervalMS)
	}
	if skillGap < 0 {
		return 0, fmt.Errorf("키별 간격은 0ms 이상이어야 합니다")
	}
	if skillGap > config.MaximumSkillGapMS {
		return 0, fmt.Errorf("키별 간격은 최대 %dms 이하여야 합니다", config.MaximumSkillGapMS)
	}
	if index < 0 {
		return 0, fmt.Errorf("기술 번호가 올바르지 않습니다")
	}

	base := int64(baseInterval)
	gap := int64(skillGap)
	row := int64(index)
	if gap > 0 && row > (math.MaxInt64-base)/gap {
		return 0, fmt.Errorf("적용된 실행 간격이 너무 큽니다")
	}
	interval := base + gap*row
	if interval > int64(config.MaximumIntervalMS) {
		return 0, fmt.Errorf("적용된 실행 간격은 최대 %dms 이하여야 합니다", config.MaximumIntervalMS)
	}
	if !config.MillisecondsFitDuration(int(interval)) {
		return 0, fmt.Errorf("적용된 실행 간격이 너무 큽니다")
	}
	return int(interval), nil
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
	saveOptions := config.SaveOptions{}
	if !config.HasTOMLExtension(path) {
		confirmed, err := a.confirmNonTOMLSave(path)
		if err != nil {
			messageBox(a.hwnd, "저장 확인 실패", err.Error(), mbOK|mbIconError)
			return
		}
		if !confirmed {
			a.setStatus("저장을 취소했습니다.")
			return
		}
		saveOptions.AllowNonTOMLExtension = true
	}
	if err := config.SaveFileWithOptions(path, a.cfg, saveOptions); err != nil {
		messageBox(a.hwnd, "저장 실패", err.Error(), mbOK|mbIconError)
		return
	}
	a.configPath = path
	a.setStatus("저장 완료: " + a.configPath)
}

func (a *application) confirmNonTOMLSave(path string) (bool, error) {
	result, err := messageBoxResult(
		a.hwnd,
		"확장자 확인",
		"선택한 파일은 .toml 설정 파일이 아닙니다.\n\n"+path+"\n\n이 경로에 저장하시겠습니까?",
		mbYesNo|mbIconWarning,
	)
	if err != nil {
		return false, err
	}
	return result == idYes, nil
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
	a.clicker.Stop()
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
		a.updateRuntimeStatus()
		return
	}
	a.updateRuntimeStatus()
}

func (a *application) startClickerFromHotkey() {
	if a.clicker.Running() {
		a.updateRuntimeStatus()
		return
	}
	if err := a.syncConfigFromControls(); err != nil {
		messageBox(a.hwnd, "잘못된 설정", err.Error(), mbOK|mbIconError)
		return
	}
	if !clickerRunnable(a.cfg.Clicker) {
		a.setStatus("클릭 반복에 사용할 입력 키와 간격을 지정하세요.")
		return
	}
	if a.clicker.Start(a.cfg.Clicker) {
		a.updateRuntimeStatus()
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

func (a *application) stopAllRunners(status string) {
	stopped := a.runner.Stop()
	stopped = a.clicker.Stop() || stopped
	if stopped {
		a.setStatus(status)
		return
	}
	a.updateRuntimeStatus()
}

func (a *application) syncConfigFromControls() error {
	a.cfg.NormalizeForUI()
	gapText, err := getWindowText(a.controls.bulkSkillGap)
	if err != nil {
		return fmt.Errorf("키별 간격: %w", err)
	}
	skillGap, err := parseSkillGap(gapText)
	if err != nil {
		return fmt.Errorf("키별 간격: %w", err)
	}
	a.cfg.SkillGapMS = skillGap
	clickerText, err := getWindowText(a.controls.clickerInterval)
	if err != nil {
		return fmt.Errorf("클릭 반복: %w", err)
	}
	clickerInterval, err := parseInterval(clickerText)
	if err != nil {
		return fmt.Errorf("클릭 반복: %w", err)
	}
	a.cfg.Clicker.IntervalMS = clickerInterval
	for i := range config.MaxSkills {
		skillText, err := getWindowText(a.controls.skillInterval[i])
		if err != nil {
			return fmt.Errorf("기술 %d: %w", i+1, err)
		}
		interval, err := parseInterval(skillText)
		if err != nil {
			return fmt.Errorf("기술 %d: %w", i+1, err)
		}
		a.cfg.Skills[i].IntervalMS = interval
		a.cfg.Skills[i].Enabled = a.skillEnabled[i]
	}
	a.cfg.NormalizeForUI()
	return a.cfg.Validate()
}

func (a *application) updateRuntimeStatus() {
	switch {
	case a.runner.Paused() && a.clicker.Paused():
		a.setStatus("⏸ 기술 입력과 클릭 반복을 일시정지했습니다.")
	case a.runner.Paused() && a.clicker.Running():
		a.setStatus("⏸ 기술 입력은 일시정지, 클릭 반복 실행 중.")
	case a.clicker.Paused() && a.runner.Running():
		a.setStatus("⏸ 클릭 반복은 일시정지, 기술 반복 실행 중.")
	case a.runner.Paused():
		a.setStatus("⏸ 일시정지 키를 누르고 있어 기술 입력을 중지했습니다.")
	case a.clicker.Paused():
		a.setStatus("⏸ 일시정지 키를 누르고 있어 클릭 반복을 중지했습니다.")
	case a.runner.Running() && a.clicker.Running():
		a.setStatus("▶ 기술 반복과 클릭 반복 실행 중.")
	case a.runner.Running():
		a.setStatus("▶ 기술 반복 실행 중.")
	case a.clicker.Running():
		a.setStatus("▶ 클릭 반복 실행 중.")
	default:
		a.setStatus("■ 정지.")
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
