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
	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	stopping bool
	running  atomic.Bool
	paused   atomic.Bool
	sendKey  contextTimedKeySender
	release  func(vk uint16) error
	onError  func(error)
	active   []uint16
}

func newSkillRunner(sendKey func(vk uint16, hold time.Duration) error) *skillRunner {
	return newSkillRunnerWithRelease(sendKey, nil)
}

func newSkillRunnerWithRelease(sendKey func(vk uint16, hold time.Duration) error, release func(vk uint16) error) *skillRunner {
	return newSkillRunnerWithTimedSend(wrapTimedKeySender(sendKey), release)
}

func newSkillRunnerWithTimedSend(sendKey timedKeySender, release func(vk uint16) error) *skillRunner {
	return newSkillRunnerWithContextTimedSend(func(_ context.Context, vk uint16, hold time.Duration) (time.Time, error) {
		return sendKey(vk, hold)
	}, release)
}

func newSkillRunnerWithContextTimedSend(sendKey contextTimedKeySender, release func(vk uint16) error) *skillRunner {
	return &skillRunner{sendKey: sendKey, release: release}
}

func (r *skillRunner) SetErrorHandler(onError func(error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onError = onError
}

func (r *skillRunner) Start(cfg config.Config) bool {
	cfg.NormalizeForUI()
	skills := runnableSkills(cfg)
	if len(skills) == 0 {
		return false
	}
	skillGap, ok := skillGapDuration(cfg.SkillGapMS)
	if !ok {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	r.cancel = cancel
	r.done = done
	r.stopping = false
	r.active = skillOutputKeys(skills)
	r.running.Store(true)
	r.paused.Store(false)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.run(ctx, skills, skillGap)
	}()
	go func() {
		wg.Wait()
		r.finish(done)
	}()
	return true
}

func (r *skillRunner) Stop() bool {
	r.mu.Lock()
	if r.cancel == nil {
		r.mu.Unlock()
		return false
	}
	if r.stopping {
		done := r.done
		r.mu.Unlock()
		<-done
		return false
	}
	cancel := r.cancel
	done := r.done
	release := r.release
	active := append([]uint16(nil), r.active...)
	r.stopping = true
	r.running.Store(false)
	r.paused.Store(false)
	r.mu.Unlock()

	cancel()
	<-done
	releaseOutputKeys(release, active)
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

func (r *skillRunner) run(ctx context.Context, skills []config.Skill, skillGap time.Duration) {
	scheduled := scheduledSkills(skills, skillGap)
	if len(scheduled) == 0 {
		return
	}
	timer := newStoppedTimer()
	defer timer.Stop()
	var lastStarted time.Time

	for {
		index := nextScheduledSkillIndex(scheduled)
		next := scheduled[index].next
		if !lastStarted.IsZero() && skillGap > 0 {
			minNext := lastStarted.Add(skillGap)
			if next.Before(minNext) {
				next = minNext
			}
		}
		if !waitForScheduledInput(ctx, timer, next) {
			return
		}
		if ctx.Err() != nil {
			return
		}
		if r.paused.Load() {
			scheduled[index].next = time.Now().Add(scheduled[index].interval)
			continue
		}
		if ctx.Err() != nil {
			return
		}
		startedAt, err := r.sendKey(ctx, uint16(scheduled[index].skill.Key.VK), scheduled[index].hold)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			r.fail(err)
			return
		}
		if ctx.Err() != nil {
			return
		}
		lastStarted = startedAt
		scheduled[index].next = nextScheduledInput(startedAt, scheduled[index].interval)
	}
}

func (r *skillRunner) fail(err error) {
	r.mu.Lock()
	if r.cancel == nil || r.stopping {
		r.mu.Unlock()
		return
	}
	cancel := r.cancel
	onError := r.onError
	release := r.release
	active := append([]uint16(nil), r.active...)
	r.stopping = true
	r.running.Store(false)
	r.paused.Store(false)
	r.mu.Unlock()

	cancel()
	releaseOutputKeys(release, active)
	releaseInjectedInputs()
	if onError != nil {
		go onError(err)
	}
}

func (r *skillRunner) finish(done chan struct{}) {
	r.mu.Lock()
	if r.done == done {
		r.cancel = nil
		r.done = nil
		r.stopping = false
		r.active = nil
		r.running.Store(false)
		r.paused.Store(false)
	}
	close(done)
	r.mu.Unlock()
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
	_, intervalOK := intervalDuration(skill.IntervalMS)
	_, holdOK := inputHoldDuration(skill.InputHoldMS)
	return skill.Enabled && skill.Key.Assigned() && intervalOK && holdOK
}

type clickerRunner struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	stopping bool
	running  atomic.Bool
	paused   atomic.Bool
	sendKey  contextTimedKeySender
	release  func(vk uint16) error
	onError  func(error)
	active   uint16
}

