//go:build windows

package app

import (
	"fmt"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

type systemDirectoryLookup func(buffer []uint16) (uintptr, error)

var (
	// Go loads KnownDLL kernel32.dll from System32. Use this handle only for bootstrapping the System32 path.
	bootstrapKernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetSystemDirectoryW = bootstrapKernel32.NewProc("GetSystemDirectoryW")

	getSystemDirectory = getSystemDirectoryWin32

	winapiMu          sync.Mutex
	winapiInitialized bool
	system32Dir       string
)

var (
	kernel32 *syscall.LazyDLL
	user32   *syscall.LazyDLL
	gdi32    *syscall.LazyDLL
	dwmapi   *syscall.LazyDLL
	uxtheme  *syscall.LazyDLL
	comdlg32 *syscall.LazyDLL

	procAdjustWindowRectEx       *syscall.LazyProc
	procAdjustWindowRectExForDpi *syscall.LazyProc
	procBeginPaint               *syscall.LazyProc
	procCallNextHookEx           *syscall.LazyProc
	procCreateWindowExW          *syscall.LazyProc
	procDefWindowProcW           *syscall.LazyProc
	procDestroyWindow            *syscall.LazyProc
	procDispatchMessageW         *syscall.LazyProc
	procDrawTextW                *syscall.LazyProc
	procEndPaint                 *syscall.LazyProc
	procFillRect                 *syscall.LazyProc
	procGetClientRect            *syscall.LazyProc
	procGetDC                    *syscall.LazyProc
	procGetDlgItem               *syscall.LazyProc
	procGetDpiForSystem          *syscall.LazyProc
	procGetDpiForWindow          *syscall.LazyProc
	procGetMessageW              *syscall.LazyProc
	procGetMonitorInfoW          *syscall.LazyProc
	procGetSystemMetrics         *syscall.LazyProc
	procGetWindowTextW           *syscall.LazyProc
	procGetWindowTextLenW        *syscall.LazyProc
	procInvalidateRect           *syscall.LazyProc
	procLoadCursorW              *syscall.LazyProc
	procLoadIconW                *syscall.LazyProc
	procMessageBoxW              *syscall.LazyProc
	procMonitorFromWindow        *syscall.LazyProc
	procPostMessageW             *syscall.LazyProc
	procPostQuitMessage          *syscall.LazyProc
	procRegisterClassExW         *syscall.LazyProc
	procReleaseDC                *syscall.LazyProc
	procSendInput                *syscall.LazyProc
	procSendMessageW             *syscall.LazyProc
	procSetWindowPos             *syscall.LazyProc
	procSetWindowsHookExW        *syscall.LazyProc
	procSetWindowTextW           *syscall.LazyProc
	procMoveWindow               *syscall.LazyProc
	procShowWindow               *syscall.LazyProc
	procTranslateMessage         *syscall.LazyProc
	procUnhookWindowsHook        *syscall.LazyProc
	procUpdateWindow             *syscall.LazyProc

	procGetModuleHandleW         *syscall.LazyProc
	procSetDefaultDllDirectories *syscall.LazyProc
	procSetDllDirectoryW         *syscall.LazyProc
	procCreateFontW              *syscall.LazyProc
	procGetDeviceCaps            *syscall.LazyProc
	procEllipse                  *syscall.LazyProc
	procCreatePen                *syscall.LazyProc
	procCreateSolidBrush         *syscall.LazyProc
	procDeleteObject             *syscall.LazyProc
	procRoundRect                *syscall.LazyProc
	procSelectObject             *syscall.LazyProc
	procSetBkColor               *syscall.LazyProc
	procSetBkMode                *syscall.LazyProc
	procSetTextColor             *syscall.LazyProc

	procDwmSetWindowAttribute *syscall.LazyProc
	procSetWindowTheme        *syscall.LazyProc

	procCommDlgExtendedError *syscall.LazyProc
	procGetOpenFileNameW     *syscall.LazyProc
	procGetSaveFileNameW     *syscall.LazyProc
)

func ensureWinAPI() error {
	winapiMu.Lock()
	defer winapiMu.Unlock()

	if winapiInitialized {
		return nil
	}
	dir, err := systemDirectory()
	if err != nil {
		return err
	}
	system32Dir = dir
	loadWinAPIProcs()
	winapiInitialized = true
	return nil
}

func loadWinAPIProcs() {
	kernel32 = system32LazyDLL("kernel32.dll")
	user32 = system32LazyDLL("user32.dll")
	gdi32 = system32LazyDLL("gdi32.dll")
	dwmapi = system32LazyDLL("dwmapi.dll")
	uxtheme = system32LazyDLL("uxtheme.dll")
	comdlg32 = system32LazyDLL("comdlg32.dll")

	procAdjustWindowRectEx = user32.NewProc("AdjustWindowRectEx")
	procAdjustWindowRectExForDpi = user32.NewProc("AdjustWindowRectExForDpi")
	procBeginPaint = user32.NewProc("BeginPaint")
	procCallNextHookEx = user32.NewProc("CallNextHookEx")
	procCreateWindowExW = user32.NewProc("CreateWindowExW")
	procDefWindowProcW = user32.NewProc("DefWindowProcW")
	procDestroyWindow = user32.NewProc("DestroyWindow")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procDrawTextW = user32.NewProc("DrawTextW")
	procEndPaint = user32.NewProc("EndPaint")
	procFillRect = user32.NewProc("FillRect")
	procGetClientRect = user32.NewProc("GetClientRect")
	procGetDC = user32.NewProc("GetDC")
	procGetDlgItem = user32.NewProc("GetDlgItem")
	procGetDpiForSystem = user32.NewProc("GetDpiForSystem")
	procGetDpiForWindow = user32.NewProc("GetDpiForWindow")
	procGetMessageW = user32.NewProc("GetMessageW")
	procGetMonitorInfoW = user32.NewProc("GetMonitorInfoW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procGetWindowTextW = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW = user32.NewProc("GetWindowTextLengthW")
	procInvalidateRect = user32.NewProc("InvalidateRect")
	procLoadCursorW = user32.NewProc("LoadCursorW")
	procLoadIconW = user32.NewProc("LoadIconW")
	procMessageBoxW = user32.NewProc("MessageBoxW")
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procPostMessageW = user32.NewProc("PostMessageW")
	procPostQuitMessage = user32.NewProc("PostQuitMessage")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procReleaseDC = user32.NewProc("ReleaseDC")
	procSendInput = user32.NewProc("SendInput")
	procSendMessageW = user32.NewProc("SendMessageW")
	procSetWindowPos = user32.NewProc("SetWindowPos")
	procSetWindowsHookExW = user32.NewProc("SetWindowsHookExW")
	procSetWindowTextW = user32.NewProc("SetWindowTextW")
	procMoveWindow = user32.NewProc("MoveWindow")
	procShowWindow = user32.NewProc("ShowWindow")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procUnhookWindowsHook = user32.NewProc("UnhookWindowsHookEx")
	procUpdateWindow = user32.NewProc("UpdateWindow")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procSetDefaultDllDirectories = kernel32.NewProc("SetDefaultDllDirectories")
	procSetDllDirectoryW = kernel32.NewProc("SetDllDirectoryW")
	procCreateFontW = gdi32.NewProc("CreateFontW")
	procGetDeviceCaps = gdi32.NewProc("GetDeviceCaps")
	procEllipse = gdi32.NewProc("Ellipse")
	procCreatePen = gdi32.NewProc("CreatePen")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject = gdi32.NewProc("DeleteObject")
	procRoundRect = gdi32.NewProc("RoundRect")
	procSelectObject = gdi32.NewProc("SelectObject")
	procSetBkColor = gdi32.NewProc("SetBkColor")
	procSetBkMode = gdi32.NewProc("SetBkMode")
	procSetTextColor = gdi32.NewProc("SetTextColor")

	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procSetWindowTheme = uxtheme.NewProc("SetWindowTheme")

	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")
	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW = comdlg32.NewProc("GetSaveFileNameW")
}

func system32LazyDLL(name string) *syscall.LazyDLL {
	return syscall.NewLazyDLL(filepath.Join(system32Dir, name))
}

func systemDirectory() (string, error) {
	return systemDirectoryWith(getSystemDirectory)
}

func systemDirectoryWith(lookup systemDirectoryLookup) (string, error) {
	buffer := make([]uint16, syscall.MAX_PATH)
	for {
		n, err := lookup(buffer)
		if n == 0 {
			if err == nil || err == syscall.Errno(0) {
				return "", fmt.Errorf("GetSystemDirectoryW failed")
			}
			return "", fmt.Errorf("GetSystemDirectoryW failed: %w", err)
		}
		if n < uintptr(len(buffer)) {
			return syscall.UTF16ToString(buffer[:n]), nil
		}
		buffer = make([]uint16, n+1)
	}
}

func getSystemDirectoryWin32(buffer []uint16) (uintptr, error) {
	n, _, err := procGetSystemDirectoryW.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	return n, err
}
