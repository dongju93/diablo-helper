package config

import (
	"fmt"
	"math"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxSkills                = 8
	MaxKeyNameLength         = 64
	MaxSkillNameLength       = 64
	DefaultIntervalMS        = 1000
	DefaultSkillGapMS        = 0
	DefaultClickerIntervalMS = 100
	MinimumIntervalMS        = 10
	MaximumIntervalMS        = 60 * 60 * 1000
	MaximumSkillGapMS        = 60 * 60 * 1000
	MouseLeftVK              = 0x01
	DefaultSkillEnabled      = false
)

var maximumDurationMilliseconds = int64(math.MaxInt64 / int64(time.Millisecond))

type KeyBinding struct {
	Name string
	VK   int
}

func (k KeyBinding) Assigned() bool {
	return k.VK > 0
}

type Skill struct {
	Name       string
	Key        KeyBinding
	IntervalMS int
	Enabled    bool
}

type MenuKeys struct {
	Inventory  KeyBinding
	Skills     KeyBinding
	Follower   KeyBinding
	Map        KeyBinding
	WorldMap   KeyBinding
	TownPortal KeyBinding
	Chat       KeyBinding
	Whisper    KeyBinding
}

type Clicker struct {
	Start      KeyBinding
	Stop       KeyBinding
	Key        KeyBinding
	IntervalMS int
}

type MenuBinding struct {
	ID      string
	Label   string
	Binding KeyBinding
}

type Config struct {
	Start      KeyBinding
	Stop       KeyBinding
	Pause      KeyBinding
	Menu       MenuKeys
	Skills     []Skill
	SkillGapMS int
	Clicker    Clicker
}

func Default() Config {
	cfg := Config{
		Pause: KeyBinding{Name: "Mouse Right", VK: 0x02},
		Clicker: Clicker{
			Key:        KeyBinding{Name: "Mouse Left", VK: MouseLeftVK},
			IntervalMS: DefaultClickerIntervalMS,
		},
		Menu: MenuKeys{
			Inventory:  KeyBinding{Name: "C", VK: 0x43},
			Skills:     KeyBinding{Name: "V", VK: 0x56},
			Follower:   KeyBinding{Name: "F", VK: 0x46},
			Map:        KeyBinding{Name: "Tab", VK: 0x09},
			WorldMap:   KeyBinding{Name: "M", VK: 0x4D},
			TownPortal: KeyBinding{Name: "T", VK: 0x54},
			Chat:       KeyBinding{Name: "Enter", VK: 0x0D},
			Whisper:    KeyBinding{Name: "R", VK: 0x52},
		},
	}
	cfg.Normalize()
	return cfg
}

func (m *MenuKeys) SetKeyByID(id string, binding KeyBinding) bool {
	switch id {
	case "inventory":
		m.Inventory = binding
	case "skills":
		m.Skills = binding
	case "follower":
		m.Follower = binding
	case "map":
		m.Map = binding
	case "world_map":
		m.WorldMap = binding
	case "town_portal":
		m.TownPortal = binding
	case "chat":
		m.Chat = binding
	case "whisper":
		m.Whisper = binding
	default:
		return false
	}
	return true
}

func (m *MenuKeys) forEachKey(fn func(*KeyBinding)) {
	fn(&m.Inventory)
	fn(&m.Skills)
	fn(&m.Follower)
	fn(&m.Map)
	fn(&m.WorldMap)
	fn(&m.TownPortal)
	fn(&m.Chat)
	fn(&m.Whisper)
}

func (c *Config) Normalize() {
	if len(c.Skills) > MaxSkills {
		c.Skills = c.Skills[:MaxSkills]
	}
	for len(c.Skills) < MaxSkills {
		index := len(c.Skills) + 1
		c.Skills = append(c.Skills, Skill{
			Name:       fmt.Sprintf("Skill %d", index),
			IntervalMS: DefaultIntervalMS,
			Enabled:    DefaultSkillEnabled,
		})
	}
	for i := range c.Skills {
		if c.Skills[i].Name == "" {
			c.Skills[i].Name = fmt.Sprintf("Skill %d", i+1)
		}
		if c.Skills[i].IntervalMS < MinimumIntervalMS {
			c.Skills[i].IntervalMS = DefaultIntervalMS
		}
		normalizeKey(&c.Skills[i].Key)
	}
	if c.SkillGapMS < 0 {
		c.SkillGapMS = DefaultSkillGapMS
	}
	if c.Clicker.IntervalMS < MinimumIntervalMS {
		c.Clicker.IntervalMS = DefaultClickerIntervalMS
	}
	normalizeKey(&c.Start)
	normalizeKey(&c.Stop)
	normalizeKey(&c.Pause)
	normalizeKey(&c.Clicker.Start)
	normalizeKey(&c.Clicker.Stop)
	normalizeKey(&c.Clicker.Key)
	c.Menu.forEachKey(normalizeKey)
}

