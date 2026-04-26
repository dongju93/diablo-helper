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
	if cfg.SkillGapMS != DefaultSkillGapMS {
		t.Fatalf("skill gap = %d, want %d", cfg.SkillGapMS, DefaultSkillGapMS)
	}
	if cfg.Clicker.Key != (KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}) {
		t.Fatalf("clicker key = %+v, want Mouse Left", cfg.Clicker.Key)
	}
	if cfg.Clicker.IntervalMS != DefaultClickerIntervalMS {
		t.Fatalf("clicker interval = %d, want %d", cfg.Clicker.IntervalMS, DefaultClickerIntervalMS)
	}
	if cfg.Clicker.Start.Assigned() || cfg.Clicker.Stop.Assigned() {
		t.Fatalf("clicker start/stop = %+v/%+v, want unassigned", cfg.Clicker.Start, cfg.Clicker.Stop)
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
		SkillGapMS: -1,
		Clicker: Clicker{
			Start:      KeyBinding{Name: "Bad Clicker Start", VK: 999},
			Stop:       KeyBinding{Name: "Bad Clicker Stop", VK: -1},
			Key:        KeyBinding{Name: "Bad Clicker Key", VK: 300},
			IntervalMS: 1,
		},
	}
	for range MaxSkills + 2 {
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
	if cfg.SkillGapMS != DefaultSkillGapMS {
		t.Fatalf("skill gap = %d, want %d", cfg.SkillGapMS, DefaultSkillGapMS)
	}
	if cfg.Clicker.Start != (KeyBinding{}) || cfg.Clicker.Stop != (KeyBinding{}) || cfg.Clicker.Key != (KeyBinding{}) {
		t.Fatalf("clicker bindings = %+v, want cleared", cfg.Clicker)
	}
	if cfg.Clicker.IntervalMS != DefaultClickerIntervalMS {
		t.Fatalf("clicker interval = %d, want %d", cfg.Clicker.IntervalMS, DefaultClickerIntervalMS)
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
			name: "skill interval above maximum",
			mutate: func(cfg *Config) {
				cfg.Skills[0].IntervalMS = MaximumIntervalMS + 1
			},
			wantError: "interval must be at most",
		},
		{
			name: "skill gap below zero",
			mutate: func(cfg *Config) {
				cfg.SkillGapMS = -1
			},
			wantError: "skill gap must be at least",
		},
		{
			name: "skill gap above maximum",
			mutate: func(cfg *Config) {
				cfg.SkillGapMS = MaximumSkillGapMS + 1
			},
			wantError: "skill gap must be at most",
		},
		{
			name: "clicker interval below minimum",
			mutate: func(cfg *Config) {
				cfg.Clicker.IntervalMS = MinimumIntervalMS - 1
			},
			wantError: "clicker interval must be at least",
		},
		{
			name: "clicker interval above maximum",
			mutate: func(cfg *Config) {
				cfg.Clicker.IntervalMS = MaximumIntervalMS + 1
			},
			wantError: "clicker interval must be at most",
		},
		{
			name: "clicker start mouse left",
			mutate: func(cfg *Config) {
				cfg.Clicker.Start = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
			},
			wantError: "clicker start key must not be Mouse Left",
		},
		{
			name: "clicker stop mouse left",
			mutate: func(cfg *Config) {
				cfg.Clicker.Stop = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
			},
			wantError: "clicker stop key must not be Mouse Left",
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
			name: "key name contains nul",
			mutate: func(cfg *Config) {
				cfg.Start = KeyBinding{Name: "Bad\x00Name", VK: 0x41}
			},
			wantError: "must not contain NUL",
		},
		{
			name: "key name too long",
			mutate: func(cfg *Config) {
				cfg.Start = KeyBinding{Name: strings.Repeat("A", MaxKeyNameLength+1), VK: 0x41}
			},
			wantError: "must not exceed",
		},
		{
			name: "skill name contains control character",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Name = "Bad\nName"
			},
			wantError: "must not contain control characters",
		},
		{
			name: "skill name too long",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Name = strings.Repeat("A", MaxSkillNameLength+1)
			},
			wantError: "must not exceed",
		},
		{
			name: "pause key above range",
			mutate: func(cfg *Config) {
				cfg.Pause = KeyBinding{Name: "Bad", VK: 999}
			},
			wantError: "between 0 and 255",
		},
		{
			name: "key name spoofing",
			mutate: func(cfg *Config) {
				cfg.Start = KeyBinding{Name: "F12", VK: 0x0D}
			},
			wantError: "name does not match virtual-key code",
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

func TestKeyDisplayName(t *testing.T) {
	tests := []struct {
		name string
		vk   int
		want string
	}{
		{name: "digit", vk: '7', want: "7"},
		{name: "letter", vk: 'K', want: "K"},
		{name: "f1", vk: 0x70, want: "F1"},
		{name: "f24", vk: 0x87, want: "F24"},
		{name: "numpad", vk: 0x69, want: "Numpad 9"},
		{name: "mouse left", vk: 0x01, want: "Mouse Left"},
		{name: "mouse right", vk: 0x02, want: "Mouse Right"},
		{name: "enter", vk: 0x0D, want: "Enter"},
		{name: "space", vk: 0x20, want: "Space"},
		{name: "unknown", vk: 255, want: "VK_255"},
		{name: "negative", vk: -1, want: "VK_-1"},
		{name: "over 255", vk: 256, want: "VK_256"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KeyDisplayName(tt.vk); got != tt.want {
				t.Fatalf("KeyDisplayName(%d) = %q, want %q", tt.vk, got, tt.want)
			}
		})
	}
}

func TestNormalizeCanonicalizesKeyNames(t *testing.T) {
	cfg := Config{
		Start: KeyBinding{Name: "Spoofed", VK: 0x0D},
		Stop:  KeyBinding{Name: "F12", VK: 0x7B},
		Pause: KeyBinding{Name: "", VK: 0x41},
		Menu: MenuKeys{
			Inventory: KeyBinding{Name: "Wrong", VK: 0x43},
		},
		Skills: []Skill{
			{Name: "S1", Key: KeyBinding{Name: "Fake", VK: 0x31}, IntervalMS: DefaultIntervalMS, Enabled: false},
		},
	}
	cfg.Normalize()

	if cfg.Start.Name != "Enter" {
		t.Fatalf("start name = %q, want %q", cfg.Start.Name, "Enter")
	}
	if cfg.Stop.Name != "F12" {
		t.Fatalf("stop name = %q, want %q", cfg.Stop.Name, "F12")
	}
	if cfg.Pause.Name != "A" {
		t.Fatalf("pause name = %q, want %q", cfg.Pause.Name, "A")
	}
	if cfg.Menu.Inventory.Name != "C" {
		t.Fatalf("inventory name = %q, want %q", cfg.Menu.Inventory.Name, "C")
	}
	if cfg.Skills[0].Key.Name != "1" {
		t.Fatalf("skill key name = %q, want %q", cfg.Skills[0].Key.Name, "1")
	}
}
