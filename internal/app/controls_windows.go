//go:build windows

package app

import (
	"fmt"
	"math"
	"strconv"
	"strings"
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
	idClickerHold     = 107

	idBulkInterval = 110
	idApplyBulk    = 111
	idBulkSkillGap = 112
	idInputHold    = 113

	idMenuBase = 120

	idSkillEnabledBase  = 200
	idSkillKeyBase      = 300
	idSkillIntervalBase = 400
	idSkillHoldBase     = 450

	idSave = 500
	idLoad = 501
)

const (
	bulkIntervalLabelY = 125
	bulkIntervalEditY  = 126
	bulkSkillGapLabelY = 158
	bulkSkillGapEditY  = 159
	inputHoldLabelY    = 191
	inputHoldEditY     = 192
	bulkApplyY         = 124
	bulkApplyH         = 64
	skillHeaderY       = 232
	skillFirstRowY     = 262
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
	inputHoldLabel  uintptr
	inputHold       uintptr
	inputHoldMsLbl  uintptr
	applyBulk       uintptr

	// Right column – skill grid headers
	skillUseHdr  uintptr
	skillNumHdr  uintptr
	skillKeyHdr  uintptr
	skillIntHdr  uintptr
	skillHoldHdr uintptr

	// Right column – skill rows
	skillEnabled    [config.MaxSkills]uintptr
	skillNums       [config.MaxSkills]uintptr
	skillButtons    [config.MaxSkills]uintptr
	skillInterval   [config.MaxSkills]uintptr
	skillMsLbls     [config.MaxSkills]uintptr
	skillHold       [config.MaxSkills]uintptr
	skillHoldMsLbls [config.MaxSkills]uintptr

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
	clickerHoldLabel     uintptr
	clickerHold          uintptr
	clickerHoldMsLabel   uintptr

	// Status bar
	statusLabel uintptr
	status      uintptr
}

type menuControl struct {
	id      string
	label   string
	control int
}

type controlPlacementGroup int

const (
	controlGroupHeaderAndBindings controlPlacementGroup = iota
	controlGroupBulk
	controlGroupSkillHeaders
	controlGroupPause
	controlGroupClicker
	controlGroupStatus
)

type controlKind int

const (
	controlKindStatic controlKind = iota
	controlKindButton
	controlKindEdit
)

type controlRect struct {
	x      int
	y      int
	width  int
	height int
}

type controlPlacement struct {
	group controlPlacementGroup
	kind  controlKind
	ref   func(*controlRefs) *uintptr
	id    int
	text  string
	rect  func(uiLayout) controlRect
}

type skillRowRects struct {
	enabled     controlRect
	num         controlRect
	button      controlRect
	interval    controlRect
	msLabel     controlRect
	hold        controlRect
	holdMsLabel controlRect
}

