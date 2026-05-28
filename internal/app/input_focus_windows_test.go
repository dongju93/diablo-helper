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

func TestRuntimeInputTargetMismatchReleasesTrackedInput(t *testing.T) {
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

	foreground = 200
	if err := a.sendRuntimeInput(context.Background(), 'A', 0); err != nil {
		t.Fatalf("sendRuntimeInput() error = %v", err)
	}
	if len(released) != 1 || released[0] != vkLButton {
		t.Fatalf("released keys = %v, want [%d]", released, vkLButton)
	}
	if sent != 0 {
		t.Fatalf("sent keyboard events = %d, want 0", sent)
	}
}
