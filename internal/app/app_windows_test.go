//go:build windows

package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/dongju93/diablo-helper/internal/config"
)

func TestMainWindowStyleAvoidsCompositedResizePath(t *testing.T) {
	if mainWindowExStyle&wsExComposited != 0 {
		t.Fatalf("mainWindowExStyle includes WS_EX_COMPOSITED: %#x", mainWindowExStyle)
	}
}

func TestApplicationRunCleansUpAfterGetMessageError(t *testing.T) {
	a := newApplication()
	a.runner = newSkillRunner(func(uint16) error { return nil })
	a.clicker = newClickerRunner(func(uint16) error { return nil })
	a.winapi = stubApplicationWinAPI()

	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:       "Skill 1",
		Key:        config.KeyBinding{Name: "A", VK: 0x41},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}
	if !a.runner.Start(cfg) {
		t.Fatal("runner.Start() = false, want true")
	}
	defer a.runner.Stop()
	if !a.clicker.Start(config.Clicker{Key: config.KeyBinding{Name: "A", VK: 0x41}, IntervalMS: config.MinimumIntervalMS}) {
		t.Fatal("clicker.Start() = false, want true")
	}
	defer a.clicker.Stop()

	var unhooked []uintptr
	a.winapi.unhookWindowsHook = func(hook uintptr) {
		unhooked = append(unhooked, hook)
	}
	getMessageErr := errors.New("forced GetMessageW failure")
	a.winapi.getMessage = func(*message) (int32, error) {
		return -1, getMessageErr
	}

	oldAppInstance := appInstance
	defer func() {
		appInstance = oldAppInstance
	}()

	err := a.run()
	if !errors.Is(err, getMessageErr) {
		t.Fatalf("run() error = %v, want wrapped GetMessageW error", err)
	}
	if !strings.Contains(err.Error(), "GetMessageW failed") {
		t.Fatalf("run() error = %v, want GetMessageW context", err)
	}
	if len(unhooked) != 2 || unhooked[0] != 11 || unhooked[1] != 22 {
		t.Fatalf("unhooked hooks = %v, want [11 22]", unhooked)
	}
	if a.hook != 0 || a.mouseHook != 0 {
		t.Fatalf("hooks after cleanup = (%d, %d), want zeros", a.hook, a.mouseHook)
	}
	if a.runner.Running() || a.clicker.Running() {
		t.Fatal("runners still running after cleanup")
	}
	if appInstance == a {
		t.Fatal("appInstance still points at cleaned up application")
	}
}

func TestApplicationCleanupIsIdempotent(t *testing.T) {
	a := newApplication()
	a.winapi = stubApplicationWinAPI()
	a.hook = 1
	a.mouseHook = 2

	var unhooked []uintptr
	a.winapi.unhookWindowsHook = func(hook uintptr) {
		unhooked = append(unhooked, hook)
	}

	oldAppInstance := appInstance
	appInstance = a
	defer func() {
		appInstance = oldAppInstance
	}()

	a.cleanup()
	a.cleanup()

	if len(unhooked) != 2 || unhooked[0] != 1 || unhooked[1] != 2 {
		t.Fatalf("unhooked hooks = %v, want [1 2]", unhooked)
	}
	if a.hook != 0 || a.mouseHook != 0 {
		t.Fatalf("hooks after cleanup = (%d, %d), want zeros", a.hook, a.mouseHook)
	}
	if appInstance == a {
		t.Fatal("appInstance still points at cleaned up application")
	}
}

func TestApplicationRunnerErrorPostsAndUpdatesStatus(t *testing.T) {
	a := newApplication()
	a.winapi = stubApplicationWinAPI()
	a.hwnd = 123

	var posted []uint32
	a.winapi.postMessage = func(hwnd uintptr, msg uint32, _ uintptr, _ uintptr) bool {
		if hwnd != a.hwnd {
			t.Fatalf("postMessage hwnd = %d, want %d", hwnd, a.hwnd)
		}
		posted = append(posted, msg)
		return true
	}

	a.handleRunnerError("기술 반복", errors.New("forced send failure"))
	if len(posted) != 1 || posted[0] != wmRunnerError {
		t.Fatalf("posted messages = %v, want [%d]", posted, wmRunnerError)
	}
	if a.statusText != "■ 정지." {
		t.Fatalf("status before message = %q, want unchanged", a.statusText)
	}

	a.showPendingRunnerError()
	if !strings.Contains(a.statusText, "입력 전송 실패") || !strings.Contains(a.statusText, "기술 반복") {
		t.Fatalf("status after message = %q, want runner error", a.statusText)
	}
}

func stubApplicationWinAPI() applicationWinAPI {
	return applicationWinAPI{
		getModuleHandle: func() (uintptr, error) {
			return 100, nil
		},
		loadCursor: func(uintptr, uintptr) uintptr {
			return 101
		},
		loadIcon: func(uintptr, uintptr) uintptr {
			return 102
		},
		registerClassEx: func(*windowClassEx) (uintptr, error) {
			return 1, nil
		},
		createWindowEx: func(
			uintptr,
			*uint16,
			*uint16,
			uintptr,
			uintptr,
			uintptr,
			uintptr,
			uintptr,
			uintptr,
			uintptr,
			uintptr,
			uintptr,
		) (uintptr, error) {
			return 200, nil
		},
		setWindowVisuals: func(uintptr) {},
		setWindowsHookEx: func(idHook int, _ uintptr, _ uintptr, _ uint32) (uintptr, error) {
			switch idHook {
			case whKeyboardLL:
				return 11, nil
			case whMouseLL:
				return 22, nil
			default:
				return 0, nil
			}
		},
		unhookWindowsHook: func(uintptr) {},
		showWindow:        func(uintptr, uintptr) {},
		updateWindow:      func(uintptr) {},
		postMessage:       func(uintptr, uint32, uintptr, uintptr) bool { return true },
		getMessage: func(*message) (int32, error) {
			return 0, nil
		},
		translateMessage: func(*message) {},
		dispatchMessage:  func(*message) {},
		destroyWindow:    func(uintptr) {},
		postQuitMessage:  func(int) {},
	}
}
