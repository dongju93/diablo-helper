//go:build windows

package app

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	// Go loads kernel32.dll from System32; it is the bootstrap for resolving other system DLL paths.
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetSystemDirectoryW = kernel32.NewProc("GetSystemDirectoryW")
)

var system32Dir = mustSystemDirectory()

var (
	user32   = system32LazyDLL("user32.dll")
	gdi32    = system32LazyDLL("gdi32.dll")
	dwmapi   = system32LazyDLL("dwmapi.dll")
	uxtheme  = system32LazyDLL("uxtheme.dll")
	comdlg32 = system32LazyDLL("comdlg32.dll")

	procBeginPaint        = user32.NewProc("BeginPaint")
	procCallNextHookEx    = user32.NewProc("CallNextHookEx")
	procCreateWindowExW   = user32.NewProc("CreateWindowExW")
	procDefWindowProcW    = user32.NewProc("DefWindowProcW")
	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procDispatchMessageW  = user32.NewProc("DispatchMessageW")
	procDrawTextW         = user32.NewProc("DrawTextW")
	procEndPaint          = user32.NewProc("EndPaint")
	procFillRect          = user32.NewProc("FillRect")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procGetDlgItem        = user32.NewProc("GetDlgItem")
	procGetMessageW       = user32.NewProc("GetMessageW")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW = user32.NewProc("GetWindowTextLengthW")
	procInvalidateRect    = user32.NewProc("InvalidateRect")
	procLoadCursorW       = user32.NewProc("LoadCursorW")
	procLoadIconW         = user32.NewProc("LoadIconW")
	procMessageBoxW       = user32.NewProc("MessageBoxW")
	procPostQuitMessage   = user32.NewProc("PostQuitMessage")
	procRegisterClassExW  = user32.NewProc("RegisterClassExW")
	procSendInput         = user32.NewProc("SendInput")
	procSendMessageW      = user32.NewProc("SendMessageW")
	procSetWindowsHookExW = user32.NewProc("SetWindowsHookExW")
	procSetWindowTextW    = user32.NewProc("SetWindowTextW")
	procMoveWindow        = user32.NewProc("MoveWindow")
	procShowWindow        = user32.NewProc("ShowWindow")
	procTranslateMessage  = user32.NewProc("TranslateMessage")
	procUnhookWindowsHook = user32.NewProc("UnhookWindowsHookEx")
	procUpdateWindow      = user32.NewProc("UpdateWindow")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procEllipse          = gdi32.NewProc("Ellipse")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procSetBkColor       = gdi32.NewProc("SetBkColor")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSetTextColor     = gdi32.NewProc("SetTextColor")

	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procSetWindowTheme        = uxtheme.NewProc("SetWindowTheme")

	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")
	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW     = comdlg32.NewProc("GetSaveFileNameW")
)

func system32LazyDLL(name string) *syscall.LazyDLL {
	return syscall.NewLazyDLL(filepath.Join(system32Dir, name))
}

func mustSystemDirectory() string {
	buffer := make([]uint16, syscall.MAX_PATH)
	for {
		n, _, err := procGetSystemDirectoryW.Call(
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(len(buffer)),
		)
		if n == 0 {
			panic("GetSystemDirectoryW failed: " + err.Error())
		}
		if n < uintptr(len(buffer)) {
			return syscall.UTF16ToString(buffer[:n])
		}
		buffer = make([]uint16, n+1)
	}
}
