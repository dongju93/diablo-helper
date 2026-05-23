//go:build windows

package app

import (
	"context"
	"fmt"
	"time"
	"unsafe"
)

var sendInputCall = callSendInput

func sendVirtualKey(vk uint16, hold time.Duration) error {
	return sendVirtualKeyContext(context.Background(), vk, hold)
}

func sendVirtualKeyContext(ctx context.Context, vk uint16, hold time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	isMouse, err := sendMouseButtonContext(ctx, vk, hold)
	if isMouse || err != nil {
		return err
	}
	down := newKeyboardInput(vk, 0)
	if err := sendSingleInput(down, "keyboard down", vk); err != nil {
		return err
	}
	holdErr := waitInputHold(ctx, hold)
	up := newKeyboardInput(vk, keyEventKeyUp)
	if err := sendSingleInput(up, "keyboard up", vk); err != nil {
		// down injected but up was not; best-effort recovery avoids a stuck key.
		_ = sendSingleInput(up, "keyboard up recovery", vk)
		return err
	}
	return holdErr
}

func sendMouseButton(vk uint16, hold time.Duration) (bool, error) {
	return sendMouseButtonContext(context.Background(), vk, hold)
}

func sendMouseButtonContext(ctx context.Context, vk uint16, hold time.Duration) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	downFlags, upFlags, data, ok := mouseButtonInput(vk)
	if !ok {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return true, err
	}

	down := newMouseInput(downFlags, data)
	if err := sendSingleInput(down, "mouse down", vk); err != nil {
		return true, err
	}
	holdErr := waitInputHold(ctx, hold)
	up := newMouseInput(upFlags, data)
	if err := sendSingleInput(up, "mouse up", vk); err != nil {
		// mouseDown injected but mouseUp was not; best-effort recovery.
		_ = sendSingleInput(up, "mouse up recovery", vk)
		return true, err
	}
	return true, holdErr
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

func waitInputHold(ctx context.Context, hold time.Duration) error {
	if hold <= 0 {
		return nil
	}
	timer := time.NewTimer(hold)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
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
