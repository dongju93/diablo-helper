//go:build windows

package app

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	regWindowPlacementValue         = `WindowPlacement`
	maxWindowPlacementBytes         = 256
	maxWindowPlacementAbsCoordinate = 1_000_000
	maxWindowPlacementDimension     = 100_000
)

type savedWindowPlacement struct {
	x         int32
	y         int32
	width     int32
	height    int32
	maximized bool
}

type initialWindowPlacement struct {
	x          uintptr
	y          uintptr
	width      uintptr
	height     uintptr
	restore    savedWindowPlacement
	hasRestore bool
}

func defaultInitialWindowPlacement(bounds windowBounds) initialWindowPlacement {
	return initialWindowPlacement{
		x:      cwUseDefault,
		y:      cwUseDefault,
		width:  uintptr(bounds.maxW),
		height: uintptr(bounds.maxH),
	}
}

func (a *application) initialWindowPlacement(defaultBounds windowBounds) initialWindowPlacement {
	initial := defaultInitialWindowPlacement(defaultBounds)
	if a == nil || a.winapi.loadWindowPlacement == nil {
		return initial
	}

	placement, ok := a.winapi.loadWindowPlacement()
	if !ok {
		return initial
	}

	placement, ok = normalizeWindowPlacement(placement, defaultBounds)
	if !ok {
		return initial
	}

	initial.restore = placement
	initial.hasRestore = true
	return initial
}

func (a *application) restoreWindowPlacement(hwnd uintptr, placement savedWindowPlacement) bool {
	if a == nil || hwnd == 0 || a.winapi.setWindowPlacement == nil {
		return false
	}
	info := placement.windowPlacementInfo()
	return a.winapi.setWindowPlacement(hwnd, &info)
}

func (a *application) saveCurrentWindowPlacement() {
	if a == nil || a.hwnd == 0 || a.winapi.getWindowPlacement == nil || a.winapi.saveWindowPlacement == nil {
		return
	}
	placement, ok := a.winapi.getWindowPlacement(a.hwnd)
	if !ok {
		return
	}
	a.winapi.saveWindowPlacement(placement)
}

func normalizeWindowPlacement(placement savedWindowPlacement, bounds windowBounds) (savedWindowPlacement, bool) {
	if placement.width <= 0 || placement.height <= 0 {
		return savedWindowPlacement{}, false
	}

	placement.width = clampWindowDimension(placement.width, bounds.minW, bounds.maxW)
	placement.height = clampWindowDimension(placement.height, bounds.minH, bounds.maxH)
	return placement, true
}

func clampWindowDimension(value int32, minValue int32, maxValue int32) int32 {
	if minValue < 1 {
		minValue = 1
	}
	if maxValue < minValue {
		maxValue = minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (placement savedWindowPlacement) rect() rect {
	return rect{
		Left:   placement.x,
		Top:    placement.y,
		Right:  placement.x + placement.width,
		Bottom: placement.y + placement.height,
	}
}

func (placement savedWindowPlacement) windowPlacementInfo() windowPlacementInfo {
	showCommand := uint32(swShowNormal)
	if placement.maximized {
		showCommand = swShowMaximized
	}
	return windowPlacementInfo{
		Length:         uint32(unsafe.Sizeof(windowPlacementInfo{})),
		ShowCmd:        showCommand,
		NormalPosition: placement.rect(),
	}
}

func captureWindowPlacement(hwnd uintptr) (savedWindowPlacement, bool) {
	if hwnd == 0 || procGetWindowPlacement == nil {
		return savedWindowPlacement{}, false
	}

	info := windowPlacementInfo{Length: uint32(unsafe.Sizeof(windowPlacementInfo{}))}
	ret, _, _ := procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return savedWindowPlacement{}, false
	}

	normal := info.NormalPosition
	width := normal.Right - normal.Left
	height := normal.Bottom - normal.Top
	if width <= 0 || height <= 0 {
		return savedWindowPlacement{}, false
	}
	return savedWindowPlacement{
		x:         normal.Left,
		y:         normal.Top,
		width:     width,
		height:    height,
		maximized: info.ShowCmd == swShowMaximized || info.Flags&wpfRestoreToMaximized != 0,
	}, true
}

func loadSavedWindowPlacement() (savedWindowPlacement, bool) {
	value, ok := loadWindowPlacementRegistryString()
	if !ok {
		return savedWindowPlacement{}, false
	}
	return parseSavedWindowPlacement(value)
}

func saveSavedWindowPlacement(placement savedWindowPlacement) {
	if !validSavedWindowPlacement(placement) {
		return
	}
	saveWindowPlacementRegistryString(formatSavedWindowPlacement(placement))
}

func parseSavedWindowPlacement(value string) (savedWindowPlacement, bool) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) != 4 && len(parts) != 5 {
		return savedWindowPlacement{}, false
	}

	x, ok := parseWindowPlacementInt32(parts[0])
	if !ok || absInt32(x) > maxWindowPlacementAbsCoordinate {
		return savedWindowPlacement{}, false
	}
	y, ok := parseWindowPlacementInt32(parts[1])
	if !ok || absInt32(y) > maxWindowPlacementAbsCoordinate {
		return savedWindowPlacement{}, false
	}
	width, ok := parseWindowPlacementInt32(parts[2])
	if !ok || width <= 0 || width > maxWindowPlacementDimension {
		return savedWindowPlacement{}, false
	}
	height, ok := parseWindowPlacementInt32(parts[3])
	if !ok || height <= 0 || height > maxWindowPlacementDimension {
		return savedWindowPlacement{}, false
	}

	maximized := false
	if len(parts) == 5 {
		switch strings.TrimSpace(parts[4]) {
		case "0":
		case "1":
			maximized = true
		default:
			return savedWindowPlacement{}, false
		}
	}
	return savedWindowPlacement{x: x, y: y, width: width, height: height, maximized: maximized}, true
}

