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
	if cfg.Menu.Character != (KeyBinding{Name: "C", VK: 0x43}) {
		t.Fatalf("character = %+v, want C", cfg.Menu.Character)
	}
	if cfg.Menu.SkillAssign != (KeyBinding{Name: "S", VK: 0x53}) {
		t.Fatalf("skill assign = %+v, want S", cfg.Menu.SkillAssign)
	}
	if cfg.Menu.Shop != (KeyBinding{Name: "P", VK: 0x50}) {
		t.Fatalf("shop = %+v, want P", cfg.Menu.Shop)
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

func TestNormalizeForUIRepairsConfigShapeAndValues(t *testing.T) {
	cfg := Config{
		Start: KeyBinding{Name: "Bad Start", VK: 300},
		Stop:  KeyBinding{Name: "No Code", VK: 0},
		Pause: KeyBinding{Name: "Bad Pause", VK: -1},
		Menu: MenuKeys{
			Character: KeyBinding{Name: "Bad Menu", VK: 999},
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

	cfg.NormalizeForUI()

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
	if cfg.Menu.Character != (KeyBinding{}) {
		t.Fatalf("character = %+v, want cleared", cfg.Menu.Character)
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

func TestMenuBindingDefinitionsOrderAndLabels(t *testing.T) {
	got := MenuBindingDefinitions()
	want := []MenuBindingDefinition{
		{ID: "character", Label: "Character", UILabel: "캐릭터"},
		{ID: "skill_assign", Label: "Skill Assign", UILabel: "스킬 배치"},
		{ID: "talents", Label: "Talents", UILabel: "능력치"},
		{ID: "map", Label: "Map", UILabel: "지도"},
		{ID: "journal", Label: "Journal", UILabel: "일지"},
		{ID: "social", Label: "Social", UILabel: "소셜"},
		{ID: "clan", Label: "Clan", UILabel: "클랜"},
		{ID: "town_portal", Label: "Town Portal", UILabel: "차원문"},
		{ID: "collection", Label: "Collection", UILabel: "컬렉션"},
		{ID: "shop", Label: "Shop", UILabel: "상점"},
	}

	if len(got) != len(want) {
		t.Fatalf("MenuBindingDefinitions() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("MenuBindingDefinitions()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}

	got[0].ID = "mutated"
	if again := MenuBindingDefinitions(); again[0].ID == "mutated" {
		t.Fatal("MenuBindingDefinitions() exposed mutable backing storage")
	}
}

func TestMenuBindingsOrderLabelsAndValues(t *testing.T) {
	cfg := Default()
	got := cfg.MenuBindings()
	want := []MenuBinding{
		{ID: "character", Label: "Character", Binding: cfg.Menu.Character},
		{ID: "skill_assign", Label: "Skill Assign", Binding: cfg.Menu.SkillAssign},
		{ID: "talents", Label: "Talents", Binding: cfg.Menu.Talents},
		{ID: "map", Label: "Map", Binding: cfg.Menu.Map},
		{ID: "journal", Label: "Journal", Binding: cfg.Menu.Journal},
		{ID: "social", Label: "Social", Binding: cfg.Menu.Social},
		{ID: "clan", Label: "Clan", Binding: cfg.Menu.Clan},
		{ID: "town_portal", Label: "Town Portal", Binding: cfg.Menu.TownPortal},
		{ID: "collection", Label: "Collection", Binding: cfg.Menu.Collection},
		{ID: "shop", Label: "Shop", Binding: cfg.Menu.Shop},
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
		{
			name: "skill key is Left Win",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "Left Win", VK: 0x5B}
				cfg.Skills[0].Enabled = true
			},
			wantError: "must not be a system key",
		},
		{
			name: "skill key is Right Win",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "Right Win", VK: 0x5C}
				cfg.Skills[0].Enabled = true
			},
			wantError: "must not be a system key",
		},
		{
			name: "skill key is Esc",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "Esc", VK: 0x1B}
				cfg.Skills[0].Enabled = true
			},
			wantError: "must not be a system key",
		},
		{
			name: "skill key is Num Lock",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "Num Lock", VK: 0x90}
				cfg.Skills[0].Enabled = true
			},
			wantError: "must not be a system key",
		},
		{
			name: "skill key is Scroll Lock",
			mutate: func(cfg *Config) {
				cfg.Skills[0].Key = KeyBinding{Name: "Scroll Lock", VK: 0x91}
				cfg.Skills[0].Enabled = true
			},
			wantError: "must not be a system key",
		},
		{
			name: "clicker key is Left Win",
			mutate: func(cfg *Config) {
				cfg.Clicker.Key = KeyBinding{Name: "Left Win", VK: 0x5B}
			},
			wantError: "must not be a system key",
		},
		{
			name: "clicker key is Esc",
			mutate: func(cfg *Config) {
				cfg.Clicker.Key = KeyBinding{Name: "Esc", VK: 0x1B}
			},
			wantError: "must not be a system key",
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

func TestHotkeysPermitKeysBlockedForOutput(t *testing.T) {
	// Hotkeys (start/stop/pause) use validateKey, not validateOutputKey, so
	// output-blocked keys like Esc are allowed as user-controlled triggers.
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "start key can be Esc",
			mutate: func(cfg *Config) {
				cfg.Start = KeyBinding{Name: "Esc", VK: 0x1B}
			},
		},
		{
			name: "stop key can be Shift",
			mutate: func(cfg *Config) {
				cfg.Stop = KeyBinding{Name: "Shift", VK: 0x10}
			},
		},
		{
			name: "pause key can be Alt",
			mutate: func(cfg *Config) {
				cfg.Pause = KeyBinding{Name: "Alt", VK: 0x12}
			},
		},
		{
			name: "stop key can be Left Ctrl",
			mutate: func(cfg *Config) {
				cfg.Stop = KeyBinding{Name: "Left Ctrl", VK: 0xA2}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestOutputKeysPermitModifiers(t *testing.T) {
	tests := []KeyBinding{
		{Name: "Shift", VK: 0x10},
		{Name: "Ctrl", VK: 0x11},
		{Name: "Alt", VK: 0x12},
		{Name: "Left Shift", VK: 0xA0},
		{Name: "Right Shift", VK: 0xA1},
		{Name: "Left Ctrl", VK: 0xA2},
		{Name: "Right Ctrl", VK: 0xA3},
		{Name: "Left Alt", VK: 0xA4},
		{Name: "Right Alt", VK: 0xA5},
	}

	for _, binding := range tests {
		t.Run("skill "+binding.Name, func(t *testing.T) {
			cfg := Default()
			cfg.Skills[0].Key = binding
			cfg.Skills[0].Enabled = true

			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})

		t.Run("clicker "+binding.Name, func(t *testing.T) {
			cfg := Default()
			cfg.Clicker.Key = binding

			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
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
		{name: "left shift", vk: 0xA0, want: "Left Shift"},
		{name: "right shift", vk: 0xA1, want: "Right Shift"},
		{name: "left ctrl", vk: 0xA2, want: "Left Ctrl"},
		{name: "right ctrl", vk: 0xA3, want: "Right Ctrl"},
		{name: "left alt", vk: 0xA4, want: "Left Alt"},
		{name: "right alt", vk: 0xA5, want: "Right Alt"},
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

func TestNormalizeForUICanonicalizesKeyNames(t *testing.T) {
	cfg := Config{
		Start: KeyBinding{Name: "Spoofed", VK: 0x0D},
		Stop:  KeyBinding{Name: "F12", VK: 0x7B},
		Pause: KeyBinding{Name: "", VK: 0x41},
		Menu: MenuKeys{
			Character: KeyBinding{Name: "Wrong", VK: 0x43},
		},
		Skills: []Skill{
			{Name: "S1", Key: KeyBinding{Name: "Fake", VK: 0x31}, IntervalMS: DefaultIntervalMS, Enabled: false},
		},
	}
	cfg.NormalizeForUI()

	if cfg.Start.Name != "Enter" {
		t.Fatalf("start name = %q, want %q", cfg.Start.Name, "Enter")
	}
	if cfg.Stop.Name != "F12" {
		t.Fatalf("stop name = %q, want %q", cfg.Stop.Name, "F12")
	}
	if cfg.Pause.Name != "A" {
		t.Fatalf("pause name = %q, want %q", cfg.Pause.Name, "A")
	}
	if cfg.Menu.Character.Name != "C" {
		t.Fatalf("character name = %q, want %q", cfg.Menu.Character.Name, "C")
	}
	if cfg.Skills[0].Key.Name != "1" {
		t.Fatalf("skill key name = %q, want %q", cfg.Skills[0].Key.Name, "1")
	}
}

func TestMenuKeysMatches(t *testing.T) {
	m := Default().Menu
	m.SetKeyByID("clan", KeyBinding{Name: "F7", VK: 0x76})

	if !m.Matches(0x76) {
		t.Fatal("Matches(F7) = false, want true (assigned clan binding)")
	}
	if m.Matches(0x77) {
		t.Fatal("Matches(F8) = true, want false (not assigned)")
	}
	if m.Matches(0) {
		t.Fatal("Matches(0) = true, want false (VK 0 is unassigned)")
	}
}

func TestMenuKeysBindingByID(t *testing.T) {
	m := Default().Menu
	b, ok := m.BindingByID("character")
	if !ok || b != m.Character {
		t.Fatalf("BindingByID(character) = %+v, %v; want %+v, true", b, ok, m.Character)
	}
	_, ok = m.BindingByID("nonexistent")
	if ok {
		t.Fatal("BindingByID(nonexistent) returned ok=true")
	}
}

func BenchmarkMenuKeysMatches(b *testing.B) {
	m := Default().Menu
	m.SetKeyByID("shop", KeyBinding{Name: "F7", VK: 0x76})
	vk := uint16(0x76)
	b.ReportAllocs()
	for b.Loop() {
		m.Matches(vk)
	}
}
