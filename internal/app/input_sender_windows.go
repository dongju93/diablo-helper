//go:build windows

package app

import (
	"context"
	"time"
)

type serializedInputSender struct {
	mu      chan struct{}
	send    func(ctx context.Context, vk uint16, hold time.Duration) error
	release func(vk uint16) error
}

func newSerializedInputSender(send func(vk uint16, hold time.Duration) error, release func(vk uint16) error) *serializedInputSender {
	return newSerializedContextInputSender(func(_ context.Context, vk uint16, hold time.Duration) error {
		return send(vk, hold)
	}, release)
}

func newSerializedContextInputSender(send func(ctx context.Context, vk uint16, hold time.Duration) error, release func(vk uint16) error) *serializedInputSender {
	return &serializedInputSender{
		mu:      make(chan struct{}, 1),
		send:    send,
		release: release,
	}
}

func (s *serializedInputSender) Send(vk uint16, hold time.Duration) (time.Time, error) {
	return s.SendContext(context.Background(), vk, hold)
}

func (s *serializedInputSender) SendContext(ctx context.Context, vk uint16, hold time.Duration) (time.Time, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.acquire(ctx); err != nil {
		return time.Now(), err
	}
	defer s.releaseLock()
	if err := ctx.Err(); err != nil {
		return time.Now(), err
	}
	startedAt := time.Now()
	return startedAt, s.send(ctx, vk, hold)
}

func (s *serializedInputSender) Release(vk uint16) error {
	return s.ReleaseContext(context.Background(), vk)
}

func (s *serializedInputSender) ReleaseContext(ctx context.Context, vk uint16) error {
	if s.release == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.acquire(ctx); err != nil {
		return err
	}
	defer s.releaseLock()
	return s.release(vk)
}

func (s *serializedInputSender) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case s.mu <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *serializedInputSender) releaseLock() {
	<-s.mu
}
