//go:build windows

package app

import (
	"context"
	"testing"
	"time"
)

func TestSerializedInputSenderDoesNotOverlapSends(t *testing.T) {
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	sender := newSerializedInputSender(func(vk uint16, _ time.Duration) error {
		switch vk {
		case 'A':
			close(firstStarted)
			<-releaseFirst
		case 'B':
			close(secondStarted)
		}
		return nil
	}, nil)

	firstDone := make(chan error, 1)
	go func() {
		_, err := sender.Send('A', 0)
		firstDone <- err
	}()
	select {
	case <-firstStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for first send to start")
	}

	secondDone := make(chan error, 1)
	go func() {
		_, err := sender.Send('B', 0)
		secondDone <- err
	}()
	select {
	case <-secondStarted:
		t.Fatal("second send started before first send completed")
	case <-time.After(30 * time.Millisecond):
	}

	close(releaseFirst)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first Send() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for first send to finish")
	}
	select {
	case <-secondStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for second send to start")
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second Send() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for second send to finish")
	}
}

func TestSerializedInputSenderCancelsWhileWaitingForAnotherSend(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})
	sender := newSerializedInputSender(func(vk uint16, _ time.Duration) error {
		switch vk {
		case 'A':
			close(firstStarted)
			<-releaseFirst
		case 'B':
			close(secondStarted)
		}
		return nil
	}, nil)

	firstDone := make(chan error, 1)
	go func() {
		_, err := sender.Send('A', 0)
		firstDone <- err
	}()
	select {
	case <-firstStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for first send to start")
	}

	ctx, cancel := context.WithCancel(context.Background())
	secondDone := make(chan error, 1)
	go func() {
		_, err := sender.SendContext(ctx, 'B', 0)
		secondDone <- err
	}()

	select {
	case err := <-secondDone:
		t.Fatalf("SendContext() finished before cancellation: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-secondDone:
		if err == nil {
			t.Fatal("SendContext() error = nil, want context cancellation")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("SendContext() waited behind another send after cancellation")
	}
	select {
	case <-secondStarted:
		t.Fatal("canceled SendContext() started a send")
	default:
	}

	close(releaseFirst)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first Send() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for first send to finish")
	}
}