func newClickerRunner(sendKey func(vk uint16, hold time.Duration) error) *clickerRunner {
	return newClickerRunnerWithRelease(sendKey, nil)
}

func newClickerRunnerWithRelease(sendKey func(vk uint16, hold time.Duration) error, release func(vk uint16) error) *clickerRunner {
	return newClickerRunnerWithTimedSend(wrapTimedKeySender(sendKey), release)
}

func newClickerRunnerWithTimedSend(sendKey timedKeySender, release func(vk uint16) error) *clickerRunner {
	return newClickerRunnerWithContextTimedSend(func(_ context.Context, vk uint16, hold time.Duration) (time.Time, error) {
		return sendKey(vk, hold)
	}, release)
}

func newClickerRunnerWithContextTimedSend(sendKey contextTimedKeySender, release func(vk uint16) error) *clickerRunner {
	return &clickerRunner{sendKey: sendKey, release: release}
}

func (r *clickerRunner) SetErrorHandler(onError func(error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onError = onError
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
	done := make(chan struct{})
	r.cancel = cancel
	r.done = done
	r.stopping = false
	r.active = uint16(clicker.Key.VK)
	r.running.Store(true)
	r.paused.Store(false)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.run(ctx, clicker)
	}()
	go func() {
		wg.Wait()
		r.finish(done)
	}()
	return true
}

func (r *clickerRunner) Stop() bool {
	r.mu.Lock()
	if r.cancel == nil {
		r.mu.Unlock()
		return false
	}
	if r.stopping {
		done := r.done
		r.mu.Unlock()
		<-done
		return false
	}
	cancel := r.cancel
	done := r.done
	release := r.release
	active := r.active
	r.stopping = true
	r.running.Store(false)
	r.paused.Store(false)
	r.mu.Unlock()

	cancel()
	<-done
	releaseOutputKeys(release, []uint16{active})
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
	hold, ok := inputHoldDuration(clicker.InputHoldMS)
	if !ok {
		return
	}
	timer := newStoppedTimer()
	defer timer.Stop()
	next := time.Now()

	for {
		if !waitForScheduledInput(ctx, timer, next) {
			return
		}
		if ctx.Err() != nil {
			return
		}
		if r.paused.Load() {
			next = time.Now().Add(interval)
			continue
		}
		if ctx.Err() != nil {
			return
		}
		startedAt, err := r.sendKey(ctx, uint16(clicker.Key.VK), hold)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			r.fail(err)
			return
		}
		if ctx.Err() != nil {
			return
		}
		next = nextScheduledInput(startedAt, interval)
	}
}

func (r *clickerRunner) fail(err error) {
	r.mu.Lock()
	if r.cancel == nil || r.stopping {
		r.mu.Unlock()
		return
	}
	cancel := r.cancel
	onError := r.onError
	release := r.release
	active := r.active
	r.stopping = true
	r.running.Store(false)
	r.paused.Store(false)
	r.mu.Unlock()

	cancel()
	releaseOutputKeys(release, []uint16{active})
	releaseInjectedInputs()
	if onError != nil {
		go onError(err)
	}
}

func (r *clickerRunner) finish(done chan struct{}) {
	r.mu.Lock()
	if r.done == done {
		r.cancel = nil
		r.done = nil
		r.stopping = false
		r.active = 0
		r.running.Store(false)
		r.paused.Store(false)
	}
	close(done)
	r.mu.Unlock()
}

