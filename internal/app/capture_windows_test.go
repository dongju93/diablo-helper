//go:build windows

package app

import (
	"strconv"
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
		{name: "uses canonical name ignoring stored name", binding: config.KeyBinding{Name: "Spoofed", VK: int('A')}, want: "A"},
		{name: "canonical name from VK", binding: config.KeyBinding{VK: int('A')}, want: "A"},
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
		{name: "above maximum", value: strconv.Itoa(config.MaximumIntervalMS + 1), wantError: "최대"},
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

func TestParseSkillGap(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      int
		wantError string
	}{
		{name: "default empty", value: "", want: config.DefaultSkillGapMS},
		{name: "zero", value: "0", want: 0},
		{name: "valid", value: "35", want: 35},
		{name: "trims whitespace", value: " 40 \t", want: 40},
		{name: "not a number", value: "slow", wantError: "숫자"},
		{name: "below zero", value: "-1", wantError: "0ms 이상"},
		{name: "above maximum", value: strconv.Itoa(config.MaximumSkillGapMS + 1), wantError: "최대"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSkillGap(tt.value)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("parseSkillGap() error = %v", err)
				}
				if got != tt.want {
					t.Fatalf("parseSkillGap() = %d, want %d", got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatal("parseSkillGap() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("parseSkillGap() error = %v, want %q", err, tt.wantError)
			}
		})
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
		{name: "clicker start", target: captureTarget{kind: captureClickerStart}, want: idClickerStartKey},
		{name: "clicker stop", target: captureTarget{kind: captureClickerStop}, want: idClickerStopKey},
		{name: "clicker key", target: captureTarget{kind: captureClickerKey}, want: idClickerKey},
		{name: "skill", target: captureTarget{kind: captureSkill, index: 3}, want: idSkillKeyBase + 3},
		{name: "skill below range", target: captureTarget{kind: captureSkill, index: -1}, want: 0},
		{name: "skill above range", target: captureTarget{kind: captureSkill, index: config.MaxSkills}, want: 0},
		{
			name:   "menu",
			target: captureTarget{kind: captureMenu, menuID: "town_portal"},
			want:   mustMenuControl(t, "town_portal").control,
		},
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

func TestSetMenuBindingAndMenuKeyMatches(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()

	a.cfg.Menu.SetKeyByID("clan", config.KeyBinding{Name: "F7", VK: vkF1 + 6})
	if a.cfg.Menu.Clan != (config.KeyBinding{Name: "F7", VK: vkF1 + 6}) {
		t.Fatalf("clan = %+v, want F7", a.cfg.Menu.Clan)
	}
	if !a.menuKeyMatches(vkF1 + 6) {
		t.Fatal("menuKeyMatches(F7) = false, want true")
	}
	if a.menuKeyMatches(vkF1 + 7) {
		t.Fatal("menuKeyMatches(F8) = true, want false")
	}

	a.cfg.Menu.SetKeyByID("missing", config.KeyBinding{Name: "F8", VK: vkF1 + 7})
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

	a.startCapture(captureTarget{kind: captureMenu, menuID: "social"})
	if !a.handleKeyEvent(vkF1, true) {
		t.Fatal("captured menu key was not consumed")
	}
	if a.cfg.Menu.Social != (config.KeyBinding{Name: "F1", VK: vkF1}) {
		t.Fatalf("social = %+v, want F1", a.cfg.Menu.Social)
	}

	a.startCapture(captureTarget{kind: captureClickerKey})
	if !a.handleKeyEvent(vkLButton, true) {
		t.Fatal("captured clicker key was not consumed")
	}
	if a.cfg.Clicker.Key != (config.KeyBinding{Name: "Mouse Left", VK: vkLButton}) {
		t.Fatalf("clicker key = %+v, want Mouse Left", a.cfg.Clicker.Key)
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

func TestHandleKeyEventRejectsForbiddenOutputKeysDuringCapture(t *testing.T) {
	tests := []struct {
		name     string
		target   captureTarget
		vk       uint16
		keyName  string
		existing func(*application) config.KeyBinding
	}{
		{
			name:    "clicker num lock",
			target:  captureTarget{kind: captureClickerKey},
			vk:      0x90,
			keyName: "Num Lock",
			existing: func(a *application) config.KeyBinding {
				a.cfg.Clicker.Key = config.KeyBinding{Name: "Mouse Left", VK: vkLButton}
				return a.cfg.Clicker.Key
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			want := tt.existing(a)
			a.startCapture(tt.target)

			if !a.handleKeyEvent(tt.vk, true) {
				t.Fatal("forbidden output key during capture was not consumed")
			}
			switch tt.target.kind {
			case captureSkill:
				if a.cfg.Skills[tt.target.index].Key != want {
					t.Fatalf("skill key = %+v, want preserved %+v", a.cfg.Skills[tt.target.index].Key, want)
				}
			case captureClickerKey:
				if a.cfg.Clicker.Key != want {
					t.Fatalf("clicker key = %+v, want preserved %+v", a.cfg.Clicker.Key, want)
				}
			}
			if !a.capture.valid() {
				t.Fatal("capture should remain active after forbidden output key")
			}
			if !strings.Contains(a.statusText, tt.keyName) || !strings.Contains(a.statusText, "사용할 수 없습니다") {
				t.Fatalf("status = %q, want rejection mentioning %s", a.statusText, tt.keyName)
			}
		})
	}
}

func TestHandleKeyEventRejectsMouseLeftForStartAndStopCapture(t *testing.T) {
	tests := []struct {
		name string
		kind captureKind
	}{
		{name: "start", kind: captureStart},
		{name: "stop", kind: captureStop},
		{name: "clicker start", kind: captureClickerStart},
		{name: "clicker stop", kind: captureClickerStop},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			a.startCapture(captureTarget{kind: tt.kind})

			if !a.handleKeyEvent(vkLButton, true) {
				t.Fatal("Mouse Left during start/stop capture was not consumed")
			}
			if a.cfg.Start.Assigned() || a.cfg.Stop.Assigned() || a.cfg.Clicker.Start.Assigned() || a.cfg.Clicker.Stop.Assigned() {
				t.Fatalf("start/stop = %+v/%+v clicker %+v/%+v, want unassigned", a.cfg.Start, a.cfg.Stop, a.cfg.Clicker.Start, a.cfg.Clicker.Stop)
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

func TestHandleKeyEventDoesNotRememberUnconfiguredKeys(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()

	if a.handleKeyEvent('Z', true) {
		t.Fatal("unconfigured Z down should not be consumed")
	}
	if a.pressed.any() {
		t.Fatal("unconfigured Z down was retained in pressed state")
	}
}

func TestHandleKeyEventTracksConfiguredKeysForRepeatSuppression(t *testing.T) {
	a := newApplication()
	a.startCapture(captureTarget{kind: capturePause})

	if !a.handleKeyEvent(vkF1, true) {
		t.Fatal("captured F1 down should be consumed")
	}
	if !a.pressed.has(vkF1) {
		t.Fatal("captured F1 was not retained for repeat suppression")
	}

	a.handleKeyEvent(vkF1, false)
	if a.pressed.has(vkF1) {
		t.Fatal("captured F1 remained pressed after key up")
	}
}

func TestHandleKeyEventIgnoresStopPauseAndMenuBeforeRunnersStart(t *testing.T) {
	tests := []struct {
		name string
		vk   uint16
	}{
		{name: "stop key", vk: vkF1 + 1},
		{name: "pause key", vk: vkF1 + 2},
		{name: "menu key", vk: vkF1 + 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			a.cfg = config.Default()
			a.cfg.Start = config.KeyBinding{Name: "F1", VK: vkF1}
			a.cfg.Stop = config.KeyBinding{Name: "F2", VK: vkF1 + 1}
			a.cfg.Pause = config.KeyBinding{Name: "F3", VK: vkF1 + 2}
			a.cfg.Menu.Character = config.KeyBinding{Name: "F4", VK: vkF1 + 3}

			if a.handleKeyEvent(tt.vk, true) {
				t.Fatal("idle non-start hotkey down should not be consumed")
			}
			if a.pressed.has(tt.vk) {
				t.Fatal("idle non-start hotkey down was retained in pressed state")
			}
		})
	}
}

func TestShouldTrackPressedKeyUsesRuntimeRelevantKeys(t *testing.T) {
	const (
		startVK        = vkF1
		stopVK         = vkF1 + 1
		pauseVK        = vkF1 + 2
		clickerStartVK = vkF1 + 3
		clickerStopVK  = vkF1 + 4
		menuVK         = vkF1 + 5
	)

	tests := []struct {
		name           string
		startRunner    bool
		startClicker   bool
		wantStart      bool
		wantStop       bool
		wantPause      bool
		wantClickStart bool
		wantClickStop  bool
		wantMenu       bool
	}{
		{
			name:           "idle tracks skill and clicker start keys",
			wantStart:      true,
			wantClickStart: true,
		},
		{
			name:           "skill runner tracks skill controls and idle clicker start",
			startRunner:    true,
			wantStop:       true,
			wantPause:      true,
			wantClickStart: true,
			wantMenu:       true,
		},
		{
			name:          "clicker tracks clicker controls and idle skill start",
			startClicker:  true,
			wantStart:     true,
			wantPause:     true,
			wantClickStop: true,
			wantMenu:      true,
		},
		{
			name:          "both runners track only runtime controls",
			startRunner:   true,
			startClicker:  true,
			wantStop:      true,
			wantPause:     true,
			wantClickStop: true,
			wantMenu:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			a.cfg = config.Default()
			enableRunnableTestSkill(&a.cfg)
			a.cfg.Start = config.KeyBinding{Name: "F1", VK: startVK}
			a.cfg.Stop = config.KeyBinding{Name: "F2", VK: stopVK}
			a.cfg.Pause = config.KeyBinding{Name: "F3", VK: pauseVK}
			a.cfg.Clicker.Start = config.KeyBinding{Name: "F4", VK: clickerStartVK}
			a.cfg.Clicker.Stop = config.KeyBinding{Name: "F5", VK: clickerStopVK}
			a.cfg.Menu.Character = config.KeyBinding{Name: "F6", VK: menuVK}

			if tt.startRunner {
				if !a.runner.Start(a.cfg) {
					t.Fatal("runner did not start")
				}
				defer a.runner.Stop()
			}
			if tt.startClicker {
				if !a.clicker.Start(a.cfg.Clicker) {
					t.Fatal("clicker did not start")
				}
				defer a.clicker.Stop()
			}

			assertTrackPressedKey(t, a, "start", startVK, tt.wantStart)
			assertTrackPressedKey(t, a, "stop", stopVK, tt.wantStop)
			assertTrackPressedKey(t, a, "pause", pauseVK, tt.wantPause)
			assertTrackPressedKey(t, a, "clicker start", clickerStartVK, tt.wantClickStart)
			assertTrackPressedKey(t, a, "clicker stop", clickerStopVK, tt.wantClickStop)
			assertTrackPressedKey(t, a, "menu", menuVK, tt.wantMenu)
		})
	}
}

func assertTrackPressedKey(t *testing.T, a *application, name string, vk uint16, want bool) {
	t.Helper()
	if got := a.shouldTrackPressedKey(vk); got != want {
		t.Fatalf("shouldTrackPressedKey(%s) = %v, want %v", name, got, want)
	}
}

func TestHandleKeyEventPauseOnlyWhileHeld(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	enableRunnableTestSkill(&a.cfg)
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

// TestHandleKeyEventPauseWhenPauseKeyEqualsStartKey verifies that when the
// pause key is configured to the same key as the start key, pressing that key
// while the runner is already running pauses the runner. This is the real-world
// case where the "started" early-return must not swallow the pause action.
func TestHandleKeyEventPauseWhenPauseKeyEqualsStartKey(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	enableRunnableTestSkill(&a.cfg)
	// Assign the same key to both Start and Pause.
	sharedKey := config.KeyBinding{Name: "F5", VK: vkF1 + 4}
	a.cfg.Start = sharedKey
	a.cfg.Pause = sharedKey

	if !a.runner.Start(a.cfg) {
		t.Fatal("runner did not start")
	}
	defer a.runner.Stop()

	// Runner is already running. Pressing the shared key must pause, not start.
	a.handleKeyEvent(vkF1+4, true)
	if !a.runner.Paused() {
		t.Fatal("runner paused = false after pressing shared start/pause key while running, want true")
	}

	a.handleKeyEvent(vkF1+4, false)
	if a.runner.Paused() {
		t.Fatal("runner paused = true after key up, want false")
	}
}

// TestHandleKeyEventPauseWhenPauseKeyEqualsClickerStartKey verifies that when
// the pause key equals the clicker start key, pressing it while running still
// pauses the skill runner.
func TestHandleKeyEventPauseWhenPauseKeyEqualsClickerStartKey(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	enableRunnableTestSkill(&a.cfg)
	sharedKey := config.KeyBinding{Name: "F6", VK: vkF1 + 5}
	a.cfg.Clicker.Start = sharedKey
	a.cfg.Pause = sharedKey

	if !a.runner.Start(a.cfg) {
		t.Fatal("runner did not start")
	}
	defer a.runner.Stop()

	a.handleKeyEvent(vkF1+5, true)
	if !a.runner.Paused() {
		t.Fatal("runner paused = false after pressing shared clicker-start/pause key while running, want true")
	}

	a.handleKeyEvent(vkF1+5, false)
	if a.runner.Paused() {
		t.Fatal("runner paused = true after key up, want false")
	}
}

func TestHandleKeyEventMenuKeyWinsOverStartKeyWhileRunnerRunning(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	enableRunnableTestSkill(&a.cfg)
	sharedKey := config.KeyBinding{Name: "F7", VK: vkF1 + 6}
	a.cfg.Start = sharedKey
	a.cfg.Menu.Character = sharedKey

	if !a.runner.Start(a.cfg) {
		t.Fatal("runner did not start")
	}

	a.handleKeyEvent(vkF1+6, true)
	if a.runner.Running() {
		a.runner.Stop()
		t.Fatal("runner running = true after shared start/menu key, want stopped")
	}
}

// TestHandleKeyEventPauseDoesNotTriggerForUnassignedPauseKey verifies that
// pressing any key does not pause the runner when no pause key is configured.
// TestHandleKeyEventPauseClickerOnlyWhileHeld verifies that the clicker is
// paused while the pause key is held and resumes when released, even when the
// skill runner is not running.
func TestHandleKeyEventPauseClickerOnlyWhileHeld(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	if !a.clicker.Start(a.cfg.Clicker) {
		t.Fatal("clicker did not start")
	}
	defer a.clicker.Stop()

	a.handleKeyEvent(vkRButton, true)
	if !a.clicker.Paused() {
		t.Fatal("clicker paused = false after pause key down, want true")
	}

	a.handleKeyEvent(vkRButton, false)
	if a.clicker.Paused() {
		t.Fatal("clicker paused = true after pause key up, want false")
	}
}

// TestHandleKeyEventPauseBothRunnersWhileHeld verifies that both the skill
// runner and clicker are paused simultaneously when the pause key is held.
func TestHandleKeyEventPauseBothRunnersWhileHeld(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	enableRunnableTestSkill(&a.cfg)
	if !a.runner.Start(a.cfg) {
		t.Fatal("runner did not start")
	}
	defer a.runner.Stop()
	if !a.clicker.Start(a.cfg.Clicker) {
		t.Fatal("clicker did not start")
	}
	defer a.clicker.Stop()

	a.handleKeyEvent(vkRButton, true)
	if !a.runner.Paused() {
		t.Fatal("runner paused = false after pause key down, want true")
	}
	if !a.clicker.Paused() {
		t.Fatal("clicker paused = false after pause key down, want true")
	}

	a.handleKeyEvent(vkRButton, false)
	if a.runner.Paused() {
		t.Fatal("runner paused = true after pause key up, want false")
	}
	if a.clicker.Paused() {
		t.Fatal("clicker paused = true after pause key up, want false")
	}
}

func TestHandleKeyEventPauseDoesNotTriggerForUnassignedPauseKey(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	enableRunnableTestSkill(&a.cfg)
	a.cfg.Pause = config.KeyBinding{} // explicitly clear pause binding

	if !a.runner.Start(a.cfg) {
		t.Fatal("runner did not start")
	}
	defer a.runner.Stop()

	a.handleKeyEvent(vkRButton, true)
	if a.runner.Paused() {
		t.Fatal("runner paused = true with unassigned pause key, want false")
	}
	a.handleKeyEvent(vkRButton, false)
}

// TestHandleKeyEventPauseNotTriggeredByDifferentKey verifies that pressing a
// key that is not the configured pause key does not pause the runner.
func TestHandleKeyEventPauseNotTriggeredByDifferentKey(t *testing.T) {
	a := newApplication()
	a.cfg = config.Default()
	enableRunnableTestSkill(&a.cfg)
	// Pause is Mouse Right (vkRButton). Press a different key.
	a.cfg.Pause = config.KeyBinding{Name: "Mouse Right", VK: vkRButton}

	if !a.runner.Start(a.cfg) {
		t.Fatal("runner did not start")
	}
	defer a.runner.Stop()

	a.handleKeyEvent(vkF1, true)
	if a.runner.Paused() {
		t.Fatal("runner paused = true after pressing non-pause key, want false")
	}
	a.handleKeyEvent(vkF1, false)
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
				cfg.Menu.Character = config.KeyBinding{Name: "C", VK: int('C')}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			a.cfg = config.Default()
			enableRunnableTestSkill(&a.cfg)
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

func TestHandleKeyEventStopsClickerForClickerStopAndMenuKeys(t *testing.T) {
	tests := []struct {
		name      string
		vk        uint16
		configure func(*config.Config)
	}{
		{
			name: "clicker stop key",
			vk:   vkF1,
			configure: func(cfg *config.Config) {
				cfg.Clicker.Stop = config.KeyBinding{Name: "F1", VK: vkF1}
			},
		},
		{
			name: "menu key",
			vk:   'C',
			configure: func(cfg *config.Config) {
				cfg.Menu.Character = config.KeyBinding{Name: "C", VK: int('C')}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			a.cfg = config.Default()
			tt.configure(&a.cfg)
			if !a.clicker.Start(a.cfg.Clicker) {
				t.Fatal("clicker did not start")
			}

			a.handleKeyEvent(tt.vk, true)
			if a.clicker.Running() {
				a.clicker.Stop()
				t.Fatal("clicker running = true, want stopped")
			}
		})
	}
}

func enableRunnableTestSkill(cfg *config.Config) {
	cfg.Skills[0] = config.Skill{
		Name:       "Enabled",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}
}
