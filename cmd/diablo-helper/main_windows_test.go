//go:build windows

package main

import (
	"strings"
	"testing"

	"github.com/dongju93/diablo-helper/internal/config"
)

func TestBindingText(t *testing.T) {
	tests := []struct {
		name    string
		binding config.KeyBinding
		want    string
	}{
		{name: "unassigned", binding: config.KeyBinding{}, want: "미지정"},
		{name: "uses saved name", binding: config.KeyBinding{Name: "Custom", VK: int('A')}, want: "Custom"},
		{name: "falls back to display name", binding: config.KeyBinding{VK: int('A')}, want: "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bindingText(tt.binding); got != tt.want {
				t.Fatalf("bindingText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseInterval(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      int
		wantError string
	}{
		{name: "valid", value: "25", want: 25},
		{name: "trims whitespace", value: " 30 \t", want: 30},
		{name: "empty", value: "", wantError: "필수"},
		{name: "not a number", value: "fast", wantError: "숫자"},
		{name: "below minimum", value: "9", wantError: "최소"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInterval(tt.value)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("parseInterval() error = %v", err)
				}
				if got != tt.want {
					t.Fatalf("parseInterval() = %d, want %d", got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatal("parseInterval() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("parseInterval() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestApplicationControlClassification(t *testing.T) {
	a := newApplication()

	if !a.isPrimaryButton(idSave) {
		t.Fatal("idSave should be primary")
	}
	if !a.isPrimaryButton(idApplyBulk) {
		t.Fatal("idApplyBulk should be primary")
	}
	if a.isPrimaryButton(idLoad) {
		t.Fatal("idLoad should not be primary")
	}

	for _, id := range []int{idStartKey, idStopKey, idPauseKey, idSkillKeyBase, idSkillKeyBase + config.MaxSkills - 1, idMenuInventory, idMenuWhisper} {
		if !a.isBindingButton(id) {
			t.Fatalf("id %d should be a binding button", id)
		}
	}
	for _, id := range []int{idSave, idApplyBulk, idSkillKeyBase + config.MaxSkills} {
		if a.isBindingButton(id) {
			t.Fatalf("id %d should not be a binding button", id)
		}
	}
}

func TestCaptureTargetAndControlID(t *testing.T) {
	a := newApplication()

	if (captureTarget{}).valid() {
		t.Fatal("zero captureTarget should be invalid")
	}
	if !(captureTarget{kind: captureStart}).valid() {
		t.Fatal("captureStart target should be valid")
	}

	tests := []struct {
		name   string
		target captureTarget
		want   int
	}{
		{name: "start", target: captureTarget{kind: captureStart}, want: idStartKey},
		{name: "stop", target: captureTarget{kind: captureStop}, want: idStopKey},
		{name: "pause", target: captureTarget{kind: capturePause}, want: idPauseKey},
		{name: "skill", target: captureTarget{kind: captureSkill, index: 3}, want: idSkillKeyBase + 3},
		{name: "skill below range", target: captureTarget{kind: captureSkill, index: -1}, want: 0},
		{name: "skill above range", target: captureTarget{kind: captureSkill, index: config.MaxSkills}, want: 0},
		{name: "menu", target: captureTarget{kind: captureMenu, menuID: "world_map"}, want: idMenuWorldMap},
		{name: "unknown menu", target: captureTarget{kind: captureMenu, menuID: "missing"}, want: 0},
		{name: "none", target: captureTarget{}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := a.captureControlID(tt.target); got != tt.want {
				t.Fatalf("captureControlID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHandleCommandStartsKeyCapture(t *testing.T) {
	tests := []struct {
		name string
		id   int
		want captureTarget
	}{
		{name: "start", id: idStartKey, want: captureTarget{kind: captureStart}},
		{name: "stop", id: idStopKey, want: captureTarget{kind: captureStop}},
		{name: "pause", id: idPauseKey, want: captureTarget{kind: capturePause}},
		{name: "skill", id: idSkillKeyBase + 2, want: captureTarget{kind: captureSkill, index: 2}},
		{name: "menu", id: idMenuWorldMap, want: captureTarget{kind: captureMenu, menuID: "world_map"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			if !a.handleCommand(makeLong(tt.id, bnClicked)) {
				t.Fatal("handleCommand() = false, want true")
			}
			if a.capture != tt.want {
				t.Fatalf("capture = %+v, want %+v", a.capture, tt.want)
			}
		})
	}
}

func TestHandleCommandIgnoresNonClickOrUnknownCommand(t *testing.T) {
	a := newApplication()
	if a.handleCommand(makeLong(idStartKey, 1)) {
		t.Fatal("handleCommand() = true for non-click notification")
	}
	if a.handleCommand(makeLong(9999, bnClicked)) {
		t.Fatal("handleCommand() = true for unknown command")
	}
	if a.capture.valid() {
		t.Fatalf("capture = %+v, want unchanged", a.capture)
	}
}

func TestSetMenuBindingAndMenuKeyMatches(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()

	a.setMenuBinding("world_map", config.KeyBinding{Name: "F7", VK: vkF1 + 6})
	if a.cfg.Menu.WorldMap != (config.KeyBinding{Name: "F7", VK: vkF1 + 6}) {
		t.Fatalf("world map = %+v, want F7", a.cfg.Menu.WorldMap)
	}
	if !a.menuKeyMatches(vkF1 + 6) {
		t.Fatal("menuKeyMatches(F7) = false, want true")
	}
	if a.menuKeyMatches(vkF1 + 7) {
		t.Fatal("menuKeyMatches(F8) = true, want false")
	}

	a.setMenuBinding("missing", config.KeyBinding{Name: "F8", VK: vkF1 + 7})
	if a.menuKeyMatches(vkF1 + 7) {
		t.Fatal("unknown menu id changed a binding")
	}
}

func TestSameKey(t *testing.T) {
	if !sameKey('A', config.KeyBinding{Name: "A", VK: int('A')}) {
		t.Fatal("sameKey() = false, want true")
	}
	if sameKey('A', config.KeyBinding{Name: "B", VK: int('B')}) {
		t.Fatal("sameKey() = true for different key")
	}
	if sameKey('A', config.KeyBinding{Name: "A", VK: 0}) {
		t.Fatal("sameKey() = true for unassigned binding")
	}
}

func TestHandleKeyEventAssignsCapturedKeys(t *testing.T) {
	a := newApplication()

	a.startCapture(captureTarget{kind: captureSkill, index: 0})
	if !a.handleKeyEvent('1', true) {
		t.Fatal("captured skill key was not consumed")
	}
	if a.cfg.Skills[0].Key != (config.KeyBinding{Name: "1", VK: int('1')}) {
		t.Fatalf("skill key = %+v, want 1", a.cfg.Skills[0].Key)
	}
	if a.capture.valid() {
		t.Fatalf("capture = %+v, want cleared", a.capture)
	}
	a.handleKeyEvent('1', false)

	a.startCapture(captureTarget{kind: captureMenu, menuID: "whisper"})
	if !a.handleKeyEvent(vkF1, true) {
		t.Fatal("captured menu key was not consumed")
	}
	if a.cfg.Menu.Whisper != (config.KeyBinding{Name: "F1", VK: vkF1}) {
		t.Fatalf("whisper = %+v, want F1", a.cfg.Menu.Whisper)
	}
}

func TestHandleKeyEventEscapeClearsCapturedKey(t *testing.T) {
	a := newApplication()
	a.cfg.Skills[0].Key = config.KeyBinding{Name: "1", VK: int('1')}
	a.startCapture(captureTarget{kind: captureSkill, index: 0})

	if !a.handleKeyEvent(vkEscape, true) {
		t.Fatal("escape during capture was not consumed")
	}
	if a.cfg.Skills[0].Key.Assigned() {
		t.Fatalf("skill key = %+v, want cleared", a.cfg.Skills[0].Key)
	}
	if a.capture.valid() {
		t.Fatalf("capture = %+v, want cleared", a.capture)
	}
}

func TestHandleKeyEventRejectsMouseLeftForStartAndStopCapture(t *testing.T) {
	tests := []struct {
		name string
		kind captureKind
	}{
		{name: "start", kind: captureStart},
		{name: "stop", kind: captureStop},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			a.startCapture(captureTarget{kind: tt.kind})

			if !a.handleKeyEvent(vkLButton, true) {
				t.Fatal("Mouse Left during start/stop capture was not consumed")
			}
			if a.cfg.Start.Assigned() || a.cfg.Stop.Assigned() {
				t.Fatalf("start/stop = %+v/%+v, want unassigned", a.cfg.Start, a.cfg.Stop)
			}
			if !a.capture.valid() {
				t.Fatal("capture should remain active after rejected Mouse Left")
			}
		})
	}
}

func TestHandleKeyEventSuppressesRepeatedKeyDownUntilKeyUp(t *testing.T) {
	a := newApplication()

	a.startCapture(captureTarget{kind: capturePause})
	if !a.handleKeyEvent('A', true) {
		t.Fatal("first A down should be consumed for capture")
	}

	a.startCapture(captureTarget{kind: captureStop})
	if a.handleKeyEvent('A', true) {
		t.Fatal("repeated A down should not be consumed")
	}
	if a.cfg.Stop.Assigned() {
		t.Fatalf("stop = %+v, want unassigned after repeated down", a.cfg.Stop)
	}

	a.handleKeyEvent('A', false)
	if !a.handleKeyEvent('A', true) {
		t.Fatal("A down after key up should be consumed")
	}
	if a.cfg.Stop != (config.KeyBinding{Name: "A", VK: int('A')}) {
		t.Fatalf("stop = %+v, want A", a.cfg.Stop)
	}
}

func TestHandleKeyEventPauseOnlyWhileHeld(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	if !a.runner.Start(a.cfg) {
		t.Fatal("runner did not start")
	}
	defer a.runner.Stop()

	a.handleKeyEvent(vkRButton, true)
	if !a.runner.Paused() {
		t.Fatal("runner paused = false, want true")
	}

	a.handleKeyEvent(vkRButton, false)
	if a.runner.Paused() {
		t.Fatal("runner paused = true after key up, want false")
	}
}

func TestHandleKeyEventStopsRunnerForStopAndMenuKeys(t *testing.T) {
	tests := []struct {
		name      string
		vk        uint16
		configure func(*config.Config)
	}{
		{
			name: "stop key",
			vk:   vkF1,
			configure: func(cfg *config.Config) {
				cfg.Stop = config.KeyBinding{Name: "F1", VK: vkF1}
			},
		},
		{
			name: "menu key",
			vk:   'C',
			configure: func(cfg *config.Config) {
				cfg.Menu.Inventory = config.KeyBinding{Name: "C", VK: int('C')}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			a.cfg = config.Default()
			tt.configure(&a.cfg)
			if !a.runner.Start(a.cfg) {
				t.Fatal("runner did not start")
			}

			a.handleKeyEvent(tt.vk, true)
			if a.runner.Running() {
				a.runner.Stop()
				t.Fatal("runner running = true, want stopped")
			}
		})
	}
}

func TestMouseEventKey(t *testing.T) {
	tests := []struct {
		name      string
		message   uintptr
		mouseData uint32
		wantVK    uint16
		wantDown  bool
		wantOK    bool
	}{
		{name: "left down", message: wmLButtonDown, wantVK: vkLButton, wantDown: true, wantOK: true},
		{name: "left up", message: wmLButtonUp, wantVK: vkLButton, wantDown: false, wantOK: true},
		{name: "right down", message: wmRButtonDown, wantVK: vkRButton, wantDown: true, wantOK: true},
		{name: "right up", message: wmRButtonUp, wantVK: vkRButton, wantDown: false, wantOK: true},
		{name: "middle down", message: wmMButtonDown, wantVK: vkMButton, wantDown: true, wantOK: true},
		{name: "middle up", message: wmMButtonUp, wantVK: vkMButton, wantDown: false, wantOK: true},
		{name: "x1 down", message: wmXButtonDown, mouseData: xButton1 << 16, wantVK: vkXButton1, wantDown: true, wantOK: true},
		{name: "x1 up", message: wmXButtonUp, mouseData: xButton1 << 16, wantVK: vkXButton1, wantDown: false, wantOK: true},
		{name: "x2 down", message: wmXButtonDown, mouseData: xButton2 << 16, wantVK: vkXButton2, wantDown: true, wantOK: true},
		{name: "unknown x button", message: wmXButtonDown, mouseData: 3 << 16, wantOK: false},
		{name: "unknown message", message: wmKeyDown, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVK, gotDown, gotOK := mouseEventKey(tt.message, &mouseHookStruct{MouseData: tt.mouseData})
			if gotVK != tt.wantVK || gotDown != tt.wantDown || gotOK != tt.wantOK {
				t.Fatalf("mouseEventKey() = (%d, %v, %v), want (%d, %v, %v)", gotVK, gotDown, gotOK, tt.wantVK, tt.wantDown, tt.wantOK)
			}
		})
	}
}
