//go:build windows

package app

import (
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
	if !runner.Stop() {
		t.Fatal("Stop() = false, want true")
	}
	if runner.Running() {
		t.Fatal("Running() = true after Stop(), want false")
	}
	if runner.Stop() {
		t.Fatal("second Stop() = true, want false")
	}
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

func drainKeys(ch <-chan uint16) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
