//go:build windows

package app

import (
	"errors"
	"testing"
	"time"

	"github.com/dongju93/diablo-helper/internal/config"
)

func TestSkillRunnerStartStopState(t *testing.T) {
	runner := newSkillRunner(func(uint16) error { return nil })
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:       "Enabled",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	if !runner.Running() {
		t.Fatal("Running() = false, want true")
	}
	if runner.Start(cfg) {
		t.Fatal("second Start() = true, want false")
	}

	runner.SetPaused(true)
	if !runner.Paused() {
		t.Fatal("Paused() = false, want true")
	}

	if !runner.Stop() {
		t.Fatal("Stop() = false, want true")
	}
	if runner.Running() {
		t.Fatal("Running() = true after Stop(), want false")
	}
	if runner.Paused() {
		t.Fatal("Paused() = true after Stop(), want false")
	}
	if runner.Stop() {
		t.Fatal("second Stop() = true, want false")
	}

	runner.SetPaused(true)
	if runner.Paused() {
		t.Fatal("Paused() = true while stopped, want false")
	}
}

func TestSkillRunnerDoesNotStartWithoutRunnableSkills(t *testing.T) {
	runner := newSkillRunner(func(uint16) error { return nil })
	cfg := config.Default()

	if runner.Start(cfg) {
		t.Fatal("Start() = true for default disabled skills, want false")
	}
	if runner.Running() {
		t.Fatal("Running() = true without runnable skills, want false")
	}

	cfg.Skills[0] = config.Skill{
		Name:       "Enabled without key",
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}
	if runner.Start(cfg) {
		t.Fatal("Start() = true for enabled unassigned skill, want false")
	}

	cfg.Skills[0] = config.Skill{
		Name:       "Interval too large",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: config.MaximumIntervalMS + 1,
		Enabled:    true,
	}
	if runner.Start(cfg) {
		t.Fatal("Start() = true for too-large interval, want false")
	}
}

func TestRunnableSkillsFiltersEnabledAssignedSkills(t *testing.T) {
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:       "Runnable",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}
	cfg.Skills[1] = config.Skill{
		Name:       "Disabled",
		Key:        config.KeyBinding{Name: "2", VK: int('2')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    false,
	}
	cfg.Skills[2] = config.Skill{
		Name:       "Unassigned",
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}

	got := runnableSkills(cfg)
	if len(got) != 1 {
		t.Fatalf("runnableSkills() length = %d, want 1", len(got))
	}
	if got[0].Name != "Runnable" {
		t.Fatalf("runnable skill = %q, want Runnable", got[0].Name)
	}
}

func TestSkillRunnerSendsEnabledAssignedSkillsOnly(t *testing.T) {
	sent := make(chan uint16, 20)
	runner := newSkillRunner(func(vk uint16) error {
		sent <- vk
		return nil
	})
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:       "Enabled",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}
	cfg.Skills[1] = config.Skill{
		Name:       "Disabled",
		Key:        config.KeyBinding{Name: "2", VK: int('2')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    false,
	}
	cfg.Skills[2] = config.Skill{
		Name:       "Unassigned",
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	deadline := time.After(80 * time.Millisecond)
	received := 0
	for {
		select {
		case vk := <-sent:
			received++
			if vk != '1' {
				t.Fatalf("sent key = %d, want only %d", vk, '1')
			}
		case <-deadline:
			if received == 0 {
				t.Fatal("received no key sends")
			}
			return
		}
	}
}

func TestSkillRunnerPauseSuppressesAndResumes(t *testing.T) {
	sent := make(chan uint16, 20)
	runner := newSkillRunner(func(vk uint16) error {
		sent <- vk
		return nil
	})
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:       "Enabled",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: 20,
		Enabled:    true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	expectKey(t, sent, '1')
	runner.SetPaused(true)
	drainKeys(sent)

	time.Sleep(60 * time.Millisecond)
	select {
	case vk := <-sent:
		t.Fatalf("received key %d while paused", vk)
	default:
	}

	runner.SetPaused(false)
	expectKey(t, sent, '1')
}

func TestSkillRunnerStopWaitsForSendAndBlocksRestart(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	runner := newSkillRunner(func(uint16) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return nil
	})
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:       "Enabled",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for sendKey")
	}

	stopDone := make(chan bool, 1)
	go func() {
		stopDone <- runner.Stop()
	}()
	waitForRunnerCondition(t, func() bool {
		return !runner.Running()
	})
	select {
	case got := <-stopDone:
		t.Fatalf("Stop() returned %t before sendKey completed", got)
	default:
	}
	if runner.Start(cfg) {
		t.Fatal("Start() = true while previous runner goroutine is stopping")
	}

	close(release)
	select {
	case got := <-stopDone:
		if !got {
			t.Fatal("Stop() = false, want true")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for Stop()")
	}
	if !runner.Start(cfg) {
		t.Fatal("Start() = false after Stop() completed, want true")
	}
	defer runner.Stop()
}

