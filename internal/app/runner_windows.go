//go:build windows

package app

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dongju93/diablo-helper/internal/config"
)

type skillRunner struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	running atomic.Bool
	paused  atomic.Bool
	sendKey func(vk uint16)
}

func newSkillRunner(sendKey func(vk uint16)) *skillRunner {
	return &skillRunner{sendKey: sendKey}
}

func (r *skillRunner) Start(cfg config.Config) bool {
	cfg.Normalize()
	skills := runnableSkills(cfg)
	if len(skills) == 0 {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.running.Store(true)
	r.paused.Store(false)

	for _, skill := range skills {
		go r.runSkill(ctx, skill)
	}
	return true
}

func (r *skillRunner) Stop() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel == nil {
		return false
	}
	r.cancel()
	r.cancel = nil
	r.running.Store(false)
	r.paused.Store(false)
	return true
}

func (r *skillRunner) SetPaused(paused bool) {
	if !r.running.Load() {
		r.paused.Store(false)
		return
	}
	r.paused.Store(paused)
}

func (r *skillRunner) Running() bool {
	return r.running.Load()
}

func (r *skillRunner) Paused() bool {
	return r.running.Load() && r.paused.Load()
}

func (r *skillRunner) runSkill(ctx context.Context, skill config.Skill) {
	interval := time.Duration(skill.IntervalMS) * time.Millisecond
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !r.paused.Load() {
				r.sendKey(uint16(skill.Key.VK))
			}
			timer.Reset(interval)
		}
	}
}

func runnableSkills(cfg config.Config) []config.Skill {
	skills := make([]config.Skill, 0, len(cfg.Skills))
	for _, skill := range cfg.Skills {
		if skill.Enabled && skill.Key.Assigned() {
			skills = append(skills, skill)
		}
	}
	return skills
}
