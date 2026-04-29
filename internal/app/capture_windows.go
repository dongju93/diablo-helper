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
	if down {
		if a.pressed.has(vk) {
			return false
		}
		if !a.shouldTrackPressedKey(vk) {
			return false
		}
		a.pressed.set(vk)
	} else {
		a.pressed.clear(vk)
		if sameKey(vk, a.cfg.Pause) {
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
		if vk == vkLButton && captureRejectsMouseLeft(a.capture.kind) {
			a.setStatus("시작/종료 키에는 Mouse Left를 사용할 수 없습니다.")
			return true
		}
		a.assignCapturedKey(vk)
		return true
	}

	stopped := false
	if a.runner.Running() && sameKey(vk, a.cfg.Stop) {
		stopped = a.runner.Stop()
	}
	if a.clicker.Running() && sameKey(vk, a.cfg.Clicker.Stop) {
		stopped = a.clicker.Stop() || stopped
	}
	if stopped {
		a.setStatus("종료 키 입력으로 정지했습니다.")
		return false
	}

	if sameKey(vk, a.cfg.Pause) && (a.runner.Running() || a.clicker.Running()) {
		a.runner.SetPaused(true)
		a.clicker.SetPaused(true)
		a.updateRuntimeStatus()
		return false
	}

	started := false
	if sameKey(vk, a.cfg.Start) {
		a.startRunnerFromHotkey()
		started = true
	}
	if sameKey(vk, a.cfg.Clicker.Start) {
		a.startClickerFromHotkey()
		started = true
	}
	if started {
		return false
	}

	if a.menuKeyMatches(vk) {
		a.stopAllRunners("게임 메뉴 키 입력으로 정지했습니다.")
	}
	return false
}

func (a *application) shouldTrackPressedKey(vk uint16) bool {
	// Track configured control keys even when they are idle right now. If the
	// runner starts while one is still held, Windows auto-repeat must not turn
	// that held key into a fresh stop/pause/menu press.
	if a.capture.valid() {
		return true
	}
	if sameKey(vk, a.cfg.Start) {
		return true
	}
	if sameKey(vk, a.cfg.Stop) {
		return true
	}
	if sameKey(vk, a.cfg.Pause) {
		return true
	}
	if sameKey(vk, a.cfg.Clicker.Start) {
		return true
	}
	if sameKey(vk, a.cfg.Clicker.Stop) {
		return true
	}
	return a.menuKeyMatches(vk)
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
		setWindowText(a.controls.startButton, bindingText(a.cfg.Start))
	case captureStop:
		setWindowText(a.controls.stopButton, bindingText(a.cfg.Stop))
	case capturePause:
		setWindowText(a.controls.pauseButton, bindingText(a.cfg.Pause))
	case captureSkill:
		if target.index >= 0 && target.index < len(a.cfg.Skills) {
			setWindowText(a.controls.skillButtons[target.index], bindingText(a.cfg.Skills[target.index].Key))
		}
	case captureClickerStart:
		setWindowText(a.controls.clickerStartButton, bindingText(a.cfg.Clicker.Start))
	case captureClickerStop:
		setWindowText(a.controls.clickerStopButton, bindingText(a.cfg.Clicker.Stop))
	case captureClickerKey:
		setWindowText(a.controls.clickerKeyButton, bindingText(a.cfg.Clicker.Key))
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

func (a *application) menuKeyMatches(vk uint16) bool {
	for _, menu := range a.cfg.MenuBindings() {
		if sameKey(vk, menu.Binding) {
			return true
		}
	}
	return false
}

func captureRejectsMouseLeft(kind captureKind) bool {
	return kind == captureStart || kind == captureStop || kind == captureClickerStart || kind == captureClickerStop
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
