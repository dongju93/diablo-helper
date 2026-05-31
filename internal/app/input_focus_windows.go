//go:build windows

package app

import (
	"context"
	"time"
)

func (a *application) captureRuntimeInputTarget() {
	if a == nil || a.winapi.getForegroundWindow == nil {
		return
	}
	target := a.winapi.getForegroundWindow()
	if target == 0 {
		return
	}
	a.runtimeInputTarget.Store(target)
}

func (a *application) clearRuntimeInputTargetIfIdle() {
	if a == nil {
		return
	}
	if !a.runner.Running() && !a.clicker.Running() {
		a.runtimeInputTarget.Store(0)
	}
}

func (a *application) sendRuntimeInput(ctx context.Context, vk uint16, hold time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if a != nil && a.shuttingDown.Load() {
		return context.Canceled
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !a.runtimeInputTargetIsForeground() {
		// Target window lost foreground: suspend input. Anything still held in
		// the target (a button whose up landed on another window after a
		// mid-press focus change) stays tracked so it is released into the
		// target once it regains focus, never into the wrong window.
		return nil
	}
	// Target is foreground: first release any input left held in it by an
	// earlier press whose key/button-up was delivered elsewhere, then send.
	releaseInjectedInputs()
	err := sendVirtualKeyContext(ctx, vk, hold)
	if err == nil && isMouseButton(vk) && !a.runtimeInputTargetIsForeground() {
		// Foreground left the target during this press, so the button-up went
		// to another window and the button is still down in the target. Track
		// it for release the next time the target is foreground (or on stop).
		markInjectedInputDown(vk)
	}
	return err
}

func (a *application) runtimeInputTargetIsForeground() bool {
	if a == nil || a.winapi.getForegroundWindow == nil {
		return true
	}
	target := a.runtimeInputTarget.Load()
	if target == 0 {
		return true
	}
	return a.winapi.getForegroundWindow() == target
}
