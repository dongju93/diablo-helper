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
)

type captureTarget struct {
	kind   captureKind
	index  int
	menuID string
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
