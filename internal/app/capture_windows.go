//go:build windows

package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dongju93/diablo-helper/internal/config"
)

type captureKind int

const (
	captureNone captureKind = iota
	captureStart
	captureStop
	capturePause
	captureMenu
	captureSkill
	captureClickerStart
	captureClickerStop
	captureClickerKey
)

type captureTarget struct {
	kind   captureKind
	index  int
	menuID string
}

// pressedKeys keeps repeat-suppression state as bits instead of heap map keys.
type pressedKeys [4]uint64

func (p *pressedKeys) set(vk uint16) {
	if vk > 255 {
		return
	}
	p[vk/64] |= uint64(1) << (vk % 64)
}

func (p *pressedKeys) clear(vk uint16) {
	if vk > 255 {
		return
	}
	p[vk/64] &^= uint64(1) << (vk % 64)
}

func (p *pressedKeys) has(vk uint16) bool {
	if vk > 255 {
		return false
	}
	return p[vk/64]&(uint64(1)<<(vk%64)) != 0
}

func (p *pressedKeys) any() bool {
	for _, slot := range p {
		if slot != 0 {
			return true
		}
	}
	return false
}

func (t captureTarget) valid() bool {
	return t.kind != captureNone
}

func (a *application) startCapture(target captureTarget) {
	previous := a.capture
	a.capture = target
	a.invalidateCaptureControls(previous, target)
	a.setStatus("할당할 키를 입력하세요. Esc는 해제입니다.")
}

func (a *application) handleKeyEvent(vk uint16, down bool) bool {
	if a.shuttingDown.Load() {
		return false
	}
	if down {
		if a.pressed.has(vk) {
			return false
		}
		if !a.shouldTrackPressedKey(vk) {
			return false
		}
		a.pressed.set(vk)
	} else {
		wasPressed := a.pressed.has(vk)
		a.pressed.clear(vk)
		if wasPressed && sameKey(vk, a.cfg.Pause) && (a.runner.Running() || a.clicker.Running()) {
			a.runner.SetPaused(false)
			a.clicker.SetPaused(false)
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
		if rejected, label := captureRejectsOutputKey(a.capture.kind, vk); rejected {
			a.rejectCapturedKey(a.capture, label)
			return true
		}
		if vk == vkLButton && captureRejectsMouseLeft(a.capture.kind) {
			a.setStatus(captureRejectionMessage(a.capture, "Mouse Left"))
			return false
		}
		a.assignCapturedKey(vk)
		return true
	}

	if a.handleRuntimeControlKey(vk) {
		return false
	}

	a.handleStartKey(vk)
	return false
}

func (a *application) shouldTrackPressedKey(vk uint16) bool {
	if a.capture.valid() {
		return true
	}

	runnerRunning := a.runner.Running()
	clickerRunning := a.clicker.Running()

	if runnerRunning {
		if sameKey(vk, a.cfg.Stop) || sameKey(vk, a.cfg.Pause) || a.menuKeyMatches(vk) {
			return true
		}
	} else if sameKey(vk, a.cfg.Start) {
		return true
	}

	if clickerRunning {
		return sameKey(vk, a.cfg.Clicker.Stop) || sameKey(vk, a.cfg.Pause) || a.menuKeyMatches(vk)
	}
	return sameKey(vk, a.cfg.Clicker.Start)
}

func (a *application) handleRuntimeControlKey(vk uint16) bool {
	stopRunner := a.runner.Running() && sameKey(vk, a.cfg.Stop)
	stopClicker := a.clicker.Running() && sameKey(vk, a.cfg.Clicker.Stop)
	stopped := false
	if stopRunner || stopClicker {
		stopped = a.requestRuntimeStop("종료 키 입력으로 정지했습니다.")
	}
	if stopped {
		a.setStatus("종료 키 입력으로 정지했습니다.")
		return true
	}

	if sameKey(vk, a.cfg.Pause) && (a.runner.Running() || a.clicker.Running()) {
		a.runner.SetPaused(true)
		a.clicker.SetPaused(true)
		a.updateRuntimeStatus()
		return true
	}

	if (a.runner.Running() || a.clicker.Running()) && a.menuKeyMatches(vk) {
		a.stopAllRunners("게임 메뉴 키 입력으로 정지했습니다.")
		return true
	}
	return false
}

func (a *application) handleStartKey(vk uint16) {
	if !a.runner.Running() && sameKey(vk, a.cfg.Start) {
		a.startRunnerFromHotkey()
	}
	if !a.clicker.Running() && sameKey(vk, a.cfg.Clicker.Start) {
		a.startClickerFromHotkey()
	}
}

func (a *application) assignCapturedKey(vk uint16) {
	target := a.capture
	binding := config.KeyBinding{Name: config.KeyDisplayName(int(vk)), VK: int(vk)}
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
	case captureClickerStart:
		a.cfg.Clicker.Start = binding
	case captureClickerStop:
		a.cfg.Clicker.Stop = binding
	case captureClickerKey:
		a.cfg.Clicker.Key = binding
	case captureMenu:
		a.cfg.Menu.SetKeyByID(target.menuID, binding)
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
	case captureClickerStart:
		a.cfg.Clicker.Start = config.KeyBinding{}
	case captureClickerStop:
		a.cfg.Clicker.Stop = config.KeyBinding{}
	case captureClickerKey:
		a.cfg.Clicker.Key = config.KeyBinding{}
	case captureMenu:
		a.cfg.Menu.SetKeyByID(a.capture.menuID, config.KeyBinding{})
	}
}

