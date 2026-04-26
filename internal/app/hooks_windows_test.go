//go:build windows

package app

import (
	"testing"
)

func TestMouseEventKey(t *testing.T) {
	tests := []struct {
		name      string
		message   uintptr
		mouseData uint32
		wantVK    uint16
		wantDown  bool
		wantOK    bool
	}{
		{name: "left down", message: wmLButtonDown, wantVK: vkLButton, wantDown: true, wantOK: true},
		{name: "left up", message: wmLButtonUp, wantVK: vkLButton, wantDown: false, wantOK: true},
		{name: "right down", message: wmRButtonDown, wantVK: vkRButton, wantDown: true, wantOK: true},
		{name: "right up", message: wmRButtonUp, wantVK: vkRButton, wantDown: false, wantOK: true},
		{name: "middle down", message: wmMButtonDown, wantVK: vkMButton, wantDown: true, wantOK: true},
		{name: "middle up", message: wmMButtonUp, wantVK: vkMButton, wantDown: false, wantOK: true},
		{name: "x1 down", message: wmXButtonDown, mouseData: xButton1 << 16, wantVK: vkXButton1, wantDown: true, wantOK: true},
		{name: "x1 up", message: wmXButtonUp, mouseData: xButton1 << 16, wantVK: vkXButton1, wantDown: false, wantOK: true},
		{name: "x2 down", message: wmXButtonDown, mouseData: xButton2 << 16, wantVK: vkXButton2, wantDown: true, wantOK: true},
		{name: "unknown x button", message: wmXButtonDown, mouseData: 3 << 16, wantOK: false},
		{name: "unknown message", message: wmKeyDown, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVK, gotDown, gotOK := mouseEventKey(tt.message, &mouseHookStruct{MouseData: tt.mouseData})
			if gotVK != tt.wantVK || gotDown != tt.wantDown || gotOK != tt.wantOK {
				t.Fatalf("mouseEventKey() = (%d, %v, %v), want (%d, %v, %v)", gotVK, gotDown, gotOK, tt.wantVK, tt.wantDown, tt.wantOK)
			}
		})
	}
}
