//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/dongju93/diablo-helper/internal/config"
	"github.com/dongju93/diablo-helper/internal/meta"
)

type application struct {
	hwnd               uintptr
	instance           uintptr
	hook               uintptr
	mouseHook          uintptr
	font               uintptr
	titleFont          uintptr
	sectionFont        uintptr
	bgBrush            uintptr
	panelBrush         uintptr
	editBrush          uintptr
	borderPen          uintptr
	borderStrongPen    uintptr
	borderBrush        uintptr
	accentBrush        uintptr
	accentPen          uintptr
	fontScale          float64
	dpi                int
	configPath         string
	cfg                config.Config
	controls           controlRefs
	capture            captureTarget
	pressed            pressedKeys
	runner             *skillRunner
	clicker            *clickerRunner
	statusText         string
	runnerErrorMu      sync.Mutex
	pendingRunnerError string
	skillEnabled       [config.MaxSkills]bool
	winapi             applicationWinAPI
	cleanedUp          bool
}

var (
	appInstance      *application
	windowProc       = syscall.NewCallback(wndProc)
	keyboardHookProc = syscall.NewCallback(lowLevelKeyboardProc)
	mouseHookProc    = syscall.NewCallback(lowLevelMouseProc)
)

const defaultConfigFileName = "default.toml"

func Run() error {
	runtime.LockOSThread()
	if err := ensureWinAPI(); err != nil {
		return err
	}
	if err := hardenDLLSearchPath(); err != nil {
		return err
	}

	app := newApplication()
	return app.run()
}

func ShowFatalError(err error) {
	if err != nil {
		if initErr := ensureWinAPI(); initErr == nil {
			_ = messageBox(0, "diablo-helper", err.Error(), mbOK|mbIconError)
			return
		}
		_ = fallbackMessageBox("diablo-helper", err.Error(), mbOK|mbIconError)
	}
}

func fallbackMessageBox(title string, text string, flags uintptr) error {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = os.Getenv("windir")
	}
	if root == "" {
		return fmt.Errorf("SystemRoot is not set")
	}

	textPtr, err := utf16PtrSafe(text)
	if err != nil {
		return err
	}
	titlePtr, err := utf16PtrSafe(title)
	if err != nil {
		return err
	}
	user32Path := filepath.Join(root, "System32", "user32.dll")
	if !filepath.IsAbs(user32Path) {
		return fmt.Errorf("SystemRoot does not resolve to an absolute path")
	}
	proc := syscall.NewLazyDLL(user32Path).NewProc("MessageBoxW")
	ret, _, callErr := proc.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		flags,
	)
	if ret == 0 && callErr != syscall.Errno(0) {
		return callErr
	}
	return nil
}

func newApplication() *application {
	_ = ensureWinAPI()
	input := newSerializedContextInputSender(sendVirtualKeyContext, releaseVirtualKey)
	a := &application{
		cfg:        config.Default(),
		runner:     newSkillRunnerWithContextTimedSend(input.SendContext, input.Release),
		statusText: "■ 정지.",
		controls: controlRefs{
			menuLabels:  make(map[string]uintptr),
			menuButtons: make(map[string]uintptr),
		},
		clicker: newClickerRunnerWithContextTimedSend(input.SendContext, input.Release),
		winapi:  defaultApplicationWinAPI(),
	}
	a.runner.SetErrorHandler(func(err error) {
		a.handleRunnerError("기술 반복", err)
	})
	a.clicker.SetErrorHandler(func(err error) {
		a.handleRunnerError("클릭 반복", err)
	})
	return a
}

func (a *application) handleRunnerError(name string, err error) {
	if a == nil || err == nil {
		return
	}
	status := fmt.Sprintf("입력 전송 실패로 %s을 정지했습니다: %v", name, err)
	a.runnerErrorMu.Lock()
	a.pendingRunnerError = status
	a.runnerErrorMu.Unlock()
	if a.hwnd != 0 && a.winapi.postMessage != nil {
		a.winapi.postMessage(a.hwnd, wmRunnerError, 0, 0)
	}
}

func (a *application) showPendingRunnerError() {
	a.runnerErrorMu.Lock()
	status := a.pendingRunnerError
	a.pendingRunnerError = ""
	a.runnerErrorMu.Unlock()
	if status == "" {
		a.updateRuntimeStatus()
		return
	}
	a.setStatus(status)
}

