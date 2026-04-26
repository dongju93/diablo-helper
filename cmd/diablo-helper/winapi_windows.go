//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const (
	colorWindow = 5

	cwUseDefault = 0x80000000

	wsOverlappedWindow = 0x00CF0000
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabStop          = 0x00010000
	wsBorder           = 0x00800000

	bsPushButton         = 0x00000000
	bsAutoCheckbox       = 0x00000003
	bsGroupBox           = 0x00000007
	bstChecked           = 1
	esNumber             = 0x00002000
	ssLeft               = 0x00000000
	defaultGUIFont       = 17
	idcArrow             = 32512
	inputMouse           = 0
	inputKeyboard        = 1
	keyEventKeyUp        = 0x0002
	llkhfInjected        = 0x00000010
	llmhfInjected        = 0x00000001
	mbOK                 = 0x00000000
	mbIconError          = 0x00000010
	mbIconInfo           = 0x00000040
	mouseEventLeftDown   = 0x0002
	mouseEventLeftUp     = 0x0004
	mouseEventRightDown  = 0x0008
	mouseEventRightUp    = 0x0010
	mouseEventMiddleDown = 0x0020
	mouseEventMiddleUp   = 0x0040
	mouseEventXDown      = 0x0080
	mouseEventXUp        = 0x0100
	swShow               = 5
	whKeyboardLL         = 13
	whMouseLL            = 14
	wmCreate             = 0x0001
	wmDestroy            = 0x0002
	wmClose              = 0x0010
	wmCommand            = 0x0111
	wmSetFont            = 0x0030
	wmKeyDown            = 0x0100
	wmKeyUp              = 0x0101
	wmSysKeyDown         = 0x0104
	wmSysKeyUp           = 0x0105
	wmLButtonDown        = 0x0201
	wmLButtonUp          = 0x0202
	wmRButtonDown        = 0x0204
	wmRButtonUp          = 0x0205
	wmMButtonDown        = 0x0207
	wmMButtonUp          = 0x0208
	wmXButtonDown        = 0x020B
	wmXButtonUp          = 0x020C
	bmGetCheck           = 0x00F0
	bmSetCheck           = 0x00F1
	bnClicked            = 0
	xButton1             = 0x0001
	xButton2             = 0x0002
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	procCallNextHookEx    = user32.NewProc("CallNextHookEx")
	procCreateWindowExW   = user32.NewProc("CreateWindowExW")
	procDefWindowProcW    = user32.NewProc("DefWindowProcW")
	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procDispatchMessageW  = user32.NewProc("DispatchMessageW")
	procGetDlgItem        = user32.NewProc("GetDlgItem")
	procGetMessageW       = user32.NewProc("GetMessageW")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW = user32.NewProc("GetWindowTextLengthW")
	procLoadCursorW       = user32.NewProc("LoadCursorW")
	procMessageBoxW       = user32.NewProc("MessageBoxW")
	procPostQuitMessage   = user32.NewProc("PostQuitMessage")
	procRegisterClassExW  = user32.NewProc("RegisterClassExW")
	procSendInput         = user32.NewProc("SendInput")
	procSendMessageW      = user32.NewProc("SendMessageW")
	procSetWindowsHookExW = user32.NewProc("SetWindowsHookExW")
	procSetWindowTextW    = user32.NewProc("SetWindowTextW")
	procShowWindow        = user32.NewProc("ShowWindow")
	procTranslateMessage  = user32.NewProc("TranslateMessage")
	procUnhookWindowsHook = user32.NewProc("UnhookWindowsHookEx")
	procUpdateWindow      = user32.NewProc("UpdateWindow")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
)

type point struct {
	X int32
	Y int32
}

type message struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type windowClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type keyboardHookStruct struct {
	VKCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type mouseHookStruct struct {
	Pt          point
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type mouseInput struct {
	DX          int32
	DY          int32
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type keyboardInput struct {
	VK          uint16
	Scan        uint16
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type input struct {
	Type uint32
	MI   mouseInput
}

func utf16Ptr(value string) *uint16 {
	ptr, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		panic(err)
	}
	return ptr
}

func lowWord(value uintptr) int {
	return int(value & 0xffff)
}

func highWord(value uintptr) int {
	return int((value >> 16) & 0xffff)
}

func defWindowProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func sendMessage(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	ret, _, _ := procSendMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func getDlgItem(hwnd uintptr, id int) uintptr {
	ret, _, _ := procGetDlgItem.Call(hwnd, uintptr(id))
	return ret
}

func setWindowText(hwnd uintptr, text string) {
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(text))))
}

func getWindowText(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLenW.Call(hwnd)
	buffer := make([]uint16, int(length)+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer)
}

func messageBox(hwnd uintptr, title string, text string, flags uintptr) {
	procMessageBoxW.Call(
		hwnd,
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		flags,
	)
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

func callNextKeyboardHook(code int, wParam uintptr, lParam uintptr) uintptr {
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(code), wParam, lParam)
	return ret
}

func sendVirtualKey(vk uint16) {
	if sendMouseButton(vk) {
		return
	}
	down := newKeyboardInput(vk, 0)
	up := newKeyboardInput(vk, keyEventKeyUp)
	inputs := []input{down, up}
	procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
}

func sendMouseButton(vk uint16) bool {
	var downFlags uint32
	var upFlags uint32
	var data uint32

	switch vk {
	case vkLButton:
		downFlags = mouseEventLeftDown
		upFlags = mouseEventLeftUp
	case vkRButton:
		downFlags = mouseEventRightDown
		upFlags = mouseEventRightUp
	case vkMButton:
		downFlags = mouseEventMiddleDown
		upFlags = mouseEventMiddleUp
	case vkXButton1:
		downFlags = mouseEventXDown
		upFlags = mouseEventXUp
		data = xButton1
	case vkXButton2:
		downFlags = mouseEventXDown
		upFlags = mouseEventXUp
		data = xButton2
	default:
		return false
	}

	down := newMouseInput(downFlags, data)
	up := newMouseInput(upFlags, data)
	inputs := []input{down, up}
	procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	return true
}

func newKeyboardInput(vk uint16, flags uint32) input {
	result := input{Type: inputKeyboard}
	keyboard := (*keyboardInput)(unsafe.Pointer(&result.MI))
	*keyboard = keyboardInput{VK: vk, Flags: flags}
	return result
}

func newMouseInput(flags uint32, data uint32) input {
	return input{
		Type: inputMouse,
		MI: mouseInput{
			MouseData: data,
			Flags:     flags,
		},
	}
}
