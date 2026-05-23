//go:build windows

package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSendVirtualKeyDoesNotWaitDuringAnotherKeyInputHold(t *testing.T) {
	oldSendInputCall := sendInputCall
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
		sendInputCall = oldSendInputCall
		close(errs)
		for err := range errs {
			if err != nil {
				t.Errorf("sendVirtualKey() error = %v", err)
			}
		}
	}()

	firstDown := make(chan struct{})
	secondDown := make(chan struct{})
	var firstOnce sync.Once
	var secondOnce sync.Once
	sendInputCall = func(_ input, label string, vk uint16) error {
		if label != "keyboard down" {
			return nil
		}
		switch vk {
		case 'A':
			firstOnce.Do(func() { close(firstDown) })
		case 'B':
			secondOnce.Do(func() { close(secondDown) })
		}
		return nil
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		errs <- sendVirtualKey('A', 200*time.Millisecond)
	}()

	select {
	case <-firstDown:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for first key down")
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		errs <- sendVirtualKey('B', time.Millisecond)
	}()

	select {
	case <-secondDown:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("second key down waited for first key hold to finish")
	}
}

func TestSendVirtualKeyContextCancelsInputHoldAndReleasesKey(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	keyDown := make(chan struct{})
	keyUp := make(chan struct{})
	var downOnce sync.Once
	var upOnce sync.Once
	sendInputCall = func(_ input, label string, _ uint16) error {
		switch label {
		case "keyboard down":
			downOnce.Do(func() { close(keyDown) })
		case "keyboard up":
			upOnce.Do(func() { close(keyUp) })
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- sendVirtualKeyContext(ctx, 'A', 200*time.Millisecond)
	}()

	select {
	case <-keyDown:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for key down")
	}

	cancel()
	select {
	case <-keyUp:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for key up after cancellation")
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("sendVirtualKeyContext() error = %v, want context.Canceled", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for sendVirtualKeyContext to return")
	}
}

func TestReleaseInjectedInputsRecoversMouseButtonAfterMissingUp(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	upErr := errors.New("up failed")
	releaseCount := 0
	sendInputCall = func(_ input, label string, _ uint16) error {
		switch label {
		case "mouse up", "mouse up recovery":
			return upErr
		case "mouse up release":
			releaseCount++
		}
		return nil
	}

	if err := sendVirtualKeyContext(context.Background(), vkLButton, 0); !errors.Is(err, upErr) {
		t.Fatalf("sendVirtualKeyContext() error = %v, want %v", err, upErr)
	}
	releaseInjectedInputs()
	if releaseCount != 1 {
		t.Fatalf("release count = %d, want 1", releaseCount)
	}
	releaseInjectedInputs()
	if releaseCount != 1 {
		t.Fatalf("second release count = %d, want still 1", releaseCount)
	}
}

func TestReleaseInjectedInputsKeepsFailedReleaseTracked(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	releaseErr := errors.New("release failed")
	releaseCount := 0
	sendInputCall = func(_ input, label string, _ uint16) error {
		if label == "mouse up release" {
			releaseCount++
			return releaseErr
		}
		return nil
	}

	markInjectedInputDown(vkLButton)
	releaseInjectedInputs()
	releaseInjectedInputs()
	if releaseCount != 2 {
		t.Fatalf("release count = %d, want retry while release fails", releaseCount)
	}
}

func TestSendMouseButtonPreReleasesMouseButtons(t *testing.T) {
	injectedInputs.reset()
	oldSendInputCall := sendInputCall
	defer func() {
		sendInputCall = oldSendInputCall
		injectedInputs.reset()
	}()

	var labels []string
	sendInputCall = func(_ input, label string, _ uint16) error {
		labels = append(labels, label)
		return nil
	}

	if err := sendVirtualKeyContext(context.Background(), vkLButton, 0); err != nil {
		t.Fatalf("sendVirtualKeyContext() error = %v", err)
	}
	wantPrefix := []string{
		"mouse down",
		"mouse up",
		"mouse up release",
	}
	if len(labels) != len(wantPrefix) {
		t.Fatalf("labels = %v, want %v", labels, wantPrefix)
	}
	for i := range wantPrefix {
		if labels[i] != wantPrefix[i] {
			t.Fatalf("labels = %v, want %v", labels, wantPrefix)
		}
	}
}

func TestInjectedInputTrackerIgnoresOutOfRangeVK(t *testing.T) {
	injectedInputs.reset()
	defer injectedInputs.reset()

	markInjectedInputDown(256)
	if got := injectedInputs.keys(); len(got) != 0 {
		t.Fatalf("tracked keys = %v, want none", got)
	}

	for _, vk := range []uint16{vkLButton, 'A'} {
		markInjectedInputDown(vk)
	}
	got := injectedInputs.keys()
	if len(got) != 2 || got[0] != vkLButton || got[1] != 'A' {
		t.Fatalf("tracked keys = %v, want [%d %d]", got, vkLButton, 'A')
	}
}