func (a *application) run() error {
	a.configPath = defaultConfigPath()
	if loaded, err := config.LoadFile(a.configPath); err == nil {
		a.cfg = loaded
	} else if !errors.Is(err, os.ErrNotExist) {
		messageBox(0, "diablo-helper", "Failed to load "+defaultConfigFileName+". Defaults will be used.\n\n"+err.Error(), mbOK|mbIconError)
	}
	a.cfg.NormalizeForUI()

	instance, err := a.winapi.getModuleHandle()
	if instance == 0 {
		return fmt.Errorf("GetModuleHandleW failed: %w", err)
	}
	a.instance = instance

	if err := a.registerWindowClass(); err != nil {
		return err
	}

	appInstance = a
	bounds := a.currentWindowBounds(0)
	hwnd, err := a.winapi.createWindowEx(
		mainWindowExStyle,
		utf16Ptr("DiabloHelperWindow"),
		utf16Ptr(meta.Title()),
		mainWindowStyle,
		cwUseDefault,
		cwUseDefault,
		uintptr(bounds.maxW),
		uintptr(bounds.maxH),
		0,
		0,
		a.instance,
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", err)
	}
	a.hwnd = hwnd
	a.winapi.setWindowVisuals(hwnd)

	hook, err := a.winapi.setWindowsHookEx(whKeyboardLL, keyboardHookProc, a.instance, 0)
	if hook == 0 {
		return fmt.Errorf("SetWindowsHookExW failed: %w", err)
	}
	a.hook = hook
	defer a.cleanup()

	mouseHook, err := a.winapi.setWindowsHookEx(whMouseLL, mouseHookProc, a.instance, 0)
	if mouseHook == 0 {
		return fmt.Errorf("SetWindowsHookExW mouse hook failed: %w", err)
	}
	a.mouseHook = mouseHook

	a.winapi.showWindow(hwnd, swShow)
	a.winapi.updateWindow(hwnd)

	var msg message
	for {
		ret, callErr := a.winapi.getMessage(&msg)
		switch ret {
		case -1:
			return fmt.Errorf("GetMessageW failed: %w", callErr)
		case 0:
			return nil
		default:
			a.winapi.translateMessage(&msg)
			a.winapi.dispatchMessage(&msg)
		}
	}
}

func (a *application) registerWindowClass() error {
	cursor := a.winapi.loadCursor(0, idcArrow)
	icon := a.winapi.loadIcon(a.instance, uintptr(1))
	className := utf16Ptr("DiabloHelperWindow")
	wc := windowClassEx{
		Size:       uint32(unsafe.Sizeof(windowClassEx{})),
		WndProc:    windowProc,
		Instance:   a.instance,
		Icon:       icon,
		IconSm:     icon,
		Cursor:     cursor,
		Background: colorWindow + 1,
		ClassName:  className,
	}
	ret, err := a.winapi.registerClassEx(&wc)
	if ret == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", err)
	}
	return nil
}

func (a *application) cleanup() {
	if a.cleanedUp {
		return
	}
	a.cleanedUp = true

	if appInstance == a {
		appInstance = nil
	}
	if a.hook != 0 {
		a.winapi.unhookWindowsHook(a.hook)
		a.hook = 0
	}
	if a.mouseHook != 0 {
		a.winapi.unhookWindowsHook(a.mouseHook)
		a.mouseHook = 0
	}
	stopRuntimeRunners(a.runner, a.clicker)
	a.disposeUIResources()
}

func (a *application) handleDPIChanged(hwnd uintptr, wParam uintptr, lParam unsafe.Pointer) {
	if a == nil {
		return
	}
	dpi := dpiFromWParam(wParam)
	if dpi <= 0 {
		dpi = getWindowDPI(hwnd)
	}
	a.dpi = dpi

	if lParam != nil {
		suggested := (*rect)(lParam)
		setWindowPos(
			hwnd,
			int(suggested.Left),
			int(suggested.Top),
			int(suggested.Right-suggested.Left),
			int(suggested.Bottom-suggested.Top),
			swpNoZOrder|swpNoActivate,
		)
	}
	if a.controls.status != 0 {
		a.repositionControls()
	}
	if hwnd != 0 {
		invalidateRect(hwnd, true)
	}
}

func dpiFromWParam(wParam uintptr) int {
	dpi := lowWord(wParam)
	if dpi == 0 {
		dpi = highWord(wParam)
	}
	return dpi
}

func defaultConfigPath() string {
	executable, err := os.Executable()
	if err != nil {
		return defaultConfigFileName
	}
	return filepath.Join(filepath.Dir(executable), defaultConfigFileName)
}

