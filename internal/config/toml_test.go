package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarshalParseRoundTrip(t *testing.T) {
	cfg := Default()
	cfg.Start = KeyBinding{Name: "F5", VK: 0x74}
	cfg.Stop = KeyBinding{Name: "F6", VK: 0x75}
	cfg.Pause = KeyBinding{Name: "Space", VK: 0x20}
	cfg.SkillGapMS = 45
	cfg.Clicker.Start = KeyBinding{Name: "F7", VK: 0x76}
	cfg.Clicker.Stop = KeyBinding{Name: "F8", VK: 0x77}
	cfg.Clicker.Key = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
	cfg.Clicker.IntervalMS = 75
	cfg.Menu.Map = KeyBinding{Name: "Tab", VK: 0x09}
	cfg.Menu.Collection = KeyBinding{Name: "Y", VK: 0x59}
	cfg.Skills[0] = Skill{
		Name:       "Primary",
		Key:        KeyBinding{Name: "1", VK: 0x31},
		IntervalMS: 125,
		Enabled:    true,
	}
	cfg.Skills[1] = Skill{
		Name:       "Disabled",
		Key:        KeyBinding{Name: "2", VK: 0x32},
		IntervalMS: 2500,
		Enabled:    false,
	}

	data, err := MarshalTOML(cfg)
	if err != nil {
		t.Fatalf("MarshalTOML() error = %v", err)
	}

	parsed, err := ParseTOML(data)
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}

	if parsed.Start != cfg.Start {
		t.Fatalf("start binding = %+v, want %+v", parsed.Start, cfg.Start)
	}
	if parsed.Menu.Map != cfg.Menu.Map {
		t.Fatalf("map binding = %+v, want %+v", parsed.Menu.Map, cfg.Menu.Map)
	}
	if parsed.Menu.Collection != cfg.Menu.Collection {
		t.Fatalf("collection binding = %+v, want %+v", parsed.Menu.Collection, cfg.Menu.Collection)
	}
	if parsed.SkillGapMS != cfg.SkillGapMS {
		t.Fatalf("skill gap = %d, want %d", parsed.SkillGapMS, cfg.SkillGapMS)
	}
	if parsed.Clicker != cfg.Clicker {
		t.Fatalf("clicker = %+v, want %+v", parsed.Clicker, cfg.Clicker)
	}
	if len(parsed.Skills) != MaxSkills {
		t.Fatalf("skills length = %d, want %d", len(parsed.Skills), MaxSkills)
	}
	if parsed.Skills[0] != cfg.Skills[0] {
		t.Fatalf("skill 0 = %+v, want %+v", parsed.Skills[0], cfg.Skills[0])
	}
	if parsed.Skills[1] != cfg.Skills[1] {
		t.Fatalf("skill 1 = %+v, want %+v", parsed.Skills[1], cfg.Skills[1])
	}
}

func TestValidateRejectsLeftMouseStartAndStop(t *testing.T) {
	cfg := Default()
	cfg.Start = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want start Mouse Left error")
	}

	cfg = Default()
	cfg.Stop = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want stop Mouse Left error")
	}

	cfg = Default()
	cfg.Clicker.Start = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want clicker start Mouse Left error")
	}

	cfg = Default()
	cfg.Clicker.Stop = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want clicker stop Mouse Left error")
	}

	cfg = Default()
	cfg.Clicker.Key = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want clicker key Mouse Left to be allowed", err)
	}
}

func TestDefaultSkillsStartDisabled(t *testing.T) {
	cfg := Default()
	for i, skill := range cfg.Skills {
		if skill.Enabled {
			t.Fatalf("skill %d enabled = true, want false", i+1)
		}
	}
}

