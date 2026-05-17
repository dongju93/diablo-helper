//go:build windows

package app

import (
	"sync"
	"time"
)

type serializedInputSender struct {
	mu      sync.Mutex
	send    func(vk uint16, hold time.Duration) error
	release func(vk uint16) error
}

func newSerializedInputSender(send func(vk uint16, hold time.Duration) error, release func(vk uint16) error) *serializedInputSender {
	return &serializedInputSender{send: send, release: release}
}

func (s *serializedInputSender) Send(vk uint16, hold time.Duration) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	startedAt := time.Now()
	return startedAt, s.send(vk, hold)
}

func (s *serializedInputSender) Release(vk uint16) error {
	if s.release == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.release(vk)
}
