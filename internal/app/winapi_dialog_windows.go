//go:build windows

package app

import (
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func chooseConfigOpenPath(hwnd uintptr, initialPath string) (string, bool, error) {
	return chooseConfigPath(hwnd, "설정 불러오기", initialPath, false)
}

func chooseConfigSavePath(hwnd uintptr, initialPath string) (string, bool, error) {
	return chooseConfigPath(hwnd, "설정 저장", initialPath, true)
}

func chooseConfigPath(hwnd uintptr, title string, initialPath string, save bool) (string, bool, error) {
	initialName, initialDir := fileDialogInitialNameAndDir(initialPath)
	fileBuffer := fileDialogBuffer(initialName)
	filter := configSaveFileDialogFilter()
	if !save {
		filter = configOpenFileDialogFilter()
	}
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

func configOpenFileDialogFilter() []uint16 {
	return utf16Slice("TOML 설정 파일 (*.toml)\x00*.toml\x00모든 파일 (*.*)\x00*.*\x00")
}

func configSaveFileDialogFilter() []uint16 {
	return utf16Slice("TOML 설정 파일 (*.toml)\x00*.toml\x00")
}

func fileDialogInitialNameAndDir(initialPath string) (string, string) {
	if initialPath == "" {
		return defaultConfigFileName, ""
	}
	name := filepath.Base(initialPath)
	if name == "." || name == string(filepath.Separator) {
		name = defaultConfigFileName
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
