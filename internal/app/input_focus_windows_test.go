//go:build windows

package app

import (
	"context"
	"testing"
)

func TestRuntimeInputSkipsWhenForegroundLeavesCapturedTarget(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	var keyEvents int
	sendInputCall = func(_ input, label string, _ uint16) error {
		switch label {
		case "keyboard down", "keyboard up":
			keyEvents++
		}
		return nil
	}

	foreground := uintptr(100)
	a := newApplication()
	a.winapi.getForegroundWindow = func() uintptr {
		return foreground
	}
	a.captureRuntimeInputTarget()

	foreground = 200
	if err := a.sendRuntimeInput(context.Background(), 'A', 0); err != nil {
		t.Fatalf("sendRuntimeInput() error = %v", err)
	}
	if keyEvents != 0 {
		t.Fatalf("keyboard events = %d after foreground changed, want 0", keyEvents)
	}

	foreground = 100
	if err := a.sendRuntimeInput(context.Background(), 'A', 0); err != nil {
		t.Fatalf("sendRuntimeInput() error = %v", err)
	}
	if keyEvents != 2 {
		t.Fatalf("keyboard events = %d after foreground restored, want 2", keyEvents)
	}
}

func TestRuntimeInputWithoutCapturedTargetAllowsSend(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	var calls int
	sendInputCall = func(_ input, _ string, _ uint16) error {
		calls++
		return nil
	}

	a := newApplication()
	a.winapi.getForegroundWindow = func() uintptr {
		return 200
	}

	if err := a.sendRuntimeInput(context.Background(), 'A', 0); err != nil {
		t.Fatalf("sendRuntimeInput() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("sendRuntimeInput() calls = %d, want 2", calls)
	}
}

func TestRuntimeInputHoldsTrackedInputOffTargetThenReleasesIntoTarget(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	var released []uint16
	var sent int
	sendInputCall = func(_ input, label string, vk uint16) error {
		switch label {
		case "mouse up release":
			released = append(released, vk)
		case "keyboard down", "keyboard up":
			sent++
		}
		return nil
	}

	foreground := uintptr(100)
	a := newApplication()
	a.winapi.getForegroundWindow = func() uintptr {
		return foreground
	}
	a.captureRuntimeInputTarget()
	markInjectedInputDown(vkLButton)

	// Target lost foreground: the button held in the target must not be
	// released into the other window, and must stay tracked.
	foreground = 200
	if err := a.sendRuntimeInput(context.Background(), 'A', 0); err != nil {
		t.Fatalf("sendRuntimeInput() error = %v", err)
	}
	if len(released) != 0 {
		t.Fatalf("released keys = %v while off target, want none", released)
	}
	if sent != 0 {
		t.Fatalf("sent keyboard events = %d while off target, want 0", sent)
	}
	if !injectedInputs.has(vkLButton) {
		t.Fatalf("tracked button should remain held while target is not foreground")
	}

	// Target regained foreground: the held button is released into the target
	// before the next input is sent.
	foreground = 100
	if err := a.sendRuntimeInput(context.Background(), 'A', 0); err != nil {
		t.Fatalf("sendRuntimeInput() error = %v", err)
	}
	if len(released) != 1 || released[0] != vkLButton {
		t.Fatalf("released keys = %v, want [%d]", released, vkLButton)
	}
	if sent != 2 {
		t.Fatalf("sent keyboard events = %d after target restored, want 2", sent)
	}
	if injectedInputs.has(vkLButton) {
		t.Fatalf("tracked button should be cleared after release into target")
	}
}

func TestRuntimeInputRetainsMouseHeldWhenFocusLeavesMidPress(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	sendInputCall = func(_ input, _ string, _ uint16) error {
		return nil
	}

	const target = uintptr(100)
	var foregroundCalls int
	a := newApplication()
	a.winapi.getForegroundWindow = func() uintptr {
		// On target at the start of the cycle, but focus has left by the time
		// the press completes.
		foregroundCalls++
		if foregroundCalls == 1 {
			return target
		}
		return 200
	}
	a.runtimeInputTarget.Store(target)

	if err := a.sendRuntimeInput(context.Background(), vkRButton, 0); err != nil {
		t.Fatalf("sendRuntimeInput() error = %v", err)
	}
	if !injectedInputs.has(vkRButton) {
		t.Fatalf("mouse button should stay tracked as held in target after mid-press focus loss")
	}
}