func (a *application) updateBindingControl(target captureTarget) {
	switch target.kind {
	case captureStart:
		ignoreSetWindowText(a.controls.startButton, bindingText(a.cfg.Start))
	case captureStop:
		ignoreSetWindowText(a.controls.stopButton, bindingText(a.cfg.Stop))
	case capturePause:
		ignoreSetWindowText(a.controls.pauseButton, bindingText(a.cfg.Pause))
	case captureSkill:
		if target.index >= 0 && target.index < len(a.cfg.Skills) {
			ignoreSetWindowText(a.controls.skillButtons[target.index], bindingText(a.cfg.Skills[target.index].Key))
		}
	case captureClickerStart:
		ignoreSetWindowText(a.controls.clickerStartButton, bindingText(a.cfg.Clicker.Start))
	case captureClickerStop:
		ignoreSetWindowText(a.controls.clickerStopButton, bindingText(a.cfg.Clicker.Stop))
	case captureClickerKey:
		ignoreSetWindowText(a.controls.clickerKeyButton, bindingText(a.cfg.Clicker.Key))
	case captureMenu:
		if hwnd := a.controls.menuButtons[target.menuID]; hwnd != 0 {
			if b, ok := a.cfg.Menu.BindingByID(target.menuID); ok {
				ignoreSetWindowText(hwnd, bindingText(b))
			}
		}
	}
}

func (a *application) menuKeyMatches(vk uint16) bool {
	return a.cfg.Menu.Matches(vk)
}

func captureRejectsMouseLeft(kind captureKind) bool {
	return kind == captureStart || kind == captureStop || kind == captureClickerStart || kind == captureClickerStop
}

func captureRejectsOutputKey(kind captureKind, vk uint16) (bool, string) {
	if kind != captureSkill && kind != captureClickerKey {
		return false, ""
	}
	label, forbidden := config.ForbiddenOutputKeyLabel(int(vk))
	return forbidden, label
}

func (a *application) rejectCapturedKey(target captureTarget, keyName string) {
	message := captureRejectionMessage(target, keyName)
	a.setStatus(message)
	if a.hwnd != 0 {
		// Clear capture before the blocking modal so that keys pressed to dismiss
		// the dialog are not processed as new binding assignments while the
		// low-level hook keeps firing during the nested message loop.
		a.capture = captureTarget{}
		_ = messageBox(a.hwnd, "키 할당 불가", message, mbOK|mbIconWarning)
	}
}

func captureRejectionMessage(target captureTarget, keyName string) string {
	switch target.kind {
	case captureSkill:
		return "기술 키에는 " + keyName + "를 사용할 수 없습니다."
	case captureClickerKey:
		return "클릭 반복 출력 키에는 " + keyName + "를 사용할 수 없습니다."
	case captureStart, captureStop, captureClickerStart, captureClickerStop:
		return "시작/종료 키에는 " + keyName + "를 사용할 수 없습니다."
	default:
		return "이 용도에는 " + keyName + "를 사용할 수 없습니다."
	}
}

func sameKey(vk uint16, binding config.KeyBinding) bool {
	return binding.Assigned() && uint16(binding.VK) == vk
}

func bindingText(binding config.KeyBinding) string {
	if !binding.Assigned() {
		return "미지정"
	}
	return config.KeyDisplayName(binding.VK)
}

func parseInterval(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("실행 간격은 필수입니다")
	}
	if len(trimmed) > maxEditTextLen {
		return 0, fmt.Errorf("실행 간격 입력이 너무 깁니다")
	}
	interval, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("실행 간격은 숫자여야 합니다")
	}
	if interval < config.MinimumIntervalMS {
		return 0, fmt.Errorf("실행 간격은 최소 %dms 이상이어야 합니다", config.MinimumIntervalMS)
	}
	if interval > config.MaximumIntervalMS {
		return 0, fmt.Errorf("실행 간격은 최대 %dms 이하여야 합니다", config.MaximumIntervalMS)
	}
	if !config.MillisecondsFitDuration(interval) {
		return 0, fmt.Errorf("실행 간격이 너무 큽니다")
	}
	return interval, nil
}

func parseSkillGap(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return config.DefaultSkillGapMS, nil
	}
	if len(trimmed) > maxEditTextLen {
		return 0, fmt.Errorf("키별 간격 입력이 너무 깁니다")
	}
	gap, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("키별 간격은 숫자여야 합니다")
	}
	if gap < 0 {
		return 0, fmt.Errorf("키별 간격은 0ms 이상이어야 합니다")
	}
	if gap > config.MaximumSkillGapMS {
		return 0, fmt.Errorf("키별 간격은 최대 %dms 이하여야 합니다", config.MaximumSkillGapMS)
	}
	if !config.MillisecondsFitDuration(gap) {
		return 0, fmt.Errorf("키별 간격이 너무 큽니다")
	}
	return gap, nil
}

func parseInputHold(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("눌림 시간은 필수입니다")
	}
	if len(trimmed) > maxEditTextLen {
		return 0, fmt.Errorf("눌림 시간 입력이 너무 깁니다")
	}
	hold, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("눌림 시간은 숫자여야 합니다")
	}
	if hold < config.MinimumInputHoldMS {
		return 0, fmt.Errorf("눌림 시간은 최소 %dms 이상이어야 합니다", config.MinimumInputHoldMS)
	}
	if hold > config.MaximumInputHoldMS {
		return 0, fmt.Errorf("눌림 시간은 최대 %dms 이하여야 합니다", config.MaximumInputHoldMS)
	}
	if !config.MillisecondsFitDuration(hold) {
		return 0, fmt.Errorf("눌림 시간이 너무 큽니다")
	}
	return hold, nil
}
