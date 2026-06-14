//go:build windows

package app

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dongju93/diablo-helper/internal/config"
)

type runnerCore struct {
	name     string
	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	stopping bool
	running  atomic.Bool
	paused   atomic.Bool
	release  func(vk uint16) error
	onError  func(error)
	active   []uint16
}

func newRunnerCore(name string, release func(vk uint16) error) runnerCore {
	return runnerCore{name: name, release: release}
}

func (r *runnerCore) SetErrorHandler(onError func(error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onError = onError
}

func (r *runnerCore) start(parent context.Context, active []uint16, loop func(context.Context) error) bool {
	if parent == nil {
		parent = context.Background()
	}
	if parent.Err() != nil {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return false
	}

	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	r.cancel = cancel
	r.done = done
	r.stopping = false
	r.active = append([]uint16(nil), active...)
	r.running.Store(true)
	r.paused.Store(false)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := loop(ctx); err != nil && ctx.Err() == nil {
			r.fail(err)
		}
	}()
	go func() {
		wg.Wait()
		r.finish(done)
	}()
	return true
}

func (r *runnerCore) Stop() bool {
	stop := r.RequestStop()
	if stop.done == nil {
		return false
	}
	<-stop.done
	stop.releaseActive()
	return stop.requested
}

func (r *runnerCore) RequestStop() runtimeStopHandle {
	r.mu.Lock()
	if r.cancel == nil {
		r.mu.Unlock()
		return runtimeStopHandle{}
	}
	if r.stopping {
		done := r.done
		r.mu.Unlock()
		return runtimeStopHandle{name: r.name, done: done}
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
	return runtimeStopHandle{
		name:      r.name,
		requested: true,
		done:      done,
		release: func() {
			releaseOutputKeys(release, active)
		},
	}
}

func (r *runnerCore) SetPaused(paused bool) {
	if !r.running.Load() {
		r.paused.Store(false)
		return
	}
	r.paused.Store(paused)
}

func (r *runnerCore) Running() bool {
	return r.running.Load()
}

func (r *runnerCore) Paused() bool {
	return r.running.Load() && r.paused.Load()
}

func (r *runnerCore) fail(err error) {
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
	releaseMouseButtons()
	if onError != nil {
		go onError(err)
	}
}

func (r *runnerCore) finish(done chan struct{}) {
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

type skillRunner struct {
	runnerCore
	sendKey contextTimedKeySender
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
	return &skillRunner{
		runnerCore: newRunnerCore("skill runner", release),
		sendKey:    sendKey,
	}
}

func (r *skillRunner) Start(cfg config.Config) bool {
	return r.StartContext(context.Background(), cfg)
}

func (r *skillRunner) StartContext(parent context.Context, cfg config.Config) bool {
	cfg.NormalizeForUI()
	skills := runnableSkills(cfg)
	if len(skills) == 0 {
		return false
	}
	skillGap, ok := skillGapDuration(cfg.SkillGapMS)
	if !ok {
		return false
	}

	return r.start(parent, skillOutputKeys(skills), func(ctx context.Context) error {
		return r.run(ctx, skills, skillGap)
	})
}

func (r *skillRunner) run(ctx context.Context, skills []config.Skill, skillGap time.Duration) error {
	scheduled := scheduledSkills(skills, skillGap)
	if len(scheduled) == 0 {
		return nil
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
			return nil
		}
		if ctx.Err() != nil {
			return nil
		}
		if r.paused.Load() {
			scheduled[index].next = time.Now().Add(scheduled[index].interval)
			continue
		}
		if ctx.Err() != nil {
			return nil
		}
		startedAt, err := r.sendKey(ctx, uint16(scheduled[index].skill.Key.VK), scheduled[index].hold)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
		lastStarted = startedAt
		scheduled[index].next = nextScheduledInput(startedAt, scheduled[index].interval)
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
	_, intervalOK := intervalDuration(skill.IntervalMS)
	_, holdOK := inputHoldDuration(skill.InputHoldMS)
	return skill.Enabled && skill.Key.Assigned() && intervalOK && holdOK
}

type clickerRunner struct {
	runnerCore
	sendKey contextTimedKeySender
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
	return &clickerRunner{
		runnerCore: newRunnerCore("clicker runner", release),
		sendKey:    sendKey,
	}
}

func (r *clickerRunner) Start(clicker config.Clicker) bool {
	return r.StartContext(context.Background(), clicker)
}

func (r *clickerRunner) StartContext(parent context.Context, clicker config.Clicker) bool {
	if !clickerRunnable(clicker) {
		return false
	}

	return r.start(parent, []uint16{uint16(clicker.Key.VK)}, func(ctx context.Context) error {
		return r.run(ctx, clicker)
	})
}

func (r *clickerRunner) run(ctx context.Context, clicker config.Clicker) error {
	interval, ok := intervalDuration(clicker.IntervalMS)
	if !ok {
		return nil
	}
	hold, ok := inputHoldDuration(clicker.InputHoldMS)
	if !ok {
		return nil
	}
	timer := newStoppedTimer()
	defer timer.Stop()
	next := time.Now()

	for {
		if !waitForScheduledInput(ctx, timer, next) {
			return nil
		}
		if ctx.Err() != nil {
			return nil
		}
		if r.paused.Load() {
			next = time.Now().Add(interval)
			continue
		}
		if ctx.Err() != nil {
			return nil
		}
		startedAt, err := r.sendKey(ctx, uint16(clicker.Key.VK), hold)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
		next = nextScheduledInput(startedAt, interval)
	}
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
	handles := requestRuntimeRunnersStop(runner, clicker)
	return waitRuntimeStopHandles(handles)
}

type runtimeStopHandle struct {
	name      string
	requested bool
	done      <-chan struct{}
	release   func()
}

func (h runtimeStopHandle) releaseActive() {
	if h.release != nil {
		h.release()
	}
}

func requestRuntimeRunnersStop(runner *skillRunner, clicker *clickerRunner) []runtimeStopHandle {
	handles := make([]runtimeStopHandle, 0, 2)
	if runner != nil {
		if handle := runner.RequestStop(); handle.done != nil {
			handles = append(handles, handle)
		}
	}
	if clicker != nil {
		if handle := clicker.RequestStop(); handle.done != nil {
			handles = append(handles, handle)
		}
	}
	return handles
}

func waitRuntimeStopHandles(handles []runtimeStopHandle) bool {
	stopped := make(chan bool, len(handles))
	var wg sync.WaitGroup
	for _, handle := range handles {
		handle := handle
		wg.Add(1)
		go func() {
			defer wg.Done()
			if handle.done != nil {
				<-handle.done
			}
			handle.releaseActive()
			stopped <- handle.requested
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
