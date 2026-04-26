package config

import (
	"strings"
	"testing"
)

func TestKeyBindingAssigned(t *testing.T) {
	tests := []struct {
		name    string
		binding KeyBinding
		want    bool
	}{
		{name: "zero vk is unassigned", binding: KeyBinding{Name: "A", VK: 0}, want: false},
		{name: "positive vk is assigned", binding: KeyBinding{Name: "A", VK: 65}, want: true},
		{name: "negative vk is unassigned", binding: KeyBinding{Name: "Bad", VK: -1}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.binding.Assigned(); got != tt.want {
				t.Fatalf("Assigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Default().Validate() error = %v", err)
	}
	if cfg.Pause != (KeyBinding{Name: "Mouse Right", VK: 0x02}) {
		t.Fatalf("pause = %+v, want Mouse Right", cfg.Pause)
	}
	if cfg.Menu.Inventory != (KeyBinding{Name: "C", VK: 0x43}) {
		t.Fatalf("inventory = %+v, want C", cfg.Menu.Inventory)
	}
	if cfg.Menu.WorldMap != (KeyBinding{Name: "M", VK: 0x4D}) {
		t.Fatalf("world map = %+v, want M", cfg.Menu.WorldMap)
	}
	if cfg.Menu.Whisper != (KeyBinding{Name: "R", VK: 0x52}) {
		t.Fatalf("whisper = %+v, want R", cfg.Menu.Whisper)
	}
	if len(cfg.Skills) != MaxSkills {
		t.Fatalf("skills length = %d, want %d", len(cfg.Skills), MaxSkills)
	}
	for i, skill := range cfg.Skills {
		wantName := "Skill " + string(rune('1'+i))
		if skill.Name != wantName {
			t.Fatalf("skill %d name = %q, want %q", i+1, skill.Name, wantName)
		}
		if skill.IntervalMS != DefaultIntervalMS {
			t.Fatalf("skill %d interval = %d, want %d", i+1, skill.IntervalMS, DefaultIntervalMS)
		}
		if skill.Enabled != DefaultSkillEnabled {
			t.Fatalf("skill %d enabled = %v, want %v", i+1, skill.Enabled, DefaultSkillEnabled)
		}
		if skill.Key.Assigned() {
			t.Fatalf("skill %d key = %+v, want unassigned", i+1, skill.Key)
		}
	}
}

func TestNormalizeRepairsConfigShapeAndValues(t *testing.T) {
	cfg := Config{
		Start: KeyBinding{Name: "Bad Start", VK: 300},
		Stop:  KeyBinding{Name: "No Code", VK: 0},
		Pause: KeyBinding{Name: "Bad Pause", VK: -1},
		Menu: MenuKeys{
			Inventory: KeyBinding{Name: "Bad Menu", VK: 999},
		},
	}
	for i := 0; i < MaxSkills+2; i++ {
		cfg.Skills = append(cfg.Skills, Skill{
			Name:       "",
			Key:        KeyBinding{Name: "Bad Skill", VK: -10},
			IntervalMS: 1,
			Enabled:    true,
		})
	}

	cfg.Normalize()

	if len(cfg.Skills) != MaxSkills {
		t.Fatalf("skills length = %d, want %d", len(cfg.Skills), MaxSkills)
	}
	for i, skill := range cfg.Skills {
		wantName := "Skill " + string(rune('1'+i))
		if skill.Name != wantName {
			t.Fatalf("skill %d name = %q, want %q", i+1, skill.Name, wantName)
		}
		if skill.IntervalMS != DefaultIntervalMS {
			t.Fatalf("skill %d interval = %d, want %d", i+1, skill.IntervalMS, DefaultIntervalMS)
		}
		if skill.Key != (KeyBinding{}) {
			t.Fatalf("skill %d key = %+v, want cleared", i+1, skill.Key)
		}
	}
	if cfg.Start != (KeyBinding{}) {
		t.Fatalf("start = %+v, want cleared", cfg.Start)
	}
	if cfg.Stop != (KeyBinding{}) {
		t.Fatalf("stop = %+v, want cleared", cfg.Stop)
	}
	if cfg.Pause != (KeyBinding{}) {
		t.Fatalf("pause = %+v, want cleared", cfg.Pause)
	}
	if cfg.Menu.Inventory != (KeyBinding{}) {
		t.Fatalf("inventory = %+v, want cleared", cfg.Menu.Inventory)
	}
}

func TestMenuBindingsOrderLabelsAndValues(t *testing.T) {
	cfg := Default()
	got := cfg.MenuBindings()
	want := []MenuBinding{
		{ID: "inventory", Label: "Inventory", Binding: cfg.Menu.Inventory},
		{ID: "skills", Label: "Skills", Binding: cfg.Menu.Skills},
		{ID: "follower", Label: "Follower", Binding: cfg.Menu.Follower},
		{ID: "map", Label: "Map", Binding: cfg.Menu.Map},
		{ID: "world_map", Label: "World Map", Binding: cfg.Menu.WorldMap},
		{ID: "town_portal", Label: "Town Portal", Binding: cfg.Menu.TownPortal},
		{ID: "chat", Label: "Chat", Binding: cfg.Menu.Chat},
		{ID: "whisper", Label: "Whisper", Binding: cfg.Menu.Whisper},
	}

	if len(got) != len(want) {
		t.Fatalf("MenuBindings() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MenuBindings()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestValidateRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Config)
		wantError string
	}{
		{
			name: "too many skills",
			mutate: func(cfg *Config) {
				cfg.Skills = append(cfg.Skills, Skill{Name: "Extra", IntervalMS: DefaultIntervalMS})
			},
			wantError: "skills must not exceed",
		},
		{
			name: "skill interval below minimum",
			mutate: func(cfg *Config) {
				cfg.Skills[0].IntervalMS = MinimumIntervalMS - 1
			},
			wantError: "interval must be at least",
		},
		{
			name: "skill key below range",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "Bad", VK: -1}
			},
			wantError: "between 0 and 255",
		},
		{
			name: "skill key above range",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "Bad", VK: 256}
			},
			wantError: "between 0 and 255",
		},
		{
			name: "skill key name without code",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "A", VK: 0}
			},
			wantError: "has a name but no virtual-key code",
		},
		{
			name: "top-level key name without code",
			mutate: func(cfg *Config) {
				cfg.Start = KeyBinding{Name: "F5", VK: 0}
			},
			wantError: "has a name but no virtual-key code",
		},
		{
			name: "pause key above range",
			mutate: func(cfg *Config) {
				cfg.Pause = KeyBinding{Name: "Bad", VK: 999}
			},
			wantError: "between 0 and 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}