var (
	menuControls           = buildMenuControls()
	fixedControlPlacements = []controlPlacement{
		{
			group: controlGroupHeaderAndBindings,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.loadButton },
			id:    idLoad,
			text:  "불러오기",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.loadX, lo.y(26), lo.w(headerBtnW), lo.h(34)}
			},
		},
		{
			group: controlGroupHeaderAndBindings,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.saveButton },
			id:    idSave,
			text:  "저장하기",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.saveX, lo.y(26), lo.w(headerBtnW), lo.h(34)}
			},
		},
		{
			group: controlGroupHeaderAndBindings,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.startLabel },
			text:  "시작 키",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.x(layoutLX + 24), lo.y(139), lo.w(95), lo.h(24)}
			},
		},
		{
			group: controlGroupHeaderAndBindings,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.startButton },
			id:    idStartKey,
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.x(layoutLX + 130), lo.y(134), lo.w(190), lo.h(34)}
			},
		},
		{
			group: controlGroupHeaderAndBindings,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.stopLabel },
			text:  "종료 키",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.x(layoutLX + 24), lo.y(181), lo.w(95), lo.h(24)}
			},
		},
		{
			group: controlGroupHeaderAndBindings,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.stopButton },
			id:    idStopKey,
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.x(layoutLX + 130), lo.y(176), lo.w(190), lo.h(34)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.bulkLabel },
			text:  "일괄 간격",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkLabelX, lo.y(bulkIntervalLabelY), lo.w(78), lo.h(24)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindEdit,
			ref:   func(c *controlRefs) *uintptr { return &c.bulkInterval },
			id:    idBulkInterval,
			text:  strconv.Itoa(config.DefaultIntervalMS),
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkEditX, lo.y(bulkIntervalEditY), lo.w(bulkEditW), lo.h(22)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.bulkMsLabel },
			text:  "ms",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkMsX, lo.y(bulkIntervalLabelY), lo.w(bulkMsW), lo.h(24)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.bulkSkillGapLbl },
			text:  "키별 간격",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkLabelX, lo.y(bulkSkillGapLabelY), lo.w(78), lo.h(24)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindEdit,
			ref:   func(c *controlRefs) *uintptr { return &c.bulkSkillGap },
			id:    idBulkSkillGap,
			text:  strconv.Itoa(config.DefaultSkillGapMS),
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkEditX, lo.y(bulkSkillGapEditY), lo.w(bulkEditW), lo.h(22)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.bulkGapMsLabel },
			text:  "ms",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkMsX, lo.y(bulkSkillGapLabelY), lo.w(bulkMsW), lo.h(24)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.inputHoldLabel },
			text:  "일괄 눌림",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkLabelX, lo.y(inputHoldLabelY), lo.w(78), lo.h(24)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindEdit,
			ref:   func(c *controlRefs) *uintptr { return &c.inputHold },
			id:    idInputHold,
			text:  strconv.Itoa(config.DefaultInputHoldMS),
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkEditX, lo.y(inputHoldEditY), lo.w(bulkEditW), lo.h(22)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.inputHoldMsLbl },
			text:  "ms",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkMsX, lo.y(inputHoldLabelY), lo.w(bulkMsW), lo.h(24)}
			},
		},
		{
			group: controlGroupBulk,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.applyBulk },
			id:    idApplyBulk,
			text:  "일괄 적용",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.bulkApplyX, lo.y(bulkApplyY), lo.w(bulkApplyW), lo.h(bulkApplyH)}
			},
		},
		{
			group: controlGroupSkillHeaders,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.skillUseHdr },
			text:  "사용",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.skillUseHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24)}
			},
		},
		{
			group: controlGroupSkillHeaders,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.skillNumHdr },
			text:  "기술",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.skillNumHdrX, lo.y(skillHeaderY), lo.w(55), lo.h(24)}
			},
		},
		{
			group: controlGroupSkillHeaders,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.skillKeyHdr },
			text:  "키",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.skillKeyHdrX, lo.y(skillHeaderY), lo.w(35), lo.h(24)}
			},
		},
		{
			group: controlGroupSkillHeaders,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.skillIntHdr },
			text:  "실행 간격",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.skillIntHdrX, lo.y(skillHeaderY), lo.w(80), lo.h(24)}
			},
		},
		{
			group: controlGroupSkillHeaders,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.skillHoldHdr },
			text:  "눌림",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.skillHoldHdrX, lo.y(skillHeaderY), lo.w(50), lo.h(24)}
			},
		},
		{
			group: controlGroupPause,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.pauseLabel },
			text:  "키",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.pauseLabelX, lo.y(pauseRowY + 6), lo.w(45), lo.h(24)}
			},
		},
		{
			group: controlGroupPause,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.pauseButton },
			id:    idPauseKey,
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.pauseBtnX, lo.y(pauseRowY), lo.pauseBtnW, lo.h(34)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerStartLabel },
			text:  "시작",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerStartLabelX, lo.y(clickerHotkeyY + 6), lo.w(44), lo.h(24)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerStartButton },
			id:    idClickerStartKey,
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerStartBtnX, lo.y(clickerHotkeyY), lo.w(clickerStartBtnW), lo.h(34)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerStopLabel },
			text:  "종료",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerStopLabelX, lo.y(clickerHotkeyY + 6), lo.w(44), lo.h(24)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerStopButton },
			id:    idClickerStopKey,
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerStopBtnX, lo.y(clickerHotkeyY), lo.w(clickerStopBtnW), lo.h(34)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerKeyLabel },
			text:  "입력",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerKeyLabelX, lo.y(clickerSettingY + 6), lo.w(44), lo.h(24)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindButton,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerKeyButton },
			id:    idClickerKey,
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerKeyBtnX, lo.y(clickerSettingY), lo.w(clickerKeyBtnW), lo.h(34)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerIntervalLabel },
			text:  "간격",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerIntLabelX, lo.y(clickerSettingY + 6), lo.w(44), lo.h(24)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindEdit,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerInterval },
			id:    idClickerInterval,
			text:  strconv.Itoa(config.DefaultClickerIntervalMS),
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerIntEditX, lo.y(clickerSettingY + 7), lo.w(clickerIntEditW), lo.h(22)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerMsLabel },
			text:  "ms",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerMsLabelX, lo.y(clickerSettingY + 6), lo.w(32), lo.h(24)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerHoldLabel },
			text:  "눌림",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerHoldLabelX, lo.y(clickerSettingY + 6), lo.w(44), lo.h(24)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindEdit,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerHold },
			id:    idClickerHold,
			text:  strconv.Itoa(config.DefaultInputHoldMS),
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerHoldEditX, lo.y(clickerSettingY + 7), lo.w(clickerHoldEditW), lo.h(22)}
			},
		},
		{
			group: controlGroupClicker,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.clickerHoldMsLabel },
			text:  "ms",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.clickerHoldMsX, lo.y(clickerSettingY + 6), lo.w(32), lo.h(24)}
			},
		},
		{
			group: controlGroupStatus,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.statusLabel },
			text:  "상태",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.x(layoutLX + 24), lo.y(statusBarY + 11), lo.w(55), lo.h(24)}
			},
		},
		{
			group: controlGroupStatus,
			kind:  controlKindStatic,
			ref:   func(c *controlRefs) *uintptr { return &c.status },
			text:  "■ 정지.",
			rect: func(lo uiLayout) controlRect {
				return controlRect{lo.statusTextX, lo.y(statusBarY + 11), lo.statusTextW, lo.h(24)}
			},
		},
	}
)

