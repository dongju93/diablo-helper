//go:build windows

package app

import "fmt"

const (
	vkLButton  = 0x01
	vkRButton  = 0x02
	vkMButton  = 0x04
	vkXButton1 = 0x05
	vkXButton2 = 0x06
	vkBack     = 0x08
	vkTab      = 0x09
	vkReturn   = 0x0D
	vkShift    = 0x10
	vkControl  = 0x11
	vkMenu     = 0x12
	vkPause    = 0x13
	vkCaps     = 0x14
	vkEscape   = 0x1B
	vkSpace    = 0x20
	vkPrior    = 0x21
	vkNext     = 0x22
	vkEnd      = 0x23
	vkHome     = 0x24
	vkLeft     = 0x25
	vkUp       = 0x26
	vkRight    = 0x27
	vkDown     = 0x28
	vkInsert   = 0x2D
	vkDelete   = 0x2E
	vkLWin     = 0x5B
	vkRWin     = 0x5C
	vkNumpad0  = 0x60
	vkF1       = 0x70
	vkF24      = 0x87
)

func keyDisplayName(vk uint16) string {
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= vkF1 && vk <= vkF24 {
		return fmt.Sprintf("F%d", vk-vkF1+1)
	}
	if vk >= vkNumpad0 && vk <= vkNumpad0+9 {
		return fmt.Sprintf("Numpad %d", vk-vkNumpad0)
	}

	switch vk {
	case vkLButton:
		return "Mouse Left"
	case vkRButton:
		return "Mouse Right"
	case vkMButton:
		return "Mouse Middle"
	case vkXButton1:
		return "Mouse X1"
	case vkXButton2:
		return "Mouse X2"
	case vkBack:
		return "Backspace"
	case vkTab:
		return "Tab"
	case vkReturn:
		return "Enter"
	case vkShift:
		return "Shift"
	case vkControl:
		return "Ctrl"
	case vkMenu:
		return "Alt"
	case vkPause:
		return "Pause"
	case vkCaps:
		return "Caps Lock"
	case vkEscape:
		return "Esc"
	case vkSpace:
		return "Space"
	case vkPrior:
		return "Page Up"
	case vkNext:
		return "Page Down"
	case vkEnd:
		return "End"
	case vkHome:
		return "Home"
	case vkLeft:
		return "Left"
	case vkUp:
		return "Up"
	case vkRight:
		return "Right"
	case vkDown:
		return "Down"
	case vkInsert:
		return "Insert"
	case vkDelete:
		return "Delete"
	case vkLWin:
		return "Left Win"
	case vkRWin:
		return "Right Win"
	default:
		return fmt.Sprintf("VK_%d", vk)
	}
}