func clickerRunnable(clicker config.Clicker) bool {
	_, intervalOK := intervalDuration(clicker.IntervalMS)
	_, holdOK := inputHoldDuration(clicker.InputHoldMS)
	return clicker.Key.Assigned() && intervalOK && holdOK
}

func skillOutputKeys(skills []config.Skill) []uint16 {
	keys := make([]uint16, 0, len(skills))
	var seen [256]bool
	for _, skill := range skills {
		vk := skill.Key.VK
		if vk <= 0 || vk > 255 || seen[vk] {
			continue
		}
		seen[vk] = true
		keys = append(keys, uint16(vk))
	}
	return keys
}

func releaseOutputKeys(release func(vk uint16) error, keys []uint16) {
	if release == nil {
		return
	}
	for _, vk := range keys {
		if vk == 0 {
			continue
		}
		_ = release(vk)
	}
}

func stopRuntimeRunners(runner *skillRunner, clicker *clickerRunner) bool {
	stopped := make(chan bool, 2)
	var wg sync.WaitGroup
	if runner != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stopped <- runner.Stop()
		}()
	}
	if clicker != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stopped <- clicker.Stop()
		}()
	}
	wg.Wait()
	close(stopped)

	stoppedAny := false
	for result := range stopped {
		stoppedAny = result || stoppedAny
	}
	return stoppedAny
}

type timedKeySender func(vk uint16, hold time.Duration) (time.Time, error)

type contextTimedKeySender func(ctx context.Context, vk uint16, hold time.Duration) (time.Time, error)

func wrapTimedKeySender(sendKey func(vk uint16, hold time.Duration) error) timedKeySender {
	return func(vk uint16, hold time.Duration) (time.Time, error) {
		startedAt := time.Now()
		return startedAt, sendKey(vk, hold)
	}
}

type scheduledSkill struct {
	skill    config.Skill
	interval time.Duration
	hold     time.Duration
	next     time.Time
	index    int
}

func scheduledSkills(skills []config.Skill, skillGap time.Duration) []scheduledSkill {
	now := time.Now()
	scheduled := make([]scheduledSkill, 0, len(skills))
	for i, skill := range skills {
		interval, intervalOK := intervalDuration(skill.IntervalMS)
		hold, holdOK := inputHoldDuration(skill.InputHoldMS)
		if !intervalOK || !holdOK {
			continue
		}
		scheduled = append(scheduled, scheduledSkill{
			skill:    skill,
			interval: interval,
			hold:     hold,
			next:     now.Add(time.Duration(i) * skillGap),
			index:    i,
		})
	}
	return scheduled
}

func nextScheduledSkillIndex(skills []scheduledSkill) int {
	next := 0
	for i := 1; i < len(skills); i++ {
		if skills[i].next.Before(skills[next].next) {
			next = i
			continue
		}
		if skills[i].next.Equal(skills[next].next) && skills[i].index < skills[next].index {
			next = i
		}
	}
	return next
}

func intervalDuration(ms int) (time.Duration, bool) {
	if ms < config.MinimumIntervalMS || ms > config.MaximumIntervalMS || !config.MillisecondsFitDuration(ms) {
		return 0, false
	}
	return time.Duration(ms) * time.Millisecond, true
}

func skillGapDuration(ms int) (time.Duration, bool) {
	if ms < 0 || ms > config.MaximumSkillGapMS || !config.MillisecondsFitDuration(ms) {
		return 0, false
	}
	return time.Duration(ms) * time.Millisecond, true
}

func inputHoldDuration(ms int) (time.Duration, bool) {
	if ms < config.MinimumInputHoldMS || ms > config.MaximumInputHoldMS || !config.MillisecondsFitDuration(ms) {
		return 0, false
	}
	return time.Duration(ms) * time.Millisecond, true
}

func waitForScheduledInput(ctx context.Context, timer *time.Timer, next time.Time) bool {
	delay := time.Until(next)
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer.Reset(delay)
	select {
	case <-ctx.Done():
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		return false
	case <-timer.C:
		return true
	}
}

func nextScheduledInput(previous time.Time, interval time.Duration) time.Time {
	next := previous.Add(interval)
	now := time.Now()
	if !next.After(now) {
		return now.Add(interval)
	}
	return next
}

func newStoppedTimer() *time.Timer {
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	return timer
}
