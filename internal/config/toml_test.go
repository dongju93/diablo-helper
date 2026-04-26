package config

import (
	"strings"
	"testing"
)

func TestMarshalParseRoundTrip(t *testing.T) {
	cfg := Default()
	cfg.Start = KeyBinding{Name: "F5", VK: 0x74}
	cfg.Stop = KeyBinding{Name: "F6", VK: 0x75}
	cfg.Pause = KeyBinding{Name: "Space", VK: 0x20}
	cfg.Menu.WorldMap = KeyBinding{Name: "M", VK: 0x4D}
	cfg.Menu.Whisper = KeyBinding{Name: "R", VK: 0x52}
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
	if parsed.Menu.WorldMap != cfg.Menu.WorldMap {
		t.Fatalf("world map binding = %+v, want %+v", parsed.Menu.WorldMap, cfg.Menu.WorldMap)
	}
	if parsed.Menu.Whisper != cfg.Menu.Whisper {
		t.Fatalf("whisper binding = %+v, want %+v", parsed.Menu.Whisper, cfg.Menu.Whisper)
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