func TestParseTOMLRejectsUnknownKeys(t *testing.T) {
	_, err := ParseTOML([]byte(`start_key_name = "F5"
bad_key = 123
`))
	if err == nil {
		t.Fatal("ParseTOML() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("ParseTOML() error = %v, want unknown key error", err)
	}
}

func TestParseTOMLRejectsIntervalsAboveMaximum(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "skill gap",
			input:     fmt.Sprintf("skill_gap_ms = %d\n", MaximumSkillGapMS+1),
			wantError: "skill gap must be at most",
		},
		{
			name:      "clicker interval",
			input:     fmt.Sprintf("clicker_interval_ms = %d\n", MaximumIntervalMS+1),
			wantError: "clicker interval must be at most",
		},
		{
			name: "skill interval",
			input: fmt.Sprintf(`[[skills]]
name = "Too Large"
key_name = "1"
key_vk = 49
interval_ms = %d
enabled = true
`, MaximumIntervalMS+1),
			wantError: "interval must be at most",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTOML([]byte(tt.input))
			if err == nil {
				t.Fatal("ParseTOML() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ParseTOML() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestParseTOMLNormalizesSkillCountAndIntervals(t *testing.T) {
	cfg, err := ParseTOML([]byte(`[[skills]]
name = "Only"
key_name = "1"
key_vk = 49
interval_ms = 0
enabled = true
`))
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}
	if len(cfg.Skills) != MaxSkills {
		t.Fatalf("skills length = %d, want %d", len(cfg.Skills), MaxSkills)
	}
	if cfg.Skills[0].IntervalMS != DefaultIntervalMS {
		t.Fatalf("interval = %d, want %d", cfg.Skills[0].IntervalMS, DefaultIntervalMS)
	}
}

func TestParseTOMLDefaultsMissingSkillEnabledToFalse(t *testing.T) {
	cfg, err := ParseTOML([]byte(`[[skills]]
name = "Only"
key_name = "1"
key_vk = 49
interval_ms = 100
`))
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}
	if cfg.Skills[0].Enabled {
		t.Fatal("skill enabled = true, want false")
	}
}

func TestSaveFileAndLoadFileRoundTrip(t *testing.T) {
	cfg := Default()
	cfg.Start = KeyBinding{Name: "F5", VK: 0x74}
	cfg.Stop = KeyBinding{Name: "Mouse X1", VK: 0x05}
	cfg.SkillGapMS = 25
	cfg.Skills[0] = Skill{
		Name:       "Primary",
		Key:        KeyBinding{Name: "1", VK: 0x31},
		IntervalMS: 33,
		Enabled:    true,
	}

	path := filepath.Join(t.TempDir(), "default.toml")
	if err := SaveFile(path, cfg); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if loaded.Start != cfg.Start {
		t.Fatalf("start = %+v, want %+v", loaded.Start, cfg.Start)
	}
	if loaded.Stop != cfg.Stop {
		t.Fatalf("stop = %+v, want %+v", loaded.Stop, cfg.Stop)
	}
	if loaded.SkillGapMS != cfg.SkillGapMS {
		t.Fatalf("skill gap = %d, want %d", loaded.SkillGapMS, cfg.SkillGapMS)
	}
	if loaded.Clicker != cfg.Clicker {
		t.Fatalf("clicker = %+v, want %+v", loaded.Clicker, cfg.Clicker)
	}
	if loaded.Skills[0] != cfg.Skills[0] {
		t.Fatalf("skill 0 = %+v, want %+v", loaded.Skills[0], cfg.Skills[0])
	}
	if len(loaded.Skills) != MaxSkills {
		t.Fatalf("skills length = %d, want %d", len(loaded.Skills), MaxSkills)
	}
}

func TestMarshalTOMLRejectsInvalidStartStopBindings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "start mouse left",
			mutate: func(cfg *Config) {
				cfg.Start = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
			},
		},
		{
			name: "stop mouse left",
			mutate: func(cfg *Config) {
				cfg.Stop = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
			},
		},
		{
			name: "clicker start mouse left",
			mutate: func(cfg *Config) {
				cfg.Clicker.Start = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
			},
		},
		{
			name: "clicker stop mouse left",
			mutate: func(cfg *Config) {
				cfg.Clicker.Stop = KeyBinding{Name: "Mouse Left", VK: MouseLeftVK}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg)
			if _, err := MarshalTOML(cfg); err == nil {
				t.Fatal("MarshalTOML() error = nil, want error")
			}
		})
	}
}

