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
	pressed         map[uint16]bool
	runner          *skillRunner
	clicker         *clickerRunner
	skillEnabled    [config.MaxSkills]bool
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
		cfg:     config.Default(),
		pressed: make(map[uint16]bool),
		runner:  newSkillRunner(sendVirtualKey),
		controls: controlRefs{
			menuLabels:  make(map[string]uintptr),
			menuButtons: make(map[string]uintptr),
		},
		clicker: newClickerRunner(sendVirtualKey),
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

	instance, _, err := procGetModuleHandleW.Call(0)
	if instance == 0 {
		return fmt.Errorf("GetModuleHandleW failed: %w", err)
	}
	a.instance = instance

	if err := a.registerWindowClass(); err != nil {
		return err
	}

	appInstance = a
	hwnd, _, err := procCreateWindowExW.Call(
		wsExComposited,
		uintptr(unsafe.Pointer(utf16Ptr("DiabloHelperWindow"))),
		uintptr(unsafe.Pointer(utf16Ptr(meta.Title()))),
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
	setWindowVisuals(hwnd)

	hook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, keyboardHookProc, a.instance, 0)
	if hook == 0 {
		return fmt.Errorf("SetWindowsHookExW failed: %w", err)
	}
	a.hook = hook
	mouseHook, _, err := procSetWindowsHookExW.Call(whMouseLL, mouseHookProc, a.instance, 0)
	if mouseHook == 0 {
		procUnhookWindowsHook.Call(a.hook)
		a.hook = 0
		return fmt.Errorf("SetWindowsHookExW mouse hook failed: %w", err)
	}
	a.mouseHook = mouseHook

	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)

	var msg message
	for {
		ret, _, callErr := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			return fmt.Errorf("GetMessageW failed: %w", callErr)
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func (a *application) registerWindowClass() error {
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	icon, _, _ := procLoadIconW.Call(a.instance, uintptr(1))
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
	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", err)
	}
	return nil
}

func defaultConfigPath() string {
	executable, err := os.Executable()
	if err != nil {
		return defaultConfigFileName
	}
	return filepath.Join(filepath.Dir(executable), defaultConfigFileName)
}