type applicationWinAPI struct {
	getModuleHandle   func() (uintptr, error)
	loadCursor        func(instance uintptr, cursor uintptr) uintptr
	loadIcon          func(instance uintptr, icon uintptr) uintptr
	registerClassEx   func(wc *windowClassEx) (uintptr, error)
	createWindowEx    createWindowExFunc
	setWindowVisuals  func(hwnd uintptr)
	setWindowsHookEx  func(idHook int, hookProc uintptr, instance uintptr, threadID uint32) (uintptr, error)
	unhookWindowsHook func(hook uintptr)
	showWindow        func(hwnd uintptr, command uintptr)
	updateWindow      func(hwnd uintptr)
	postMessage       func(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) bool
	getMessage        func(msg *message) (int32, error)
	translateMessage  func(msg *message)
	dispatchMessage   func(msg *message)
	destroyWindow     func(hwnd uintptr)
	postQuitMessage   func(exitCode int)
	monitorMetrics    func(hwnd uintptr) monitorMetrics
}

type createWindowExFunc func(
	exStyle uintptr,
	className *uint16,
	windowName *uint16,
	style uintptr,
	x uintptr,
	y uintptr,
	width uintptr,
	height uintptr,
	parent uintptr,
	menu uintptr,
	instance uintptr,
	param uintptr,
) (uintptr, error)

func defaultApplicationWinAPI() applicationWinAPI {
	return applicationWinAPI{
		getModuleHandle: func() (uintptr, error) {
			instance, _, err := procGetModuleHandleW.Call(0)
			return instance, err
		},
		loadCursor: func(instance uintptr, cursor uintptr) uintptr {
			ret, _, _ := procLoadCursorW.Call(instance, cursor)
			return ret
		},
		loadIcon: func(instance uintptr, icon uintptr) uintptr {
			ret, _, _ := procLoadIconW.Call(instance, icon)
			return ret
		},
		registerClassEx: func(wc *windowClassEx) (uintptr, error) {
			ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
			return ret, err
		},
		createWindowEx: func(
			exStyle uintptr,
			className *uint16,
			windowName *uint16,
			style uintptr,
			x uintptr,
			y uintptr,
			width uintptr,
			height uintptr,
			parent uintptr,
			menu uintptr,
			instance uintptr,
			param uintptr,
		) (uintptr, error) {
			hwnd, _, err := procCreateWindowExW.Call(
				exStyle,
				uintptr(unsafe.Pointer(className)),
				uintptr(unsafe.Pointer(windowName)),
				style,
				x,
				y,
				width,
				height,
				parent,
				menu,
				instance,
				param,
			)
			return hwnd, err
		},
		setWindowVisuals: setWindowVisuals,
		setWindowsHookEx: func(idHook int, hookProc uintptr, instance uintptr, threadID uint32) (uintptr, error) {
			hook, _, err := procSetWindowsHookExW.Call(uintptr(idHook), hookProc, instance, uintptr(threadID))
			return hook, err
		},
		unhookWindowsHook: func(hook uintptr) {
			procUnhookWindowsHook.Call(hook)
		},
		showWindow: func(hwnd uintptr, command uintptr) {
			procShowWindow.Call(hwnd, command)
		},
		updateWindow: func(hwnd uintptr) {
			procUpdateWindow.Call(hwnd)
		},
		postMessage: func(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) bool {
			ret, _, _ := procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
			return ret != 0
		},
		getMessage: func(msg *message) (int32, error) {
			ret, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0)
			return int32(ret), err
		},
		translateMessage: func(msg *message) {
			procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
		},
		dispatchMessage: func(msg *message) {
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
		},
		destroyWindow: func(hwnd uintptr) {
			procDestroyWindow.Call(hwnd)
		},
		postQuitMessage: func(exitCode int) {
			procPostQuitMessage.Call(uintptr(exitCode))
		},
		monitorMetrics: getMonitorMetrics,
	}
}

func (a *application) currentWindowBounds(hwnd uintptr) windowBounds {
	if a == nil || a.winapi.monitorMetrics == nil {
		metrics := monitorMetrics{
			monitorW: windowFallbackMonitorW,
			monitorH: windowFallbackMonitorH,
			workW:    windowFallbackMonitorW,
			workH:    windowFallbackMonitorH,
			dpi:      defaultDPI,
		}
		return computeWindowBounds(metrics, windowFrame{})
	}
	metrics := a.winapi.monitorMetrics(hwnd)
	if a.dpi > 0 {
		metrics.dpi = a.dpi
	} else {
		a.dpi = normalizedDPI(metrics.dpi)
	}
	return computeWindowBounds(metrics, windowFrameForDPI(metrics.dpi))
}

func (a *application) currentDPI(hwnd uintptr) int {
	if a != nil && a.dpi > 0 {
		return a.dpi
	}
	dpi := getWindowDPI(hwnd)
	if a != nil {
		a.dpi = dpi
	}
	return dpi
}
