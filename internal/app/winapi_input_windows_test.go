//go:build windows

package app

import (
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
