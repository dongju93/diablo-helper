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
	sendKey func(vk uint16) error
}

func newSkillRunner(sendKey func(vk uint16) error) *skillRunner {
	return &skillRunner{sendKey: sendKey}
}

func (r *skillRunner) Start(cfg config.Config) bool {
	cfg.NormalizeForUI()
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
	interval, ok := intervalDuration(skill.IntervalMS)
	if !ok {
		return
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !r.paused.Load() {
				if err := r.sendKey(uint16(skill.Key.VK)); err != nil {
					r.Stop()
					return
				}
			}
			timer.Reset(interval)
		}
	}
}

func runnableSkills(cfg config.Config) []config.Skill {
	skills := make([]config.Skill, 0, len(cfg.Skills))
	for _, skill := range cfg.Skills {
		if skillRunnable(skill) {
			skills = append(skills, skill)
		}
	}
	return skills
}

func skillRunnable(skill config.Skill) bool {
	_, ok := intervalDuration(skill.IntervalMS)
	return skill.Enabled && skill.Key.Assigned() && ok
}

type clickerRunner struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	running atomic.Bool
	paused  atomic.Bool
	sendKey func(vk uint16) error
}

func newClickerRunner(sendKey func(vk uint16) error) *clickerRunner {
	return &clickerRunner{sendKey: sendKey}
}

func (r *clickerRunner) Start(clicker config.Clicker) bool {
	if !clickerRunnable(clicker) {
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

	go r.run(ctx, clicker)
	return true
}

func (r *clickerRunner) Stop() bool {
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

func (r *clickerRunner) SetPaused(paused bool) {
	if !r.running.Load() {
		r.paused.Store(false)
		return
	}
	r.paused.Store(paused)
}

func (r *clickerRunner) Paused() bool {
	return r.running.Load() && r.paused.Load()
}

func (r *clickerRunner) Running() bool {
	return r.running.Load()
}

func (r *clickerRunner) run(ctx context.Context, clicker config.Clicker) {
	interval, ok := intervalDuration(clicker.IntervalMS)
	if !ok {
		return
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !r.paused.Load() {
				if err := r.sendKey(uint16(clicker.Key.VK)); err != nil {
					r.Stop()
					return
				}
			}
			timer.Reset(interval)
		}
	}
}

func clickerRunnable(clicker config.Clicker) bool {
	_, ok := intervalDuration(clicker.IntervalMS)
	return clicker.Key.Assigned() && ok
}

func intervalDuration(ms int) (time.Duration, bool) {
	if ms < config.MinimumIntervalMS || ms > config.MaximumIntervalMS || !config.MillisecondsFitDuration(ms) {
		return 0, false
	}
	return time.Duration(ms) * time.Millisecond, true
}
