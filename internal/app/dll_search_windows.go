//go:build windows

package app

import (
	"fmt"
	"unsafe"
)

const (
	loadLibrarySearchUserDirs = 0x00000400
	loadLibrarySearchSystem32 = 0x00000800
)

func hardenDLLSearchPath() error {
	flags := uintptr(loadLibrarySearchSystem32 | loadLibrarySearchUserDirs)
	if ret, _, err := procSetDefaultDllDirectories.Call(flags); ret == 0 {
		return fmt.Errorf("SetDefaultDllDirectories failed: %w", err)
	}

	if ret, _, err := procSetDllDirectoryW.Call(uintptr(unsafe.Pointer(utf16Ptr("")))); ret == 0 {
		return fmt.Errorf("SetDllDirectoryW failed: %w", err)
	}
	return nil
}
