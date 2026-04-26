//go:build windows

package app

import (
	"unsafe"
)

func sendVirtualKey(vk uint16) {
	if sendMouseButton(vk) {
		return
	}
	down := newKeyboardInput(vk, 0)
	up := newKeyboardInput(vk, keyEventKeyUp)
	inputs := []input{down, up}
	procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
}

func sendMouseButton(vk uint16) bool {
	var downFlags uint32
	var upFlags uint32
	var data uint32

	switch vk {
	case vkLButton:
		downFlags = mouseEventLeftDown
		upFlags = mouseEventLeftUp
	case vkRButton:
		downFlags = mouseEventRightDown
		upFlags = mouseEventRightUp
	case vkMButton:
		downFlags = mouseEventMiddleDown
		upFlags = mouseEventMiddleUp
	case vkXButton1:
		downFlags = mouseEventXDown
		upFlags = mouseEventXUp
		data = xButton1
	case vkXButton2:
		downFlags = mouseEventXDown
		upFlags = mouseEventXUp
		data = xButton2
	default:
		return false
	}

	down := newMouseInput(downFlags, data)
	up := newMouseInput(upFlags, data)
	inputs := []input{down, up}
	procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	return true
}

func newKeyboardInput(vk uint16, flags uint32) input {
	result := input{Type: inputKeyboard}
	keyboard := (*keyboardInput)(unsafe.Pointer(&result.MI))
	*keyboard = keyboardInput{VK: vk, Flags: flags}
	return result
}

func newMouseInput(flags uint32, data uint32) input {
	return input{
		Type: inputMouse,
		MI: mouseInput{
			MouseData: data,
			Flags:     flags,
		},
	}
}
