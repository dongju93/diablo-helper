//go:build windows

package app

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"unicode/utf16"
	"unsafe"
)

func TestRGBAndWordHelpers(t *testing.T) {
	if got := rgb(1, 2, 3); got != 0x030201 {
		t.Fatalf("rgb(1, 2, 3) = %#x, want %#x", got, uintptr(0x030201))
	}
	if got := int32Arg(-1); got != 0xffffffff {
		t.Fatalf("int32Arg(-1) = %#x, want 0xffffffff", got)
	}
	value := uintptr(0x12345678)
	if got := lowWord(value); got != 0x5678 {
		t.Fatalf("lowWord() = %#x, want 0x5678", got)
	}
	if got := highWord(value); got != 0x1234 {
		t.Fatalf("highWord() = %#x, want 0x1234", got)
	}
	if got := makeLong(0x5678, 0x1234); got != value {
		t.Fatalf("makeLong() = %#x, want %#x", got, value)
	}
}

func TestInputConstructors(t *testing.T) {
	keyboard := newKeyboardInput('A', keyEventKeyUp)
	if keyboard.Type != inputKeyboard {
		t.Fatalf("keyboard input type = %d, want %d", keyboard.Type, inputKeyboard)
	}
	keyboardData := (*keyboardInput)(unsafe.Pointer(&keyboard.MI))
	if keyboardData.VK != 'A' {
		t.Fatalf("keyboard VK = %d, want %d", keyboardData.VK, 'A')
	}
	if keyboardData.Flags != keyEventKeyUp {
		t.Fatalf("keyboard flags = %#x, want %#x", keyboardData.Flags, keyEventKeyUp)
	}

	mouse := newMouseInput(mouseEventXDown, xButton2)
	if mouse.Type != inputMouse {
		t.Fatalf("mouse input type = %d, want %d", mouse.Type, inputMouse)
	}
	if mouse.MI.Flags != mouseEventXDown {
		t.Fatalf("mouse flags = %#x, want %#x", mouse.MI.Flags, mouseEventXDown)
	}
	if mouse.MI.MouseData != xButton2 {
		t.Fatalf("mouse data = %#x, want %#x", mouse.MI.MouseData, xButton2)
	}
}

func TestFileDialogInitialNameAndDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantName string
		wantDir  string
	}{
		{name: "empty", wantName: "default.toml"},
		{name: "relative file", path: "default.toml", wantName: "default.toml"},
		{name: "absolute file", path: `C:\profiles\season.toml`, wantName: "season.toml", wantDir: `C:\profiles`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDir := fileDialogInitialNameAndDir(tt.path)
			if gotName != tt.wantName || gotDir != tt.wantDir {
				t.Fatalf("fileDialogInitialNameAndDir() = (%q, %q), want (%q, %q)", gotName, gotDir, tt.wantName, tt.wantDir)
			}
		})
	}
}

func TestFileDialogBuffer(t *testing.T) {
	buffer := fileDialogBuffer("custom.toml")
	if len(buffer) != maxFileDialogPath {
		t.Fatalf("buffer length = %d, want %d", len(buffer), maxFileDialogPath)
	}
	if got := syscall.UTF16ToString(buffer); got != "custom.toml" {
		t.Fatalf("buffer text = %q, want custom.toml", got)
	}
	if buffer[len("custom.toml")] != 0 {
		t.Fatal("buffer is not null-terminated after the initial name")
	}
}

func TestFileDialogOpenFilterIncludesAllFiles(t *testing.T) {
	filter := configOpenFileDialogFilter()
	text := string(utf16.Decode(filter))
	if !strings.Contains(text, "*.toml") {
		t.Fatalf("filter = %q, want *.toml", text)
	}
	if !strings.Contains(text, "*.*") {
		t.Fatalf("filter = %q, want all-files option", text)
	}
	if len(filter) < 2 || filter[len(filter)-1] != 0 || filter[len(filter)-2] != 0 {
		t.Fatalf("filter is not double-null terminated: %#v", filter)
	}
}

func TestFileDialogSaveFilterAllowsOnlyTOML(t *testing.T) {
	filter := configSaveFileDialogFilter()
	text := string(utf16.Decode(filter))
	if !strings.Contains(text, "*.toml") {
		t.Fatalf("filter = %q, want *.toml", text)
	}
	if strings.Contains(text, "*.*") {
		t.Fatalf("filter = %q, want no all-files option", text)
	}
	if len(filter) < 2 || filter[len(filter)-1] != 0 || filter[len(filter)-2] != 0 {
		t.Fatalf("filter is not double-null terminated: %#v", filter)
	}
}

func TestWinAPIDLLsLoadFromSystem32(t *testing.T) {
	dlls := map[*syscall.LazyDLL]string{
		user32:   "user32.dll",
		gdi32:    "gdi32.dll",
		dwmapi:   "dwmapi.dll",
		uxtheme:  "uxtheme.dll",
		comdlg32: "comdlg32.dll",
	}

	for dll, wantName := range dlls {
		if !filepath.IsAbs(dll.Name) {
			t.Fatalf("%s path = %q, want an absolute path", wantName, dll.Name)
		}
		if got := filepath.Base(dll.Name); !strings.EqualFold(got, wantName) {
			t.Fatalf("DLL name = %q, want %q", got, wantName)
		}
		if got := filepath.Dir(dll.Name); !strings.EqualFold(got, system32Dir) {
			t.Fatalf("%s directory = %q, want %q", wantName, got, system32Dir)
		}
	}
}
