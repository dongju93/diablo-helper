//go:build windows

package app

import (
	"fmt"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func utf16Ptr(value string) *uint16 {
	ptr, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		panic(err)
	}
	return ptr
}

func utf16PtrSafe(value string) (*uint16, error) {
	return syscall.UTF16PtrFromString(value)
}

func utf16Slice(value string) []uint16 {
	return append(utf16.Encode([]rune(value)), 0)
}

func lowWord(value uintptr) int {
	return int(value & 0xffff)
}

func highWord(value uintptr) int {
	return int((value >> 16) & 0xffff)
}

func makeLong(low int, high int) uintptr {
	return uintptr(uint32(uint16(low)) | uint32(uint16(high))<<16)
}

func defWindowProc(hwnd uintptr, msg uint32, wParam uintptr, lParam unsafe.Pointer) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, uintptr(lParam))
	return ret
}

func sendMessage(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	ret, _, _ := procSendMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func invalidateRect(hwnd uintptr, erase bool) {
	eraseValue := uintptr(0)
	if erase {
		eraseValue = 1
	}
	procInvalidateRect.Call(hwnd, 0, eraseValue)
}

func getDlgItem(hwnd uintptr, id int) uintptr {
	ret, _, _ := procGetDlgItem.Call(hwnd, uintptr(id))
	return ret
}

func setWindowText(hwnd uintptr, text string) error {
	ptr, err := utf16PtrSafe(text)
	if err != nil {
		return err
	}
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(ptr)))
	return nil
}

func setWindowTheme(hwnd uintptr, theme string) {
	if hwnd == 0 {
		return
	}
	procSetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(theme))), 0)
}

func setWindowVisuals(hwnd uintptr) {
	cornerPreference := int32(dwmwcpRound)
	procDwmSetWindowAttribute.Call(
		hwnd,
		uintptr(dwmwaWindowCornerPref),
		uintptr(unsafe.Pointer(&cornerPreference)),
		unsafe.Sizeof(cornerPreference),
	)
	backdrop := int32(dwmSystemBackdropMain)
	procDwmSetWindowAttribute.Call(
		hwnd,
		uintptr(dwmwaSystemBackdrop),
		uintptr(unsafe.Pointer(&backdrop)),
		unsafe.Sizeof(backdrop),
	)
}

func getWindowText(hwnd uintptr) (string, error) {
	length, _, _ := procGetWindowTextLenW.Call(hwnd)
	if int(length) > maxWindowTextLen {
		return "", fmt.Errorf("window text length %d exceeds maximum %d", int(length), maxWindowTextLen)
	}
	buffer := make([]uint16, int(length)+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer), nil
}

func messageBox(hwnd uintptr, title string, text string, flags uintptr) error {
	_, err := messageBoxResult(hwnd, title, text, flags)
	return err
}

func messageBoxResult(hwnd uintptr, title string, text string, flags uintptr) (uintptr, error) {
	textPtr, err := utf16PtrSafe(text)
	if err != nil {
		return 0, err
	}
	titlePtr, err := utf16PtrSafe(title)
	if err != nil {
		return 0, err
	}
	ret, _, _ := procMessageBoxW.Call(
		hwnd,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		flags,
	)
	return ret, nil
}

func checked(hwnd uintptr) bool {
	return sendMessage(hwnd, bmGetCheck, 0, 0) == bstChecked
}

func setChecked(hwnd uintptr, value bool) {
	check := uintptr(0)
	if value {
		check = bstChecked
	}
	sendMessage(hwnd, bmSetCheck, check, 0)
}

func getClientSize(hwnd uintptr) (int, int) {
	var cr rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
	return int(cr.Right), int(cr.Bottom)
}

type monitorMetrics struct {
	monitorW int
	monitorH int
	workW    int
	workH    int
}

func getMonitorMetrics(hwnd uintptr) monitorMetrics {
	metrics := fallbackMonitorMetrics()
	flag := uintptr(monitorDefaultToNearest)
	if hwnd == 0 {
		flag = monitorDefaultToPrimary
	}
	monitor, _, _ := procMonitorFromWindow.Call(hwnd, flag)
	if monitor == 0 {
		return metrics
	}

	info := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	ret, _, _ := procGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return metrics
	}

	monitorW := int(info.Monitor.Right - info.Monitor.Left)
	monitorH := int(info.Monitor.Bottom - info.Monitor.Top)
	workW := int(info.Work.Right - info.Work.Left)
	workH := int(info.Work.Bottom - info.Work.Top)
	if monitorW <= 0 || monitorH <= 0 {
		return metrics
	}
	metrics.monitorW = monitorW
	metrics.monitorH = monitorH
	if workW > 0 && workH > 0 {
		metrics.workW = workW
		metrics.workH = workH
	} else {
		metrics.workW = monitorW
		metrics.workH = monitorH
	}
	return metrics
}

func fallbackMonitorMetrics() monitorMetrics {
	width, _, _ := procGetSystemMetrics.Call(smCxScreen)
	height, _, _ := procGetSystemMetrics.Call(smCyScreen)
	monitorW := int(width)
	monitorH := int(height)
	if monitorW <= 0 || monitorH <= 0 {
		monitorW = windowReferenceMonitorW
		monitorH = windowReferenceMonitorH
	}
	return monitorMetrics{
		monitorW: monitorW,
		monitorH: monitorH,
		workW:    monitorW,
		workH:    monitorH,
	}
}

func moveControl(hwnd uintptr, x, y, width, height int) {
	procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 1)
}

func callNextHookEx(code int, wParam uintptr, lParam unsafe.Pointer) uintptr {
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(code), wParam, uintptr(lParam))
	return ret
}
