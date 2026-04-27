//go:build windows

package app

import (
	"fmt"
	"unsafe"
)

func sendVirtualKey(vk uint16) error {
	isMouse, err := sendMouseButton(vk)
	if isMouse {
		return err
	}
	down := newKeyboardInput(vk, 0)
	up := newKeyboardInput(vk, keyEventKeyUp)
	inputs := []input{down, up}
	n, _, _ := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if int(n) == len(inputs) {
		return nil
	}
	// down injected but up was not — best-effort recovery to avoid stuck key
	if int(n) == 1 {
		upOnly := []input{up}
		procSendInput.Call(1, uintptr(unsafe.Pointer(&upOnly[0])), unsafe.Sizeof(upOnly[0]))
	}
	return fmt.Errorf("SendInput: sent %d of %d keyboard events for vk %d", int(n), len(inputs), vk)
}

func sendMouseButton(vk uint16) (bool, error) {
	var downFlags, upFlags, data uint32

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
		return false, nil
	}

	down := newMouseInput(downFlags, data)
	up := newMouseInput(upFlags, data)
	inputs := []input{down, up}
	n, _, _ := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if int(n) == len(inputs) {
		return true, nil
	}
	// mouseDown injected but mouseUp was not — best-effort recovery
	if int(n) == 1 {
		upOnly := []input{up}
		procSendInput.Call(1, uintptr(unsafe.Pointer(&upOnly[0])), unsafe.Sizeof(upOnly[0]))
	}
	return true, fmt.Errorf("SendInput: sent %d of %d mouse events for vk %d", int(n), len(inputs), vk)
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
