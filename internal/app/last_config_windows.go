//go:build windows

package app

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	regAppSubKey           = `Software\diablo-helper`
	regLastConfigValue     = `LastConfigPath`
	maxLastConfigPathBytes = 32 * 1024
)

// loadLastConfigPath reads the last-used config path from the registry.
// Returns "" if the key is absent, unreadable, oversized, or contains a relative path.
func loadLastConfigPath() string {
	subKey, err := syscall.UTF16PtrFromString(regAppSubKey)
	if err != nil {
		return ""
	}
	valueName, err := syscall.UTF16PtrFromString(regLastConfigValue)
	if err != nil {
		return ""
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
	if ret != 0 || cbData == 0 || cbData > maxLastConfigPathBytes {
		return ""
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
		return ""
	}
	path := syscall.UTF16ToString(buf)
	if !filepath.IsAbs(path) {
		return ""
	}
	return path
}

// saveLastConfigPath persists configPath to HKCU\Software\diablo-helper.
// Errors are silently discarded — this is best-effort state.
func saveLastConfigPath(configPath string) {
	subKey, err := syscall.UTF16PtrFromString(regAppSubKey)
	if err != nil {
		return
	}
	valueName, err := syscall.UTF16PtrFromString(regLastConfigValue)
	if err != nil {
		return
	}
	data, err := syscall.UTF16FromString(configPath)
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