func TestMarshalTOMLNormalizesOutput(t *testing.T) {
	cfg := Config{
		Skills: []Skill{
			{Name: "", Key: KeyBinding{Name: "Bad", VK: 500}, IntervalMS: 0},
		},
	}

	data, err := MarshalTOML(cfg)
	if err != nil {
		t.Fatalf("MarshalTOML() error = %v", err)
	}
	text := string(data)
	for _, want := range []string{
		`name = "Skill 1"`,
		`key_name = ""`,
		`key_vk = 0`,
		`skill_gap_ms = 0`,
		`clicker_interval_ms = 100`,
		`interval_ms = 1000`,
		`enabled = false`,
		`name = "Skill 8"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("MarshalTOML() output missing %q:\n%s", want, text)
		}
	}
}

func TestParseTOMLHandlesCommentsAndQuotedHashes(t *testing.T) {
	cfg, err := ParseTOML([]byte(`
pause_key_name = "Hash # Key"
pause_key_vk = 35

[[skills]]
name = "Quote \" # inside"
key_name = "1#not-comment"
key_vk = 49 # real comment
interval_ms = 10
enabled = true
`))
	if err != nil {
		t.Fatalf("ParseTOML() error = %v", err)
	}
	if cfg.Pause != (KeyBinding{Name: "End", VK: 35}) {
		t.Fatalf("pause = %+v, want canonical name for VK 35", cfg.Pause)
	}
	if cfg.Skills[0].Name != `Quote " # inside` {
		t.Fatalf("skill name = %q, want escaped quote and hash", cfg.Skills[0].Name)
	}
	if cfg.Skills[0].Key != (KeyBinding{Name: "1", VK: 49}) {
		t.Fatalf("skill key = %+v, want canonical key name for VK 49", cfg.Skills[0].Key)
	}
	if !cfg.Skills[0].Enabled {
		t.Fatal("skill enabled = false, want true")
	}
}

func TestParseTOMLRejectsUnsafeStrings(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "key name nul",
			input:     `start_key_name = "\u0000"` + "\n",
			wantError: "must not contain NUL",
		},
		{
			name:      "key name control character",
			input:     `start_key_name = "Bad\nName"` + "\n",
			wantError: "must not contain control characters",
		},
		{
			name:      "key name too long",
			input:     fmt.Sprintf("start_key_name = %q\n", strings.Repeat("A", MaxKeyNameLength+1)),
			wantError: "must not exceed",
		},
		{
			name:      "skill name too long",
			input:     fmt.Sprintf("[[skills]]\nname = %q\n", strings.Repeat("A", MaxSkillNameLength+1)),
			wantError: "must not exceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTOML([]byte(tt.input))
			if err == nil {
				t.Fatal("ParseTOML() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ParseTOML() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestParseTOMLRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{name: "unsupported section", input: "[profile]\n", wantError: "unsupported section"},
		{name: "missing equals", input: `start_key_name "F5"` + "\n", wantError: "expected key = value"},
		{name: "empty key", input: ` = "F5"` + "\n", wantError: "empty key"},
		{name: "bad string", input: "start_key_name = F5\n", wantError: "expected quoted string"},
		{name: "bad integer", input: "start_key_vk = nope\n", wantError: "expected integer"},
		{name: "bad boolean", input: "[[skills]]\nenabled = maybe\n", wantError: "expected boolean"},
		{name: "unknown skill key", input: "[[skills]]\nunknown = 1\n", wantError: "unknown skill key"},
		{name: "validation error", input: "start_key_name = \"Mouse Left\"\nstart_key_vk = 1\n", wantError: "Mouse Left"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTOML([]byte(tt.input))
			if err == nil {
				t.Fatal("ParseTOML() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ParseTOML() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestParseTOMLRejectsExtraSkills(t *testing.T) {
	var input strings.Builder
	for i := 1; i <= MaxSkills+2; i++ {
		fmt.Fprintf(&input, `
[[skills]]
name = "Skill %d"
key_name = "%d"
key_vk = %d
interval_ms = %d
enabled = true
`, i, i, 48+i, MinimumIntervalMS)
	}

	_, err := ParseTOML([]byte(input.String()))
	if err == nil {
		t.Fatal("ParseTOML() error = nil, want error for too many [[skills]] sections")
	}
	if !strings.Contains(err.Error(), "too many [[skills]]") {
		t.Fatalf("ParseTOML() error = %v, want 'too many [[skills]]'", err)
	}
}
