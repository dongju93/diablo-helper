//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	colorWindow = 5

	cwUseDefault = 0x80000000

	wsExClientEdge     = 0x00000200
	wsOverlappedWindow = 0x00CF0000
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabStop          = 0x00010000
	wsBorder           = 0x00800000

	bsPushButton          = 0x00000000
	bsAutoCheckbox        = 0x00000003
	bsGroupBox            = 0x00000007
	bsOwnerDraw           = 0x0000000B
	bstChecked            = 1
	cleartypeQuality      = 5
	defaultCharset        = 1
	dwmSystemBackdropMain = 2
	esNumber              = 0x00002000
	esAutoHScroll         = 0x00000080
	transparent           = 1
	ssLeft                = 0x00000000
	defaultGUIFont        = 17
	dwmwaWindowCornerPref = 33
	dwmwaSystemBackdrop   = 38
	dwmwcpRound           = 2
	dtCenter              = 0x00000001
	dtVCenter             = 0x00000004
	dtSingleLine          = 0x00000020
	dtEndEllipsis         = 0x00008000
	dtNoPrefix            = 0x00000800
	ecLeftMargin          = 0x0001
	ecRightMargin         = 0x0002
	emSetMargins          = 0x00D3
	fwNormal              = 400
	fwSemiBold            = 600
	idcArrow              = 32512
	inputMouse            = 0
	inputKeyboard         = 1
	keyEventKeyUp         = 0x0002
	llkhfInjected         = 0x00000010
	llmhfInjected         = 0x00000001
	mbOK                  = 0x00000000
	mbIconError           = 0x00000010
	mbIconWarning         = 0x00000030
	mbIconInfo            = 0x00000040
	mouseEventLeftDown    = 0x0002
	mouseEventLeftUp      = 0x0004
	mouseEventRightDown   = 0x0008
	mouseEventRightUp     = 0x0010
	mouseEventMiddleDown  = 0x0020
	mouseEventMiddleUp    = 0x0040
	mouseEventXDown       = 0x0080
	mouseEventXUp         = 0x0100
	swShow                = 5
	whKeyboardLL          = 13
	whMouseLL             = 14
	wmCreate              = 0x0001
	wmDestroy             = 0x0002
	wmPaint               = 0x000F
	wmClose               = 0x0010
	wmEraseBkgnd          = 0x0014
	wmDrawItem            = 0x002B
	wmCommand             = 0x0111
	wmSetFont             = 0x0030
	wmKeyDown             = 0x0100
	wmKeyUp               = 0x0101
	wmSysKeyDown          = 0x0104
	wmSysKeyUp            = 0x0105
	wmCtlColorEdit        = 0x0133
	wmCtlColorBtn         = 0x0135
	wmCtlColorStatic      = 0x0138
	wmLButtonDown         = 0x0201
	wmLButtonUp           = 0x0202
	wmRButtonDown         = 0x0204
	wmRButtonUp           = 0x0205
	wmMButtonDown         = 0x0207
	wmMButtonUp           = 0x0208
	wmXButtonDown         = 0x020B
	wmXButtonUp           = 0x020C
	bmGetCheck            = 0x00F0
	bmSetCheck            = 0x00F1
	bnClicked             = 0
	odsSelected           = 0x0001
	odsDisabled           = 0x0004
	odsFocus              = 0x0010
	odsHotLight           = 0x0040
	psSolid               = 0
	xButton1              = 0x0001
	xButton2              = 0x0002

	maxFileDialogPath = 32768

	ofnOverwritePrompt  = 0x00000002
	ofnHideReadonly     = 0x00000004
	ofnNoChangeDir      = 0x00000008
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnNoReadonlyReturn = 0x00008000
	ofnExplorer         = 0x00080000
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")

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
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procEllipse          = gdi32.NewProc("Ellipse")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
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

