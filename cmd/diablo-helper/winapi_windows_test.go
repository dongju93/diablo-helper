//go:build windows

package main

import (
	"syscall"
	"testing"
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
		{name: "empty", wantName: "settings.toml"},
		{name: "relative file", path: "settings.toml", wantName: "settings.toml"},
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