func buildMenuControls() []menuControl {
	definitions := config.MenuBindingDefinitions()
	controls := make([]menuControl, 0, len(definitions))
	for i, definition := range definitions {
		controls = append(controls, menuControl{
			id:      definition.ID,
			label:   definition.UILabel,
			control: idMenuBase + i,
		})
	}
	return controls
}

func menuControlByID(id string) (menuControl, bool) {
	for _, menu := range menuControls {
		if menu.id == id {
			return menu, true
		}
	}
	return menuControl{}, false
}

func menuControlByControl(control int) (menuControl, bool) {
	for _, menu := range menuControls {
		if menu.control == control {
			return menu, true
		}
	}
	return menuControl{}, false
}

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
	_, ok := menuControlByControl(id)
	return ok
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
		if menu, ok := menuControlByID(target.menuID); ok {
			return menu.control
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

func (a *application) createControlPlacements(hwnd uintptr, lo uiLayout, group controlPlacementGroup) {
	for _, placement := range fixedControlPlacements {
		if placement.group != group {
			continue
		}
		*placement.ref(&a.controls) = a.createPlacedControl(hwnd, lo, placement)
	}
}

func (a *application) createPlacedControl(hwnd uintptr, lo uiLayout, placement controlPlacement) uintptr {
	rect := placement.rect(lo)
	switch placement.kind {
	case controlKindStatic:
		return a.createStatic(hwnd, placement.text, rect.x, rect.y, rect.width, rect.height)
	case controlKindButton:
		return a.createButton(hwnd, placement.id, placement.text, rect.x, rect.y, rect.width, rect.height)
	case controlKindEdit:
		return a.createEdit(hwnd, placement.id, placement.text, rect.x, rect.y, rect.width, rect.height)
	default:
		return 0
	}
}

func (a *application) repositionControlPlacements(lo uiLayout, group controlPlacementGroup) {
	for _, placement := range fixedControlPlacements {
		if placement.group != group {
			continue
		}
		moveControlRect(*placement.ref(&a.controls), placement.rect(lo))
	}
}

func moveControlRect(hwnd uintptr, rect controlRect) {
	moveControl(hwnd, rect.x, rect.y, rect.width, rect.height)
}

func menuControlRects(lo uiLayout, y int) (controlRect, controlRect) {
	return controlRect{lo.x(layoutLX + 24), lo.y(y + 5), lo.w(120), lo.h(24)},
		controlRect{lo.x(layoutLX + 150), lo.y(y), lo.w(170), lo.h(34)}
}

func skillRowControlRects(lo uiLayout, y int) skillRowRects {
	return skillRowRects{
		enabled:     controlRect{lo.skillChkX, lo.y(y + 4), lo.w(52), lo.h(26)},
		num:         controlRect{lo.skillNumX, lo.y(y + 7), lo.w(skillNumW), lo.h(22)},
		button:      controlRect{lo.skillBtnX, lo.y(y), lo.skillBtnW, lo.h(34)},
		interval:    controlRect{lo.skillIntervalX, lo.y(y + 7), lo.w(skillEditW), lo.h(22)},
		msLabel:     controlRect{lo.skillMsX, lo.y(y + 6), lo.w(skillMsW), lo.h(22)},
		hold:        controlRect{lo.skillHoldX, lo.y(y + 7), lo.w(skillHoldEditW), lo.h(22)},
		holdMsLabel: controlRect{lo.skillHoldMsX, lo.y(y + 6), lo.w(skillMsW), lo.h(22)},
	}
}

func (a *application) createControls(hwnd uintptr) {
	cw, ch := getClientSize(hwnd)
	lo := computeLayout(cw, ch, a.currentDPI(hwnd))
	a.applyUIScale(lo.uiScale())

	// Header buttons and left column key bindings
	a.createControlPlacements(hwnd, lo, controlGroupHeaderAndBindings)

	menuY := menuFirstY
	for _, menu := range menuControls {
		labelRect, buttonRect := menuControlRects(lo, menuY)
		a.controls.menuLabels[menu.id] = a.createStatic(hwnd, menu.label, labelRect.x, labelRect.y, labelRect.width, labelRect.height)
		a.controls.menuButtons[menu.id] = a.createButton(hwnd, menu.control, "", buttonRect.x, buttonRect.y, buttonRect.width, buttonRect.height)
		menuY += 40
	}

	// Right column – bulk interval section
	a.createControlPlacements(hwnd, lo, controlGroupBulk)

	// Right column – skill grid headers
	a.createControlPlacements(hwnd, lo, controlGroupSkillHeaders)

	// Right column – skill rows
	y := skillFirstRowY
	for i := range config.MaxSkills {
		rects := skillRowControlRects(lo, y)
		a.controls.skillEnabled[i] = a.createButton(hwnd, idSkillEnabledBase+i, "", rects.enabled.x, rects.enabled.y, rects.enabled.width, rects.enabled.height)
		a.controls.skillNums[i] = a.createStatic(hwnd, strconv.Itoa(i+1), rects.num.x, rects.num.y, rects.num.width, rects.num.height)
		a.controls.skillButtons[i] = a.createButton(hwnd, idSkillKeyBase+i, "", rects.button.x, rects.button.y, rects.button.width, rects.button.height)
		a.controls.skillInterval[i] = a.createEdit(hwnd, idSkillIntervalBase+i, "", rects.interval.x, rects.interval.y, rects.interval.width, rects.interval.height)
		a.controls.skillMsLbls[i] = a.createStatic(hwnd, "ms", rects.msLabel.x, rects.msLabel.y, rects.msLabel.width, rects.msLabel.height)
		a.controls.skillHold[i] = a.createEdit(hwnd, idSkillHoldBase+i, "", rects.hold.x, rects.hold.y, rects.hold.width, rects.hold.height)
		a.controls.skillHoldMsLbls[i] = a.createStatic(hwnd, "ms", rects.holdMsLabel.x, rects.holdMsLabel.y, rects.holdMsLabel.width, rects.holdMsLabel.height)
		y += skillRowGap
	}

	// Right column – pause section
	a.createControlPlacements(hwnd, lo, controlGroupPause)

	// Right column – single-key clicker section
	a.createControlPlacements(hwnd, lo, controlGroupClicker)

	// Status bar
	a.createControlPlacements(hwnd, lo, controlGroupStatus)

	a.updateControlsFromConfig()
}

func (a *application) repositionControls() {
	cw, ch := getClientSize(a.hwnd)
	lo := computeLayout(cw, ch, a.currentDPI(a.hwnd))
	a.applyUIScale(lo.uiScale())

	a.repositionControlPlacements(lo, controlGroupHeaderAndBindings)

	menuY := menuFirstY
	for _, menu := range menuControls {
		labelRect, buttonRect := menuControlRects(lo, menuY)
		moveControlRect(a.controls.menuLabels[menu.id], labelRect)
		moveControlRect(a.controls.menuButtons[menu.id], buttonRect)
		menuY += 40
	}

	a.repositionControlPlacements(lo, controlGroupBulk)
	a.repositionControlPlacements(lo, controlGroupSkillHeaders)

	y := skillFirstRowY
	for i := range config.MaxSkills {
		rects := skillRowControlRects(lo, y)
		moveControlRect(a.controls.skillEnabled[i], rects.enabled)
		moveControlRect(a.controls.skillNums[i], rects.num)
		moveControlRect(a.controls.skillButtons[i], rects.button)
		moveControlRect(a.controls.skillInterval[i], rects.interval)
		moveControlRect(a.controls.skillMsLbls[i], rects.msLabel)
		moveControlRect(a.controls.skillHold[i], rects.hold)
		moveControlRect(a.controls.skillHoldMsLbls[i], rects.holdMsLabel)
		y += skillRowGap
	}

	a.repositionControlPlacements(lo, controlGroupPause)
	a.repositionControlPlacements(lo, controlGroupClicker)
	a.repositionControlPlacements(lo, controlGroupStatus)

	invalidateRect(a.hwnd, false)
}

func (a *application) createStatic(parent uintptr, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "STATIC", text, wsChild|wsVisible|ssLeft, x, y, width, height, 0)
}