func (c Config) MenuBindings() []MenuBinding {
	return []MenuBinding{
		{ID: "inventory", Label: "Inventory", Binding: c.Menu.Inventory},
		{ID: "skills", Label: "Skills", Binding: c.Menu.Skills},
		{ID: "follower", Label: "Follower", Binding: c.Menu.Follower},
		{ID: "map", Label: "Map", Binding: c.Menu.Map},
		{ID: "world_map", Label: "World Map", Binding: c.Menu.WorldMap},
		{ID: "town_portal", Label: "Town Portal", Binding: c.Menu.TownPortal},
		{ID: "chat", Label: "Chat", Binding: c.Menu.Chat},
		{ID: "whisper", Label: "Whisper", Binding: c.Menu.Whisper},
	}
}

func (c Config) Validate() error {
	if len(c.Skills) > MaxSkills {
		return fmt.Errorf("skills must not exceed %d entries", MaxSkills)
	}
	if c.Start.VK == MouseLeftVK {
		return fmt.Errorf("start key must not be Mouse Left")
	}
	if c.Stop.VK == MouseLeftVK {
		return fmt.Errorf("stop key must not be Mouse Left")
	}
	if c.Clicker.Start.VK == MouseLeftVK {
		return fmt.Errorf("clicker start key must not be Mouse Left")
	}
	if c.Clicker.Stop.VK == MouseLeftVK {
		return fmt.Errorf("clicker stop key must not be Mouse Left")
	}
	if c.SkillGapMS < 0 {
		return fmt.Errorf("skill gap must be at least 0ms")
	}
	if c.SkillGapMS > MaximumSkillGapMS {
		return fmt.Errorf("skill gap must be at most %dms", MaximumSkillGapMS)
	}
	if !MillisecondsFitDuration(c.SkillGapMS) {
		return fmt.Errorf("skill gap is too large for time.Duration")
	}
	if c.Clicker.IntervalMS < MinimumIntervalMS {
		return fmt.Errorf("clicker interval must be at least %dms", MinimumIntervalMS)
	}
	if c.Clicker.IntervalMS > MaximumIntervalMS {
		return fmt.Errorf("clicker interval must be at most %dms", MaximumIntervalMS)
	}
	if !MillisecondsFitDuration(c.Clicker.IntervalMS) {
		return fmt.Errorf("clicker interval is too large for time.Duration")
	}
	for i, skill := range c.Skills {
		if skill.IntervalMS < MinimumIntervalMS {
			return fmt.Errorf("skill %d interval must be at least %dms", i+1, MinimumIntervalMS)
		}
		if skill.IntervalMS > MaximumIntervalMS {
			return fmt.Errorf("skill %d interval must be at most %dms", i+1, MaximumIntervalMS)
		}
		if !MillisecondsFitDuration(skill.IntervalMS) {
			return fmt.Errorf("skill %d interval is too large for time.Duration", i+1)
		}
		if err := validateConfigString(fmt.Sprintf("skill %d name", i+1), skill.Name, MaxSkillNameLength); err != nil {
			return err
		}
		if err := validateKey(fmt.Sprintf("skill %d key", i+1), skill.Key); err != nil {
			return err
		}
	}
	for _, item := range []struct {
		name    string
		binding KeyBinding
	}{
		{name: "start key", binding: c.Start},
		{name: "stop key", binding: c.Stop},
		{name: "pause key", binding: c.Pause},
		{name: "clicker start key", binding: c.Clicker.Start},
		{name: "clicker stop key", binding: c.Clicker.Stop},
		{name: "clicker key", binding: c.Clicker.Key},
		{name: "menu inventory key", binding: c.Menu.Inventory},
		{name: "menu skills key", binding: c.Menu.Skills},
		{name: "menu follower key", binding: c.Menu.Follower},
		{name: "menu map key", binding: c.Menu.Map},
		{name: "menu world map key", binding: c.Menu.WorldMap},
		{name: "menu town portal key", binding: c.Menu.TownPortal},
		{name: "menu chat key", binding: c.Menu.Chat},
		{name: "menu whisper key", binding: c.Menu.Whisper},
	} {
		if err := validateKey(item.name, item.binding); err != nil {
			return err
		}
	}
	return nil
}

func MillisecondsFitDuration(ms int) bool {
	return ms >= 0 && int64(ms) <= maximumDurationMilliseconds
}

func normalizeKey(binding *KeyBinding) {
	if binding.VK < 0 || binding.VK > 255 {
		binding.VK = 0
	}
	if binding.VK == 0 {
		binding.Name = ""
	}
}

func validateKey(name string, binding KeyBinding) error {
	if err := validateConfigString(name+" name", binding.Name, MaxKeyNameLength); err != nil {
		return err
	}
	if binding.VK < 0 || binding.VK > 255 {
		return fmt.Errorf("%s vk must be between 0 and 255", name)
	}
	if binding.VK == 0 && binding.Name != "" {
		return fmt.Errorf("%s has a name but no virtual-key code", name)
	}
	return nil
}

func validateConfigString(name string, value string, maxLength int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", name)
	}
	if utf8.RuneCountInString(value) > maxLength {
		return fmt.Errorf("%s must not exceed %d characters", name, maxLength)
	}
	for _, r := range value {
		if r == 0 {
			return fmt.Errorf("%s must not contain NUL", name)
		}
		if unicode.IsControl(r) {
			return fmt.Errorf("%s must not contain control characters", name)
		}
	}
	return nil
}
