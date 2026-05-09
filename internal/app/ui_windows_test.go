//go:build windows

package app

import "testing"

func TestStatusTextColorReflectsRuntimeState(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*application)
		want      uintptr
	}{
		{
			name: "stopped",
			want: uiStatusStopped,
		},
		{
			name: "runner running",
			configure: func(a *application) {
				a.runner.running.Store(true)
			},
			want: uiStatusRunning,
		},
		{
			name: "clicker running",
			configure: func(a *application) {
				a.clicker.running.Store(true)
			},
			want: uiStatusRunning,
		},
		{
			name: "runner paused",
			configure: func(a *application) {
				a.runner.running.Store(true)
				a.runner.paused.Store(true)
			},
			want: uiStatusPaused,
		},
		{
			name: "clicker paused",
			configure: func(a *application) {
				a.clicker.running.Store(true)
				a.clicker.paused.Store(true)
			},
			want: uiStatusPaused,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApplication()
			if tt.configure != nil {
				tt.configure(a)
			}
			if got := a.statusTextColor(); got != tt.want {
				t.Fatalf("statusTextColor() = %#x, want %#x", got, tt.want)
			}
		})
	}
}
