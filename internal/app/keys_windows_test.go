//go:build windows

package app

import (
	"testing"

	"github.com/dongju93/diablo-helper/internal/config"
)

func TestKeyDisplayName(t *testing.T) {
	tests := []struct {
		name string
		vk   int
		want string
	}{
		{name: "digit", vk: '7', want: "7"},
		{name: "letter", vk: 'K', want: "K"},
		{name: "function first", vk: 0x70, want: "F1"},
		{name: "function last", vk: 0x87, want: "F24"},
		{name: "numpad", vk: 0x69, want: "Numpad 9"},
		{name: "mouse left", vk: 0x01, want: "Mouse Left"},
		{name: "mouse right", vk: 0x02, want: "Mouse Right"},
		{name: "mouse middle", vk: 0x04, want: "Mouse Middle"},
		{name: "mouse x1", vk: 0x05, want: "Mouse X1"},
		{name: "mouse x2", vk: 0x06, want: "Mouse X2"},
		{name: "backspace", vk: 0x08, want: "Backspace"},
		{name: "tab", vk: 0x09, want: "Tab"},
		{name: "enter", vk: 0x0D, want: "Enter"},
		{name: "shift", vk: 0x10, want: "Shift"},
		{name: "ctrl", vk: 0x11, want: "Ctrl"},
		{name: "alt", vk: 0x12, want: "Alt"},
		{name: "pause", vk: 0x13, want: "Pause"},
		{name: "caps lock", vk: 0x14, want: "Caps Lock"},
		{name: "escape", vk: 0x1B, want: "Esc"},
		{name: "space", vk: 0x20, want: "Space"},
		{name: "page up", vk: 0x21, want: "Page Up"},
		{name: "page down", vk: 0x22, want: "Page Down"},
		{name: "end", vk: 0x23, want: "End"},
		{name: "home", vk: 0x24, want: "Home"},
		{name: "left", vk: 0x25, want: "Left"},
		{name: "up", vk: 0x26, want: "Up"},
		{name: "right", vk: 0x27, want: "Right"},
		{name: "down", vk: 0x28, want: "Down"},
		{name: "insert", vk: 0x2D, want: "Insert"},
		{name: "delete", vk: 0x2E, want: "Delete"},
		{name: "left win", vk: 0x5B, want: "Left Win"},
		{name: "right win", vk: 0x5C, want: "Right Win"},
		{name: "unknown", vk: 255, want: "VK_255"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := config.KeyDisplayName(tt.vk); got != tt.want {
				t.Fatalf("KeyDisplayName(%d) = %q, want %q", tt.vk, got, tt.want)
			}
		})
	}
}
