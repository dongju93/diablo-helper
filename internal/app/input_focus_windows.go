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
	if !a.runtimeInputTargetIsForeground() {
		releaseInjectedInputs()
		releaseMouseButtons()
		return nil
	}
	return sendVirtualKeyContext(ctx, vk, hold)
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