type point struct {
	X int32
	Y int32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type paintStruct struct {
	HDC         uintptr
	Erase       int32
	RcPaint     rect
	Restore     int32
	IncUpdate   int32
	RGBReserved [32]byte
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

type drawItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     rect
	ItemData   uintptr
}

type openFileName struct {
	StructSize    uint32
	HwndOwner     uintptr
	Instance      uintptr
	Filter        *uint16
	CustomFilter  *uint16
	MaxCustFilter uint32
	FilterIndex   uint32
	File          *uint16
	MaxFile       uint32
	FileTitle     *uint16
	MaxFileTitle  uint32
	InitialDir    *uint16
	Title         *uint16
	Flags         uint32
	FileOffset    uint16
	FileExtension uint16
	DefExt        *uint16
	CustData      uintptr
	FnHook        uintptr
	TemplateName  *uint16
	PvReserved    uintptr
	DwReserved    uint32
	FlagsEx       uint32
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

func defWindowProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
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

func setWindowText(hwnd uintptr, text string) {
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(text))))
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

func chooseConfigOpenPath(hwnd uintptr, initialPath string) (string, bool, error) {
	return chooseConfigPath(hwnd, "설정 불러오기", initialPath, false)
}

func chooseConfigSavePath(hwnd uintptr, initialPath string) (string, bool, error) {
	return chooseConfigPath(hwnd, "설정 저장", initialPath, true)
}

func chooseConfigPath(hwnd uintptr, title string, initialPath string, save bool) (string, bool, error) {
	initialName, initialDir := fileDialogInitialNameAndDir(initialPath)
	fileBuffer := fileDialogBuffer(initialName)
	filter := utf16Slice("TOML 설정 파일 (*.toml)\x00*.toml\x00모든 파일 (*.*)\x00*.*\x00")
	titleText := utf16Slice(title)
	defExt := utf16Slice("toml")

	flags := uint32(ofnExplorer | ofnHideReadonly | ofnNoChangeDir | ofnPathMustExist)
	if save {
		flags |= ofnOverwritePrompt | ofnNoReadonlyReturn
	} else {
		flags |= ofnFileMustExist
	}

	ofn := openFileName{
		StructSize:  uint32(unsafe.Sizeof(openFileName{})),
		HwndOwner:   hwnd,
		Filter:      &filter[0],
		FilterIndex: 1,
		File:        &fileBuffer[0],
		MaxFile:     uint32(len(fileBuffer)),
		Title:       &titleText[0],
		Flags:       flags,
		DefExt:      &defExt[0],
	}

	var initialDirText []uint16
	if initialDir != "" {
		initialDirText = utf16Slice(initialDir)
		ofn.InitialDir = &initialDirText[0]
	}

	var ret uintptr
	if save {
		ret, _, _ = procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	} else {
		ret, _, _ = procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	}
	runtime.KeepAlive(fileBuffer)
	runtime.KeepAlive(filter)
	runtime.KeepAlive(titleText)
	runtime.KeepAlive(defExt)
	runtime.KeepAlive(initialDirText)

	if ret == 0 {
		errCode, _, _ := procCommDlgExtendedError.Call()
		if errCode == 0 {
			return "", false, nil
		}
		return "", false, fmt.Errorf("file dialog failed with code 0x%04x", errCode)
	}

	path := syscall.UTF16ToString(fileBuffer)
	if path == "" {
		return "", false, fmt.Errorf("file dialog returned an empty path")
	}
	return path, true, nil
}

func fileDialogInitialNameAndDir(initialPath string) (string, string) {
	if initialPath == "" {
		return "settings.toml", ""
	}
	name := filepath.Base(initialPath)
	if name == "." || name == string(filepath.Separator) {
		name = "settings.toml"
	}
	dir := filepath.Dir(initialPath)
	if dir == "." || dir == name {
		dir = ""
	}
	return name, dir
}

func fileDialogBuffer(initialName string) []uint16 {
	buffer := make([]uint16, maxFileDialogPath)
	initial := utf16.Encode([]rune(initialName))
	if len(initial) >= len(buffer) {
		initial = initial[:len(buffer)-1]
	}
	copy(buffer, initial)
	return buffer
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
