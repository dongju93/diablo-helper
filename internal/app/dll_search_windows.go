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

type dllSearchProc interface {
	Find() error
	Call(a ...uintptr) (uintptr, uintptr, error)
}

func hardenDLLSearchPath() error {
	if err := ensureWinAPI(); err != nil {
		return err
	}
	return hardenDLLSearchPathWithProcs(procSetDefaultDllDirectories, procSetDllDirectoryW)
}

func hardenDLLSearchPathWithProcs(setDefaultDllDirectories, setDllDirectory dllSearchProc) error {
	flags := uintptr(loadLibrarySearchSystem32 | loadLibrarySearchUserDirs)
	if err := setDefaultDllDirectories.Find(); err != nil {
		return fmt.Errorf("SetDefaultDllDirectories unavailable: %w", err)
	}
	if ret, _, err := setDefaultDllDirectories.Call(flags); ret == 0 {
		return fmt.Errorf("SetDefaultDllDirectories failed: %w", err)
	}

	if err := setDllDirectory.Find(); err != nil {
		return fmt.Errorf("SetDllDirectoryW unavailable: %w", err)
	}
	if ret, _, err := setDllDirectory.Call(uintptr(unsafe.Pointer(utf16Ptr("")))); ret == 0 {
		return fmt.Errorf("SetDllDirectoryW failed: %w", err)
	}
	return nil
}
