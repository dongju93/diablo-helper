//go:build windows

package app

import (
	"errors"
	"testing"
	"time"

	"github.com/dongju93/diablo-helper/internal/config"
)

func TestSkillRunnerStartStopState(t *testing.T) {
	runner := newSkillRunner(func(uint16, time.Duration) error { return nil })
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

func TestSkillRunnerStopReleasesActiveOutputKeys(t *testing.T) {
	var released []uint16
	runner := newSkillRunnerWithRelease(
		func(uint16, time.Duration) error { return nil },
		func(vk uint16) error {
			released = append(released, vk)
			return nil
		},
	)
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:        "Mouse skill",
		Key:         config.KeyBinding{Name: "Mouse Right", VK: vkRButton},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
	}
	cfg.Skills[1] = config.Skill{
		Name:        "Keyboard skill",
		Key:         config.KeyBinding{Name: "Q", VK: int('Q')},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	if !runner.Stop() {
		t.Fatal("Stop() = false, want true")
	}

	assertReleasedKeys(t, released, []uint16{vkRButton, 'Q'})
}

func TestSkillRunnerDoesNotStartWithoutRunnableSkills(t *testing.T) {
	runner := newSkillRunner(func(uint16, time.Duration) error { return nil })
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

	cfg = config.Default()
	cfg.Skills[0] = config.Skill{
		Name:        "Invalid hold",
		Key:         config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.MaximumInputHoldMS + 1,
		Enabled:     true,
	}
	if runner.Start(cfg) {
		t.Fatal("Start() = true for too-large input hold, want false")
	}
}

func TestRunnableSkillsFiltersEnabledAssignedSkills(t *testing.T) {
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:        "Runnable",
		Key:         config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
	}
	cfg.Skills[1] = config.Skill{
		Name:        "Disabled",
		Key:         config.KeyBinding{Name: "2", VK: int('2')},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     false,
	}
	cfg.Skills[2] = config.Skill{
		Name:        "Unassigned",
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
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
	runner := newSkillRunner(func(vk uint16, _ time.Duration) error {
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

func TestSkillRunnerSendsConfiguredInputHold(t *testing.T) {
	sent := make(chan time.Duration, 20)
	runner := newSkillRunner(func(_ uint16, hold time.Duration) error {
		sent <- hold
		return nil
	})
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:        "Enabled",
		Key:         config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: 37,
		Enabled:     true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	expectHold(t, sent, 37*time.Millisecond)
}

func TestSkillRunnerSerializesSkillSends(t *testing.T) {
	entered := make(chan uint16, 2)
	releaseFirst := make(chan struct{})
	runner := newSkillRunner(func(vk uint16, _ time.Duration) error {
		entered <- vk
		if vk == '1' {
			<-releaseFirst
		}
		return nil
	})
	cfg := config.Default()
	cfg.Skills[0] = config.Skill{
		Name:        "First",
		Key:         config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
	}
	cfg.Skills[1] = config.Skill{
		Name:        "Second",
		Key:         config.KeyBinding{Name: "2", VK: int('2')},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	first := expectKey(t, entered, '1')
	if first != '1' {
		t.Fatalf("first sent key = %d, want %d", first, '1')
	}
	select {
	case vk := <-entered:
		t.Fatalf("received key %d before first send completed", vk)
	case <-time.After(30 * time.Millisecond):
	}
	close(releaseFirst)
	second := expectKey(t, entered, '2')
	if second != '2' {
		t.Fatalf("second sent key = %d, want %d", second, '2')
	}
	if !runner.Stop() {
		t.Fatal("Stop() = false, want true")
	}
}

func TestSkillRunnerAppliesSkillGapToActualStartTimes(t *testing.T) {
	sent := make(chan struct {
		vk uint16
		at time.Time
	}, 4)
	runner := newSkillRunnerWithTimedSend(func(vk uint16, _ time.Duration) (time.Time, error) {
		startedAt := time.Now()
		sent <- struct {
			vk uint16
			at time.Time
		}{vk: vk, at: startedAt}
		return startedAt, nil
	}, nil)
	cfg := config.Default()
	cfg.SkillGapMS = 30
	cfg.Skills[0] = config.Skill{
		Name:        "First",
		Key:         config.KeyBinding{Name: "1", VK: int('1')},
		IntervalMS:  200,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
	}
	cfg.Skills[1] = config.Skill{
		Name:        "Second",
		Key:         config.KeyBinding{Name: "2", VK: int('2')},
		IntervalMS:  200,
		InputHoldMS: config.DefaultInputHoldMS,
		Enabled:     true,
	}

	if !runner.Start(cfg) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	first := expectSentKey(t, sent, '1')
	second := expectSentKey(t, sent, '2')
	if gap := second.at.Sub(first.at); gap < 20*time.Millisecond {
		t.Fatalf("actual start gap = %v, want at least 20ms", gap)
	}
}

func TestSkillRunnerPauseSuppressesAndResumes(t *testing.T) {
	sent := make(chan uint16, 20)
	runner := newSkillRunner(func(vk uint16, _ time.Duration) error {
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
	runner := newSkillRunner(func(uint16, time.Duration) error {
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
	runner := newSkillRunner(func(uint16, time.Duration) error {
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
	runner := newClickerRunner(func(uint16, time.Duration) error { return nil })
	clicker := config.Clicker{
		Key:         config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
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

func TestClickerRunnerStopReleasesActiveOutputKey(t *testing.T) {
	var released []uint16
	runner := newClickerRunnerWithRelease(
		func(uint16, time.Duration) error { return nil },
		func(vk uint16) error {
			released = append(released, vk)
			return nil
		},
	)
	clicker := config.Clicker{
		Key:         config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	if !runner.Stop() {
		t.Fatal("Stop() = false, want true")
	}

	assertReleasedKeys(t, released, []uint16{vkLButton})
}

func TestClickerRunnerPauseSuppressesAndResumes(t *testing.T) {
	sent := make(chan uint16, 20)
	runner := newClickerRunner(func(vk uint16, _ time.Duration) error {
		sent <- vk
		return nil
	})
	clicker := config.Clicker{
		Key:         config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS:  20,
		InputHoldMS: config.DefaultInputHoldMS,
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
	runner := newClickerRunner(func(uint16, time.Duration) error { return nil })

	if runner.Start(config.Clicker{IntervalMS: config.MinimumIntervalMS, InputHoldMS: config.DefaultInputHoldMS}) {
		t.Fatal("Start() = true for unassigned clicker key, want false")
	}
	if runner.Start(config.Clicker{Key: config.KeyBinding{Name: "Mouse Left", VK: vkLButton}, IntervalMS: config.MinimumIntervalMS - 1, InputHoldMS: config.DefaultInputHoldMS}) {
		t.Fatal("Start() = true for too-small interval, want false")
	}
	if runner.Start(config.Clicker{Key: config.KeyBinding{Name: "Mouse Left", VK: vkLButton}, IntervalMS: config.MaximumIntervalMS + 1, InputHoldMS: config.DefaultInputHoldMS}) {
		t.Fatal("Start() = true for too-large interval, want false")
	}
	if runner.Start(config.Clicker{Key: config.KeyBinding{Name: "Mouse Left", VK: vkLButton}, IntervalMS: config.MinimumIntervalMS, InputHoldMS: config.MaximumInputHoldMS + 1}) {
		t.Fatal("Start() = true for too-large input hold, want false")
	}
}

func TestClickerRunnerSendsConfiguredKey(t *testing.T) {
	sent := make(chan uint16, 20)
	runner := newClickerRunner(func(vk uint16, _ time.Duration) error {
		sent <- vk
		return nil
	})
	clicker := config.Clicker{
		Key:         config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	expectKey(t, sent, vkLButton)
}

func TestClickerRunnerSendsConfiguredInputHold(t *testing.T) {
	sent := make(chan time.Duration, 20)
	runner := newClickerRunner(func(_ uint16, hold time.Duration) error {
		sent <- hold
		return nil
	})
	clicker := config.Clicker{
		Key:         config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: 42,
	}

	if !runner.Start(clicker) {
		t.Fatal("Start() = false, want true")
	}
	defer runner.Stop()

	expectHold(t, sent, 42*time.Millisecond)
}

func TestClickerRunnerStopWaitsForSendAndBlocksRestart(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	runner := newClickerRunner(func(uint16, time.Duration) error {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		return nil
	})
	clicker := config.Clicker{
		Key:         config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
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
	runner := newClickerRunner(func(uint16, time.Duration) error {
		return wantErr
	})
	runner.SetErrorHandler(func(err error) {
		reported <- err
	})
	clicker := config.Clicker{
		Key:         config.KeyBinding{Name: "Mouse Left", VK: vkLButton},
		IntervalMS:  config.MinimumIntervalMS,
		InputHoldMS: config.DefaultInputHoldMS,
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

func TestNextScheduledInputPreservesIntervalCadenceAfterSendDuration(t *testing.T) {
	const (
		interval = 100 * time.Millisecond
		sendTime = 30 * time.Millisecond
	)
	before := time.Now()
	previousScheduledStart := before.Add(-sendTime)

	next := nextScheduledInput(previousScheduledStart, interval)
	if !next.After(before) {
		t.Fatalf("nextScheduledInput() = %v, want after %v", next, before)
	}
	if !next.Before(before.Add(interval - sendTime/2)) {
		t.Fatalf("nextScheduledInput() = %v, want cadence relative to previous start, not now + interval", next)
	}
}

func TestNextScheduledInputResetsAfterMissedIntervalsWithoutBursting(t *testing.T) {
	const interval = 100 * time.Millisecond
	before := time.Now()
	previousScheduledStart := before.Add(-250 * time.Millisecond)

	next := nextScheduledInput(previousScheduledStart, interval)
	if !next.After(before) {
		t.Fatalf("nextScheduledInput() = %v, want after %v", next, before)
	}
	if next.Before(before.Add(interval)) {
		t.Fatalf("nextScheduledInput() = %v, want a fresh interval after missed schedules", next)
	}
}

func expectKey(t *testing.T, ch <-chan uint16, want uint16) uint16 {
	t.Helper()

	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("received key = %d, want %d", got, want)
		}
		return got
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for key %d", want)
	}
	return 0
}

func expectSentKey(t *testing.T, ch <-chan struct {
	vk uint16
	at time.Time
}, want uint16) struct {
	vk uint16
	at time.Time
} {
	t.Helper()

	select {
	case got := <-ch:
		if got.vk != want {
			t.Fatalf("received key = %d, want %d", got.vk, want)
		}
		return got
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for key %d", want)
	}
	return struct {
		vk uint16
		at time.Time
	}{}
}

func expectHold(t *testing.T, ch <-chan time.Duration, want time.Duration) {
	t.Helper()

	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("received hold = %v, want %v", got, want)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for hold %v", want)
	}
}

func assertReleasedKeys(t *testing.T, got []uint16, want []uint16) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("released keys = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("released keys = %v, want %v", got, want)
		}
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
