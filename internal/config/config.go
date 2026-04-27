package config

import (
	"fmt"
	"math"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxConfigFileBytes       = 64 * 1024
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

// forbiddenOutputVKs is the set of virtual-key codes that must not be used as
// automated output (skill keys, clicker key). Synthesising these system-wide
// triggers OS-level or application-global side effects that the user cannot
// easily interrupt.
var forbiddenOutputVKs = map[int]string{
	0x10: "Shift",
	0x11: "Ctrl",
	0x12: "Alt",
	0x13: "Pause",
	0x14: "Caps Lock",
	0x1B: "Esc",
	0x5B: "Left Win",
	0x5C: "Right Win",
}

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
	Character   KeyBinding
	SkillAssign KeyBinding
	Talents     KeyBinding
	Map         KeyBinding
	Journal     KeyBinding
	Social      KeyBinding
	Clan        KeyBinding
	TownPortal  KeyBinding
	Collection  KeyBinding
	Shop        KeyBinding
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
			Character:   KeyBinding{Name: "C", VK: 0x43},
			SkillAssign: KeyBinding{Name: "S", VK: 0x53},
			Talents:     KeyBinding{Name: "A", VK: 0x41},
			Map:         KeyBinding{Name: "Tab", VK: 0x09},
			Journal:     KeyBinding{Name: "J", VK: 0x4A},
			Social:      KeyBinding{Name: "O", VK: 0x4F},
			Clan:        KeyBinding{Name: "N", VK: 0x4E},
			TownPortal:  KeyBinding{Name: "T", VK: 0x54},
			Collection:  KeyBinding{Name: "Y", VK: 0x59},
			Shop:        KeyBinding{Name: "P", VK: 0x50},
		},
	}
	cfg.Normalize()
	return cfg
}

func (m *MenuKeys) SetKeyByID(id string, binding KeyBinding) bool {
	switch id {
	case "character":
		m.Character = binding
	case "skill_assign":
		m.SkillAssign = binding
	case "talents":
		m.Talents = binding
	case "map":
		m.Map = binding
	case "journal":
		m.Journal = binding
	case "social":
		m.Social = binding
	case "clan":
		m.Clan = binding
	case "town_portal":
		m.TownPortal = binding
	case "collection":
		m.Collection = binding
	case "shop":
		m.Shop = binding
	default:
		return false
	}
	return true
}

func (m *MenuKeys) forEachKey(fn func(*KeyBinding)) {
	fn(&m.Character)
	fn(&m.SkillAssign)
	fn(&m.Talents)
	fn(&m.Map)
	fn(&m.Journal)
	fn(&m.Social)
	fn(&m.Clan)
	fn(&m.TownPortal)
	fn(&m.Collection)
	fn(&m.Shop)
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
		{ID: "character", Label: "Character", Binding: c.Menu.Character},
		{ID: "skill_assign", Label: "Skill Assign", Binding: c.Menu.SkillAssign},
		{ID: "talents", Label: "Talents", Binding: c.Menu.Talents},
		{ID: "map", Label: "Map", Binding: c.Menu.Map},
		{ID: "journal", Label: "Journal", Binding: c.Menu.Journal},
		{ID: "social", Label: "Social", Binding: c.Menu.Social},
		{ID: "clan", Label: "Clan", Binding: c.Menu.Clan},
		{ID: "town_portal", Label: "Town Portal", Binding: c.Menu.TownPortal},
		{ID: "collection", Label: "Collection", Binding: c.Menu.Collection},
		{ID: "shop", Label: "Shop", Binding: c.Menu.Shop},
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
		if err := validateOutputKey(fmt.Sprintf("skill %d key", i+1), skill.Key); err != nil {
			return err
		}
	}
	if err := validateOutputKey("clicker key", c.Clicker.Key); err != nil {
		return err
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
		{name: "menu character key", binding: c.Menu.Character},
		{name: "menu skill assign key", binding: c.Menu.SkillAssign},
		{name: "menu talents key", binding: c.Menu.Talents},
		{name: "menu map key", binding: c.Menu.Map},
		{name: "menu journal key", binding: c.Menu.Journal},
		{name: "menu social key", binding: c.Menu.Social},
		{name: "menu clan key", binding: c.Menu.Clan},
		{name: "menu town portal key", binding: c.Menu.TownPortal},
		{name: "menu collection key", binding: c.Menu.Collection},
		{name: "menu shop key", binding: c.Menu.Shop},
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

func KeyDisplayName(vk int) string {
	if vk < 0 || vk > 255 {
		return fmt.Sprintf("VK_%d", vk)
	}
	if vk >= '0' && vk <= '9' {
		return string(rune(vk))
	}
	if vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= 0x70 && vk <= 0x87 {
		return fmt.Sprintf("F%d", vk-0x70+1)
	}
	if vk >= 0x60 && vk <= 0x69 {
		return fmt.Sprintf("Numpad %d", vk-0x60)
	}

	switch vk {
	case 0x01:
		return "Mouse Left"
	case 0x02:
		return "Mouse Right"
	case 0x04:
		return "Mouse Middle"
	case 0x05:
		return "Mouse X1"
	case 0x06:
		return "Mouse X2"
	case 0x08:
		return "Backspace"
	case 0x09:
		return "Tab"
	case 0x0D:
		return "Enter"
	case 0x10:
		return "Shift"
	case 0x11:
		return "Ctrl"
	case 0x12:
		return "Alt"
	case 0x13:
		return "Pause"
	case 0x14:
		return "Caps Lock"
	case 0x1B:
		return "Esc"
	case 0x20:
		return "Space"
	case 0x21:
		return "Page Up"
	case 0x22:
		return "Page Down"
	case 0x23:
		return "End"
	case 0x24:
		return "Home"
	case 0x25:
		return "Left"
	case 0x26:
		return "Up"
	case 0x27:
		return "Right"
	case 0x28:
		return "Down"
	case 0x2D:
		return "Insert"
	case 0x2E:
		return "Delete"
	case 0x5B:
		return "Left Win"
	case 0x5C:
		return "Right Win"
	default:
		return fmt.Sprintf("VK_%d", vk)
	}
}

func normalizeKey(binding *KeyBinding) {
	if binding.VK < 0 || binding.VK > 255 {
		binding.VK = 0
	}
	if binding.VK == 0 {
		binding.Name = ""
	} else {
		binding.Name = KeyDisplayName(binding.VK)
	}
}

func validateOutputKey(name string, binding KeyBinding) error {
	if err := validateKey(name, binding); err != nil {
		return err
	}
	if label, forbidden := forbiddenOutputVKs[binding.VK]; forbidden {
		return fmt.Errorf("%s must not be a system key (%s)", name, label)
	}
	return nil
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
	if binding.VK > 0 && binding.Name != KeyDisplayName(binding.VK) {
		return fmt.Errorf("%s name does not match virtual-key code", name)
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
