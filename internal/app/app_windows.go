//go:build windows

package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/dongju93/diablo-helper/internal/config"
	"github.com/dongju93/diablo-helper/internal/meta"
)

type application struct {
	hwnd            uintptr
	instance        uintptr
	hook            uintptr
	mouseHook       uintptr
	font            uintptr
	titleFont       uintptr
	sectionFont     uintptr
	bgBrush         uintptr
	panelBrush      uintptr
	editBrush       uintptr
	borderPen       uintptr
	borderStrongPen uintptr
	borderBrush     uintptr
	accentBrush     uintptr
	accentPen       uintptr
	configPath      string
	cfg             config.Config
	controls        controlRefs
	capture         captureTarget
	pressed         pressedKeys
	runner          *skillRunner
	clicker         *clickerRunner
	skillEnabled    [config.MaxSkills]bool
	winapi          applicationWinAPI
	cleanedUp       bool
}

var (
	appInstance      *application
	windowProc       = syscall.NewCallback(wndProc)
	keyboardHookProc = syscall.NewCallback(lowLevelKeyboardProc)
	mouseHookProc    = syscall.NewCallback(lowLevelMouseProc)
)

const defaultConfigFileName = "default.toml"

func Run() error {
	if err := hardenDLLSearchPath(); err != nil {
		return err
	}
	runtime.LockOSThread()

	app := newApplication()
	return app.run()
}

func ShowFatalError(err error) {
	if err != nil {
		messageBox(0, "diablo-helper", err.Error(), mbOK|mbIconError)
	}
}

func newApplication() *application {
	return &application{
		cfg:    config.Default(),
		runner: newSkillRunner(sendVirtualKey),
		controls: controlRefs{
			menuLabels:  make(map[string]uintptr),
			menuButtons: make(map[string]uintptr),
		},
		clicker: newClickerRunner(sendVirtualKey),
		winapi:  defaultApplicationWinAPI(),
	}
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
	hwnd, err := a.winapi.createWindowEx(
		wsExComposited,
		utf16Ptr("DiabloHelperWindow"),
		utf16Ptr(meta.Title()),
		wsOverlappedWindow,
		cwUseDefault,
		cwUseDefault,
		windowMaxW,
		windowMaxH,
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

	if a.runner != nil {
		a.runner.Stop()
	}
	if a.clicker != nil {
		a.clicker.Stop()
	}
	if a.hook != 0 {
		a.winapi.unhookWindowsHook(a.hook)
		a.hook = 0
	}
	if a.mouseHook != 0 {
		a.winapi.unhookWindowsHook(a.mouseHook)
		a.mouseHook = 0
	}
	a.disposeUIResources()
	if appInstance == a {
		appInstance = nil
	}
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
	getMessage        func(msg *message) (int32, error)
	translateMessage  func(msg *message)
	dispatchMessage   func(msg *message)
	destroyWindow     func(hwnd uintptr)
	postQuitMessage   func(exitCode int)
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
	}
}