func formatSavedWindowPlacement(placement savedWindowPlacement) string {
	maximized := 0
	if placement.maximized {
		maximized = 1
	}
	return fmt.Sprintf("%d,%d,%d,%d,%d", placement.x, placement.y, placement.width, placement.height, maximized)
}

func validSavedWindowPlacement(placement savedWindowPlacement) bool {
	if absInt32(placement.x) > maxWindowPlacementAbsCoordinate || absInt32(placement.y) > maxWindowPlacementAbsCoordinate {
		return false
	}
	return placement.width > 0 &&
		placement.width <= maxWindowPlacementDimension &&
		placement.height > 0 &&
		placement.height <= maxWindowPlacementDimension
}

func parseWindowPlacementInt32(value string) (int32, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, false
	}
	return int32(parsed), true
}

func absInt32(value int32) int64 {
	if value < 0 {
		return -int64(value)
	}
	return int64(value)
}

func loadWindowPlacementRegistryString() (string, bool) {
	subKey, err := syscall.UTF16PtrFromString(regAppSubKey)
	if err != nil {
		return "", false
	}
	valueName, err := syscall.UTF16PtrFromString(regWindowPlacementValue)
	if err != nil {
		return "", false
	}

	var cbData uint32
	ret, _, _ := procRegGetValueW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(subKey)),
		uintptr(unsafe.Pointer(valueName)),
		rrfRtRegSz,
		0,
		0,
		uintptr(unsafe.Pointer(&cbData)),
	)
	if ret != 0 || cbData == 0 || cbData > maxWindowPlacementBytes {
		return "", false
	}

	buf := make([]uint16, (cbData+1)/2)
	ret, _, _ = procRegGetValueW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(subKey)),
		uintptr(unsafe.Pointer(valueName)),
		rrfRtRegSz,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&cbData)),
	)
	if ret != 0 {
		return "", false
	}
	return syscall.UTF16ToString(buf), true
}

func saveWindowPlacementRegistryString(value string) {
	subKey, err := syscall.UTF16PtrFromString(regAppSubKey)
	if err != nil {
		return
	}
	valueName, err := syscall.UTF16PtrFromString(regWindowPlacementValue)
	if err != nil {
		return
	}
	data, err := syscall.UTF16FromString(value)
	if err != nil {
		return
	}

	var hKey uintptr
	ret, _, _ := procRegCreateKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(subKey)),
		0,
		0,
		regOptionNonVolatile,
		keySetValue,
		0,
		uintptr(unsafe.Pointer(&hKey)),
		0,
	)
	if ret != 0 {
		return
	}
	defer procRegCloseKey.Call(hKey)

	cbData := uint32(len(data) * 2)
	procRegSetValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
		0,
		regSz,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(cbData),
	)
}
