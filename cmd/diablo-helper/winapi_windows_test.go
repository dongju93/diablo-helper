//go:build windows

package main

import (
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
