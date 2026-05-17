//go:build windows

package app

import (
	"fmt"
	"time"
	"unsafe"
)

var sendInputCall = callSendInput

func sendVirtualKey(vk uint16, hold time.Duration) error {
	isMouse, err := sendMouseButton(vk, hold)
	if isMouse {
		return err
	}
	down := newKeyboardInput(vk, 0)
	if err := sendSingleInput(down, "keyboard down", vk); err != nil {
		return err
	}
	sleepInputHold(hold)
	up := newKeyboardInput(vk, keyEventKeyUp)
	if err := sendSingleInput(up, "keyboard up", vk); err == nil {
		return nil
	} else {
		// down injected but up was not; best-effort recovery avoids a stuck key.
		_ = sendSingleInput(up, "keyboard up recovery", vk)
		return err
	}
}

func sendMouseButton(vk uint16, hold time.Duration) (bool, error) {
	downFlags, upFlags, data, ok := mouseButtonInput(vk)
	if !ok {
		return false, nil
	}

	down := newMouseInput(downFlags, data)
	if err := sendSingleInput(down, "mouse down", vk); err != nil {
		return true, err
	}
	sleepInputHold(hold)
	up := newMouseInput(upFlags, data)
	if err := sendSingleInput(up, "mouse up", vk); err == nil {
		return true, nil
	} else {
		// mouseDown injected but mouseUp was not; best-effort recovery.
		_ = sendSingleInput(up, "mouse up recovery", vk)
		return true, err
	}
}

func releaseVirtualKey(vk uint16) error {
	if vk == 0 {
		return nil
	}

	isMouse, err := releaseMouseButton(vk)
	if isMouse {
		return err
	}
	return sendSingleInput(newKeyboardInput(vk, keyEventKeyUp), "keyboard up release", vk)
}

func releaseMouseButton(vk uint16) (bool, error) {
	_, upFlags, data, ok := mouseButtonInput(vk)
	if !ok {
		return false, nil
	}
	return true, sendSingleInput(newMouseInput(upFlags, data), "mouse up release", vk)
}

func mouseButtonInput(vk uint16) (downFlags, upFlags, data uint32, ok bool) {
	switch vk {
	case vkLButton:
		return mouseEventLeftDown, mouseEventLeftUp, 0, true
	case vkRButton:
		return mouseEventRightDown, mouseEventRightUp, 0, true
	case vkMButton:
		return mouseEventMiddleDown, mouseEventMiddleUp, 0, true
	case vkXButton1:
		return mouseEventXDown, mouseEventXUp, xButton1, true
	case vkXButton2:
		return mouseEventXDown, mouseEventXUp, xButton2, true
	default:
		return 0, 0, 0, false
	}
}

func sendSingleInput(item input, label string, vk uint16) error {
	return sendInputCall(item, label, vk)
}

func callSendInput(item input, label string, vk uint16) error {
	inputs := []input{item}
	n, _, callErr := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
	if n == 1 {
		return nil
	}
	return fmt.Errorf("SendInput: sent %d of 1 %s event for vk %d: %w", int(n), label, vk, callErr)
}

func sleepInputHold(hold time.Duration) {
	if hold > 0 {
		time.Sleep(hold)
	}
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