func TestSkillRunnerReportsSendKeyError(t *testing.T) {
	wantErr := errors.New("send failed")
	reported := make(chan error, 1)
	runner := newSkillRunner(func(uint16) error {
		return wantErr
	})
	runner.SetErrorHandler(func(err error) {
		reported <- err
	})
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:       "Enabled",
		Key:        config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS: config.MinimumIntervalMS,
		Enabled:    true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	select {
	case got := <-reported:
		if !errors.Is(got, wantErr) {
			t.Fatalf("reported error = %v, want %v", got, wantErr)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for sendKey error")
	}
	if runner.Running() {
		t.Fatal("Running() = true after sendKey error, want false")
	}
	waitForRunnerCondition(t, func() bool {
		runner.mu.Lock()
		defer runner.mu.Unlock()
		return runner.cancel == nil
	})
}

func TestClickerRunnerStartStopState(t *testing.T) {
	runner := newClickerRunner(func(uint16) error { return nil })
	clicker := config.Clicker{
		Key:        config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS: config.MinimumIntervalMS,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	if !runner.Running() {
		t.Fatal("Running() = false, want true")
	}
	if runner.Start(clicker) {
		t.Fatal("second Start() = true, want false")
	}

	runner.SetPaused(true)
	if !runner.Paused() {
		t.Fatal("Paused() = false, want true")
	}

	if !runner.Stop() {
		t.Fatal("Stop() = false, want true")
	}
	if runner.Running() {
		t.Fatal("Running() = true after Stop(), want false")
	}
	if runner.Paused() {
		t.Fatal("Paused() = true after Stop(), want false")
	}
	if runner.Stop() {
		t.Fatal("second Stop() = true, want false")
	}

	runner.SetPaused(true)
	if runner.Paused() {
		t.Fatal("Paused() = true while stopped, want false")
	}
}

func TestClickerRunnerPauseSuppressesAndResumes(t *testing.T) {
	sent := make(chan uint16, 20)
	runner := newClickerRunner(func(vk uint16) error {
		sent <- vk
		return nil
	})
	clicker := config.Clicker{
		Key:        config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS: 20,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	expectKey(t, sent, vkLButton)
	runner.SetPaused(true)
	drainKeys(sent)

	time.Sleep(60 * time.Millisecond)
	select {
	case vk := <-sent:
		t.Fatalf("received key %d while clicker paused", vk)
	default:
	}

	runner.SetPaused(false)
	expectKey(t, sent, vkLButton)
}

func TestClickerRunnerDoesNotStartWithoutRunnableKey(t *testing.T) {
	runner := newClickerRunner(func(uint16) error { return nil })

	if runner.Start(config.Clicker{IntervalMS: config.MinimumIntervalMS}) {
		t.Fatal("Start() = true for unassigned clicker key, want false")
	}
	if runner.Start(config.Clicker{Key: config.KeyBinding{Name: "Mouse Left", VK: vkLButton}, IntervalMS: config.MinimumIntervalMS - 1}) {
		t.Fatal("Start() = true for too-small interval, want false")
	}
	if runner.Start(config.Clicker{Key: config.KeyBinding{Name: "Mouse Left", VK: vkLButton}, IntervalMS: config.MaximumIntervalMS + 1}) {
		t.Fatal("Start() = true for too-large interval, want false")
	}
}

func TestClickerRunnerSendsConfiguredKey(t *testing.T) {
	sent := make(chan uint16, 20)
	runner := newClickerRunner(func(vk uint16) error {
		sent <- vk
		return nil
	})
	clicker := config.Clicker{
		Key:        config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS: config.MinimumIntervalMS,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	expectKey(t, sent, vkLButton)
}

func TestClickerRunnerStopWaitsForSendAndBlocksRestart(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	runner := newClickerRunner(func(uint16) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return nil
	})
	clicker := config.Clicker{
		Key:        config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS: config.MinimumIntervalMS,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for sendKey")
	}

	stopDone := make(chan bool, 1)
	go func() {
		stopDone <- runner.Stop()
	}()
	waitForRunnerCondition(t, func() bool {
		return !runner.Running()
	})
	select {
	case got := <-stopDone:
		t.Fatalf("Stop() returned %t before sendKey completed", got)
	default:
	}
	if runner.Start(clicker) {
		t.Fatal("Start() = true while previous clicker goroutine is stopping")
	}

	close(release)
	select {
	case got := <-stopDone:
		if !got {
			t.Fatal("Stop() = false, want true")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for Stop()")
	}
	if !runner.Start(clicker) {
		t.Fatal("Start() = false after Stop() completed, want true")
	}
	defer runner.Stop()
}

func TestClickerRunnerReportsSendKeyError(t *testing.T) {
	wantErr := errors.New("send failed")
	reported := make(chan error, 1)
	runner := newClickerRunner(func(uint16) error {
		return wantErr
	})
	runner.SetErrorHandler(func(err error) {
		reported <- err
	})
	clicker := config.Clicker{
		Key:        config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS: config.MinimumIntervalMS,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	select {
	case got := <-reported:
		if !errors.Is(got, wantErr) {
			t.Fatalf("reported error = %v, want %v", got, wantErr)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for sendKey error")
	}
	if runner.Running() {
		t.Fatal("Running() = true after sendKey error, want false")
	}
	waitForRunnerCondition(t, func() bool {
		runner.mu.Lock()
		defer runner.mu.Unlock()
		return runner.cancel == nil
	})
}

func expectKey(t *testing.T, ch <-chan uint16, want uint16) {
	t.Helper()

	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("received key = %d, want %d", got, want)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for key %d", want)
	}
}

func waitForRunnerCondition(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for runner condition")
}

func drainKeys(ch <-chan uint16) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