func (a *application) createButton(parent uintptr, id int, text string, x int, y int, width int, height int) uintptr {
	return a.createControl(parent, "BUTTON", text, wsChild|wsVisible|wsTabStop|bsOwnerDraw, x, y, width, height, id)
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
		a.cfg.Skills[idx].Enabled = a.skillEnabled[idx]
		if hwnd := a.controls.skillEnabled[idx]; hwnd != 0 {
			invalidateRect(hwnd, true)
		}
		a.updateRuntimeStatus()
	case id >= idSkillKeyBase && id < idSkillKeyBase+config.MaxSkills:
		a.startCapture(captureTarget{kind: captureSkill, index: id - idSkillKeyBase})
	case id == idApplyBulk:
		a.applyBulkInterval()
	case id == idSave:
		a.saveConfig()
	case id == idLoad:
		a.loadConfig()
	default:
		if menu, ok := menuControlByControl(id); ok {
			a.startCapture(captureTarget{kind: captureMenu, menuID: menu.id})
			return true
		}
		return false
	}
	return true
}

func (a *application) updateControlsFromConfig() {
	a.cfg.NormalizeForUI()
	ignoreSetWindowText(a.controls.startButton, bindingText(a.cfg.Start))
	ignoreSetWindowText(a.controls.stopButton, bindingText(a.cfg.Stop))
	ignoreSetWindowText(a.controls.pauseButton, bindingText(a.cfg.Pause))
	ignoreSetWindowText(a.controls.clickerStartButton, bindingText(a.cfg.Clicker.Start))
	ignoreSetWindowText(a.controls.clickerStopButton, bindingText(a.cfg.Clicker.Stop))
	ignoreSetWindowText(a.controls.clickerKeyButton, bindingText(a.cfg.Clicker.Key))
	ignoreSetWindowText(a.controls.clickerInterval, strconv.Itoa(a.cfg.Clicker.IntervalMS))
	ignoreSetWindowText(a.controls.clickerHold, strconv.Itoa(a.cfg.Clicker.InputHoldMS))
	ignoreSetWindowText(a.controls.inputHold, strconv.Itoa(a.cfg.InputHoldMS))
	for _, menu := range a.cfg.MenuBindings() {
		if hwnd := a.controls.menuButtons[menu.ID]; hwnd != 0 {
			ignoreSetWindowText(hwnd, bindingText(menu.Binding))
		}
	}
	ignoreSetWindowText(a.controls.bulkSkillGap, strconv.Itoa(a.cfg.SkillGapMS))
	for i := range config.MaxSkills {
		a.skillEnabled[i] = a.cfg.Skills[i].Enabled
		if hwnd := a.controls.skillEnabled[i]; hwnd != 0 {
			invalidateRect(hwnd, true)
		}
		ignoreSetWindowText(a.controls.skillButtons[i], bindingText(a.cfg.Skills[i].Key))
		ignoreSetWindowText(a.controls.skillInterval[i], strconv.Itoa(a.cfg.Skills[i].IntervalMS))
		ignoreSetWindowText(a.controls.skillHold[i], strconv.Itoa(a.cfg.Skills[i].InputHoldMS))
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
	holdText, err := getWindowText(a.controls.inputHold)
	if err != nil {
		messageBox(a.hwnd, "잘못된 눌림 시간", err.Error(), mbOK|mbIconError)
		return
	}
	inputHold, err := parseInputHold(holdText)
	if err != nil {
		messageBox(a.hwnd, "잘못된 눌림 시간", err.Error(), mbOK|mbIconError)
		return
	}
	a.cfg.SkillGapMS = skillGap
	a.cfg.InputHoldMS = inputHold
	ignoreSetWindowText(a.controls.bulkSkillGap, strconv.Itoa(skillGap))
	ignoreSetWindowText(a.controls.inputHold, strconv.Itoa(inputHold))
	for i := range config.MaxSkills {
		skillInterval, err := bulkIntervalForSkill(interval, skillGap, i)
		if err != nil {
			messageBox(a.hwnd, "잘못된 간격", err.Error(), mbOK|mbIconError)
			return
		}
		ignoreSetWindowText(a.controls.skillInterval[i], strconv.Itoa(skillInterval))
		ignoreSetWindowText(a.controls.skillHold[i], strconv.Itoa(inputHold))
	}
	if skillGap > 0 {
		a.setStatus("일괄 간격은 키별 간격만큼 벌리고 눌림 시간을 적용했습니다.")
		return
	}
	a.setStatus("일괄 간격과 눌림 시간을 적용했습니다.")
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
	saveLastConfigPath(path)
	a.setStatusWithKeyConflictWarning("저장 완료: " + a.configPath)
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
	a.requestRuntimeStop("")
	a.runtimeInputTarget.Store(0)
	a.cfg = loaded
	a.configPath = path
	saveLastConfigPath(path)
	previous := a.capture
	a.capture = captureTarget{}
	a.updateControlsFromConfig()
	a.invalidateCaptureControls(previous)
	a.setStatusWithKeyConflictWarning("불러오기 완료: " + a.configPath)
}

func (a *application) startRunnerFromHotkey() {
	if a.shuttingDown.Load() {
		return
	}
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
	if !a.clicker.Running() {
		a.captureRuntimeInputTarget()
	}
	if a.runner.StartContext(a.ctx, a.cfg) {
		a.updateRuntimeStatus()
		return
	}
	a.clearRuntimeInputTargetIfIdle()
	a.updateRuntimeStatus()
}

func (a *application) startClickerFromHotkey() {
	if a.shuttingDown.Load() {
		return
	}
	if a.clicker.Running() {
		a.updateRuntimeStatus()
		return
	}
	if err := a.syncConfigFromControls(); err != nil {
		messageBox(a.hwnd, "잘못된 설정", err.Error(), mbOK|mbIconError)
		return
	}
	if !clickerRunnable(a.cfg.Clicker) {
		a.setStatus("클릭 반복에 사용할 입력 키, 간격, 눌림 시간을 지정하세요.")
		return
	}
	if !a.runner.Running() {
		a.captureRuntimeInputTarget()
	}
	if a.clicker.StartContext(a.ctx, a.cfg.Clicker) {
		a.updateRuntimeStatus()
		return
	}
	a.clearRuntimeInputTargetIfIdle()
	a.updateRuntimeStatus()
}

func (a *application) stopAllRunners(status string) {
	if a.requestRuntimeStop(status) {
		a.setStatus(status)
		return
	}
	a.updateRuntimeStatus()
}

func (a *application) requestRuntimeStop(status string) bool {
	handles := requestRuntimeRunnersStop(a.runner, a.clicker)
	if len(handles) == 0 {
		go func() {
			releaseInjectedInputs()
			releaseMouseButtons()
			a.clearRuntimeInputTargetIfIdle()
		}()
		return false
	}

	if status != "" {
		a.runtimeStopMu.Lock()
		a.pendingStopStatus = status
		a.runtimeStopMu.Unlock()
	}

	go func() {
		releaseInjectedInputs()
		releaseMouseButtons()
		waitRuntimeStopHandles(handles)
		releaseInjectedInputs()
		releaseMouseButtons()
		a.clearRuntimeInputTargetIfIdle()
		if a.shuttingDown.Load() || status == "" || a.hwnd == 0 || a.winapi.postMessage == nil {
			return
		}
		a.winapi.postMessage(a.hwnd, wmRuntimeStopComplete, 0, 0)
	}()
	return true
}

func (a *application) finishAsyncRuntimeStop() {
	a.runtimeStopMu.Lock()
	status := a.pendingStopStatus
	a.pendingStopStatus = ""
	a.runtimeStopMu.Unlock()
	if status != "" {
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
	holdText, err := getWindowText(a.controls.inputHold)
	if err != nil {
		return fmt.Errorf("눌림 시간: %w", err)
	}
	inputHold, err := parseInputHold(holdText)
	if err != nil {
		return fmt.Errorf("눌림 시간: %w", err)
	}
	a.cfg.InputHoldMS = inputHold
	clickerText, err := getWindowText(a.controls.clickerInterval)
	if err != nil {
		return fmt.Errorf("클릭 반복: %w", err)
	}
	clickerInterval, err := parseInterval(clickerText)
	if err != nil {
		return fmt.Errorf("클릭 반복: %w", err)
	}
	a.cfg.Clicker.IntervalMS = clickerInterval
	clickerHoldText, err := getWindowText(a.controls.clickerHold)
	if err != nil {
		return fmt.Errorf("클릭 반복 눌림 시간: %w", err)
	}
	clickerHold, err := parseInputHold(clickerHoldText)
	if err != nil {
		return fmt.Errorf("클릭 반복 눌림 시간: %w", err)
	}
	a.cfg.Clicker.InputHoldMS = clickerHold
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
		holdText, err := getWindowText(a.controls.skillHold[i])
		if err != nil {
			return fmt.Errorf("기술 %d 눌림 시간: %w", i+1, err)
		}
		hold, err := parseInputHold(holdText)
		if err != nil {
			return fmt.Errorf("기술 %d 눌림 시간: %w", i+1, err)
		}
		a.cfg.Skills[i].InputHoldMS = hold
		a.cfg.Skills[i].Enabled = a.skillEnabled[i]
	}
	a.cfg.NormalizeForUI()
	return a.cfg.Validate()
}

func (a *application) updateRuntimeStatus() {
	status := ""
	switch {
	case a.runner.Paused() && a.clicker.Paused():
		status = "⏸ 기술 입력과 클릭 반복을 일시정지했습니다."
	case a.runner.Paused() && a.clicker.Running():
		status = "⏸ 기술 입력은 일시정지, 클릭 반복 실행 중."
	case a.clicker.Paused() && a.runner.Running():
		status = "⏸ 클릭 반복은 일시정지, 기술 반복 실행 중."
	case a.runner.Paused():
		status = "⏸ 일시정지 키를 누르고 있어 기술 입력을 중지했습니다."
	case a.clicker.Paused():
		status = "⏸ 일시정지 키를 누르고 있어 클릭 반복을 중지했습니다."
	case a.runner.Running() && a.clicker.Running():
		status = "▶ 기술 반복과 클릭 반복 실행 중."
	case a.runner.Running():
		status = "▶ 기술 반복 실행 중."
	case a.clicker.Running():
		status = "▶ 클릭 반복 실행 중."
	default:
		status = "■ 정지."
	}
	a.setStatusWithKeyConflictWarning(status)
}

func (a *application) setStatusWithKeyConflictWarning(status string) {
	warning := keyConflictWarningStatus(a.cfg.KeyConflicts())
	if warning != "" {
		a.setStatus(status + " " + warning)
		return
	}
	a.setStatus(status)
}

func keyConflictWarningStatus(conflicts []config.KeyConflict) string {
	if len(conflicts) == 0 {
		return ""
	}
	first := conflicts[0]
	labels := make([]string, 0, len(first.Usages))
	for _, usage := range first.Usages {
		labels = append(labels, usage.Label)
	}
	message := fmt.Sprintf("키 충돌 경고: %s 중복 지정(%s).", bindingText(first.Key), strings.Join(labels, ", "))
	if len(conflicts) > 1 {
		message += fmt.Sprintf(" 외 %d건.", len(conflicts)-1)
	}
	return message
}

func (a *application) setStatus(text string) {
	a.statusText = text
	if a.controls.status != 0 {
		ignoreSetWindowText(a.controls.status, text)
	}
	if a.hwnd != 0 {
		invalidateRect(a.hwnd, false)
	}
}
