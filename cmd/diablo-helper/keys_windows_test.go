//go:build windows

package main

import "testing"

func TestKeyDisplayName(t *testing.T) {
	tests := []struct {
		name string
		vk   uint16
		want string
	}{
		{name: "digit", vk: '7', want: "7"},
		{name: "letter", vk: 'K', want: "K"},
		{name: "function first", vk: vkF1, want: "F1"},
		{name: "function last", vk: vkF24, want: "F24"},
		{name: "numpad", vk: vkNumpad0 + 9, want: "Numpad 9"},
		{name: "mouse left", vk: vkLButton, want: "Mouse Left"},
		{name: "mouse right", vk: vkRButton, want: "Mouse Right"},
		{name: "mouse middle", vk: vkMButton, want: "Mouse Middle"},
		{name: "mouse x1", vk: vkXButton1, want: "Mouse X1"},
		{name: "mouse x2", vk: vkXButton2, want: "Mouse X2"},
		{name: "backspace", vk: vkBack, want: "Backspace"},
		{name: "tab", vk: vkTab, want: "Tab"},
		{name: "enter", vk: vkReturn, want: "Enter"},
		{name: "shift", vk: vkShift, want: "Shift"},
		{name: "ctrl", vk: vkControl, want: "Ctrl"},
		{name: "alt", vk: vkMenu, want: "Alt"},
		{name: "pause", vk: vkPause, want: "Pause"},
		{name: "caps lock", vk: vkCaps, want: "Caps Lock"},
		{name: "escape", vk: vkEscape, want: "Esc"},
		{name: "space", vk: vkSpace, want: "Space"},
		{name: "page up", vk: vkPrior, want: "Page Up"},
		{name: "page down", vk: vkNext, want: "Page Down"},
		{name: "end", vk: vkEnd, want: "End"},
		{name: "home", vk: vkHome, want: "Home"},
		{name: "left", vk: vkLeft, want: "Left"},
		{name: "up", vk: vkUp, want: "Up"},
		{name: "right", vk: vkRight, want: "Right"},
		{name: "down", vk: vkDown, want: "Down"},
		{name: "insert", vk: vkInsert, want: "Insert"},
		{name: "delete", vk: vkDelete, want: "Delete"},
		{name: "left win", vk: vkLWin, want: "Left Win"},
		{name: "right win", vk: vkRWin, want: "Right Win"},
		{name: "unknown", vk: 255, want: "VK_255"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keyDisplayName(tt.vk); got != tt.want {
				t.Fatalf("keyDisplayName(%d) = %q, want %q", tt.vk, got, tt.want)
			}
		})
	}
}
