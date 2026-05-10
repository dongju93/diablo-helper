// Package config loads, validates, normalizes, and saves Diablo Helper settings.
package config

import (
	"fmt"
	"math"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// MaxConfigFileBytes is the largest TOML config file accepted by LoadFile.
	MaxConfigFileBytes = 64 * 1024
	// MaxSkills is the fixed number of skill slots shown by the UI and saved to TOML.
	MaxSkills = 8
	// MaxKeyNameLength is the maximum number of runes accepted for a stored key_name.
	MaxKeyNameLength = 64
	// MaxSkillNameLength is the maximum number of runes accepted for a skill name.
	MaxSkillNameLength = 64
	// DefaultIntervalMS is the default skill repeat interval in milliseconds.
	DefaultIntervalMS = 1000
	// DefaultSkillGapMS is the default delay inserted between skill sends in milliseconds.
	DefaultSkillGapMS = 0
	// DefaultClickerIntervalMS is the default clicker repeat interval in milliseconds.
	DefaultClickerIntervalMS = 100
	// MinimumIntervalMS is the minimum accepted repeat interval in milliseconds.
	MinimumIntervalMS = 10
	// MaximumIntervalMS is the maximum accepted repeat interval in milliseconds.
	MaximumIntervalMS = 60 * 60 * 1000
	// MaximumSkillGapMS is the maximum accepted delay between skill sends in milliseconds.
	MaximumSkillGapMS = 60 * 60 * 1000
	// MouseLeftVK is the Win32 virtual-key code for the left mouse button.
	MouseLeftVK = 0x01
	// DefaultSkillEnabled is the enabled state assigned to newly created skill slots.
	DefaultSkillEnabled = false
)

var maximumDurationMilliseconds = int64(math.MaxInt64 / int64(time.Millisecond))

// forbiddenOutputKey reports whether vk must not be used as automated output.
// These keys trigger OS/window actions or toggle keyboard state rather than
// acting as ordinary game inputs.
func forbiddenOutputKey(vk int) (string, bool) {
	switch vk {
	case 0x13:
		return "Pause", true
	case 0x14:
		return "Caps Lock", true
	case 0x1B:
		return "Esc", true
	case 0x5B:
		return "Left Win", true
	case 0x5C:
		return "Right Win", true
	case 0x90:
		return "Num Lock", true
	case 0x91:
		return "Scroll Lock", true
	default:
		return "", false
	}
}

// KeyBinding stores a display name together with its Win32 virtual-key code.
type KeyBinding struct {
	// Name is the stored key_name and is rewritten from VK by NormalizeForUI.
	Name string
	// VK is the Win32 virtual-key code, with 0 meaning unassigned.
	VK int
}

// Assigned reports whether the binding has a nonzero virtual-key code.
func (k KeyBinding) Assigned() bool {
	return k.VK > 0
}

// Skill describes one automated skill slot and its repeat settings.
type Skill struct {
	// Name is the user-facing skill slot label.
	Name string
	// Key is the automated output key sent for this skill.
	Key KeyBinding
	// IntervalMS is the repeat interval for this skill in milliseconds.
	IntervalMS int
	// Enabled reports whether the runner should include this skill slot.
	Enabled bool
}

// MenuKeys groups the hotkeys used to open Diablo IV menu screens.
type MenuKeys struct {
	// Character is the hotkey for the character menu.
	Character KeyBinding
	// SkillAssign is the hotkey for the skill assignment menu.
	SkillAssign KeyBinding
	// Talents is the hotkey for the talents menu.
	Talents KeyBinding
	// Map is the hotkey for the map menu.
	Map KeyBinding
	// Journal is the hotkey for the journal menu.
	Journal KeyBinding
	// Social is the hotkey for the social menu.
	Social KeyBinding
	// Clan is the hotkey for the clan menu.
	Clan KeyBinding
	// TownPortal is the hotkey for the town portal action.
	TownPortal KeyBinding
	// Collection is the hotkey for the collection menu.
	Collection KeyBinding
	// Shop is the hotkey for the shop menu.
	Shop KeyBinding
}

// Clicker describes the mouse click automation bindings and interval.
type Clicker struct {
	// Start is the hotkey that starts click automation.
	Start KeyBinding
	// Stop is the hotkey that stops click automation.
	Stop KeyBinding
	// Key is the automated output key sent by the clicker.
	Key KeyBinding
	// IntervalMS is the repeat interval for click automation in milliseconds.
	IntervalMS int
}

// MenuBinding is a resolved menu binding presented by ID and label.
type MenuBinding struct {
	// ID is the stable menu action identifier.
	ID string
	// Label is the English menu action label.
	Label string
	// Binding is the configured key for the menu action.
	Binding KeyBinding
}

// MenuBindingDefinition describes a supported game menu action.
type MenuBindingDefinition struct {
	// ID is the stable menu action identifier used by config and UI code.
	ID string
	// Label is the English menu action label.
	Label string
	// UILabel is the localized label shown in the Windows UI.
	UILabel string
}

type menuBindingSpec struct {
	definition     MenuBindingDefinition
	validationName string
	defaultVK      int
	binding        func(*MenuKeys) *KeyBinding
	value          func(MenuKeys) KeyBinding
}

var menuBindingSpecs = [...]menuBindingSpec{
	{
		definition:     MenuBindingDefinition{ID: "character", Label: "Character", UILabel: "캐릭터"},
		validationName: "menu character key",
		defaultVK:      0x43,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Character },
		value:          func(m MenuKeys) KeyBinding { return m.Character },
	},
	{
		definition:     MenuBindingDefinition{ID: "skill_assign", Label: "Skill Assign", UILabel: "스킬 배치"},
		validationName: "menu skill assign key",
		defaultVK:      0x53,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.SkillAssign },
		value:          func(m MenuKeys) KeyBinding { return m.SkillAssign },
	},
	{
		definition:     MenuBindingDefinition{ID: "talents", Label: "Talents", UILabel: "능력치"},
		validationName: "menu talents key",
		defaultVK:      0x41,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Talents },
		value:          func(m MenuKeys) KeyBinding { return m.Talents },
	},
	{
		definition:     MenuBindingDefinition{ID: "map", Label: "Map", UILabel: "지도"},
		validationName: "menu map key",
		defaultVK:      0x4D,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Map },
		value:          func(m MenuKeys) KeyBinding { return m.Map },
	},
	{
		definition:     MenuBindingDefinition{ID: "journal", Label: "Journal", UILabel: "일지"},
		validationName: "menu journal key",
		defaultVK:      0x4A,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Journal },
		value:          func(m MenuKeys) KeyBinding { return m.Journal },
	},
	{
		definition:     MenuBindingDefinition{ID: "social", Label: "Social", UILabel: "소셜"},
		validationName: "menu social key",
		defaultVK:      0x4F,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Social },
		value:          func(m MenuKeys) KeyBinding { return m.Social },
	},
	{
		definition:     MenuBindingDefinition{ID: "clan", Label: "Clan", UILabel: "클랜"},
		validationName: "menu clan key",
		defaultVK:      0x4E,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Clan },
		value:          func(m MenuKeys) KeyBinding { return m.Clan },
	},
	{
		definition:     MenuBindingDefinition{ID: "town_portal", Label: "Town Portal", UILabel: "차원문"},
		validationName: "menu town portal key",
		defaultVK:      0x54,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.TownPortal },
		value:          func(m MenuKeys) KeyBinding { return m.TownPortal },
	},
	{
		definition:     MenuBindingDefinition{ID: "collection", Label: "Collection", UILabel: "컬렉션"},
		validationName: "menu collection key",
		defaultVK:      0x59,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Collection },
		value:          func(m MenuKeys) KeyBinding { return m.Collection },
	},
	{
		definition:     MenuBindingDefinition{ID: "shop", Label: "Shop", UILabel: "상점"},
		validationName: "menu shop key",
		defaultVK:      0x50,
		binding:        func(m *MenuKeys) *KeyBinding { return &m.Shop },
		value:          func(m MenuKeys) KeyBinding { return m.Shop },
	},
}

// Config is the full persisted Diablo Helper configuration used by the UI and runners.
type Config struct {
	// Start is the hotkey that starts skill automation.
	Start KeyBinding
	// Stop is the hotkey that stops skill automation.
	Stop KeyBinding
	// Pause is the hotkey that pauses or resumes skill automation.
	Pause KeyBinding
	// Menu contains the configured game menu hotkeys.
	Menu MenuKeys
	// Skills contains the configured skill slots and is normalized to MaxSkills entries for the UI.
	Skills []Skill
	// SkillGapMS is the delay inserted between skill sends in milliseconds.
	SkillGapMS int
	// Clicker contains the mouse click automation bindings and interval.
	Clicker Clicker
}

// Default returns the UI-normalized built-in configuration.
func Default() Config {
	cfg := Config{
		Pause: KeyBinding{Name: "Mouse Right", VK: 0x02},
		Clicker: Clicker{
			Key:        KeyBinding{Name: "Mouse Left", VK: MouseLeftVK},
			IntervalMS: DefaultClickerIntervalMS,
		},
		Menu: defaultMenuKeys(),
	}
	cfg.NormalizeForUI()
	return cfg
}

// MenuBindingDefinitions returns supported menu actions in UI order.
func MenuBindingDefinitions() []MenuBindingDefinition {
	definitions := make([]MenuBindingDefinition, 0, len(menuBindingSpecs))
	for i := range menuBindingSpecs {
		definitions = append(definitions, menuBindingSpecs[i].definition)
	}
	return definitions
}

func defaultMenuKeys() MenuKeys {
	var menu MenuKeys
	for i := range menuBindingSpecs {
		spec := menuBindingSpecs[i]
		*spec.binding(&menu) = KeyBinding{
			Name: KeyDisplayName(spec.defaultVK),
			VK:   spec.defaultVK,
		}
	}
	return menu
}

// SetKeyByID replaces the menu key binding for id and reports whether id was known.
func (m *MenuKeys) SetKeyByID(id string, binding KeyBinding) bool {
	for i := range menuBindingSpecs {
		spec := menuBindingSpecs[i]
		if spec.definition.ID == id {
			*spec.binding(m) = binding
			return true
		}
	}
	return false
}

func (m *MenuKeys) forEachKey(fn func(*KeyBinding)) {
	for i := range menuBindingSpecs {
		fn(menuBindingSpecs[i].binding(m))
	}
}

// Matches reports whether vk matches any assigned menu binding without allocating in the low-level keyboard hook.
func (m MenuKeys) Matches(vk uint16) bool {
	for i := range menuBindingSpecs {
		binding := menuBindingSpecs[i].value(m)
		if binding.Assigned() && uint16(binding.VK) == vk {
			return true
		}
	}
	return false
}

// BindingByID returns the KeyBinding for the given menu ID.
func (m MenuKeys) BindingByID(id string) (KeyBinding, bool) {
	for i := range menuBindingSpecs {
		spec := menuBindingSpecs[i]
		if spec.definition.ID == id {
			return spec.value(m), true
		}
	}
	return KeyBinding{}, false
}

// NormalizeForUI repairs a partially edited config for controls and save output,
// and file loads must Validate before calling it so invalid raw key_name values
// are rejected before names are rewritten from KeyDisplayName.
func (c *Config) NormalizeForUI() {
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

// MenuBindings returns configured menu bindings in UI order.
func (c Config) MenuBindings() []MenuBinding {
	bindings := make([]MenuBinding, 0, len(menuBindingSpecs))
	for i := range menuBindingSpecs {
		spec := menuBindingSpecs[i]
		bindings = append(bindings, MenuBinding{
			ID:      spec.definition.ID,
			Label:   spec.definition.Label,
			Binding: spec.value(c.Menu),
		})
	}
	return bindings
}

// Validate checks config invariants without repairing values, including that
// each stored key_name matches key_vk or an accepted legacy alias before
// NormalizeForUI rewrites names from KeyDisplayName.
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
	} {
		if err := validateKey(item.name, item.binding); err != nil {
			return err
		}
	}
	for i := range menuBindingSpecs {
		spec := menuBindingSpecs[i]
		if err := validateKey(spec.validationName, *spec.binding(&c.Menu)); err != nil {
			return err
		}
	}
	return nil
}

// MillisecondsFitDuration reports whether ms can be represented as a nonnegative time.Duration.
func MillisecondsFitDuration(ms int) bool {
	return ms >= 0 && int64(ms) <= maximumDurationMilliseconds
}

// KeyDisplayName returns the canonical persisted and displayed key_name for a Win32 virtual-key code.
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
	case 0x90:
		return "Num Lock"
	case 0x91:
		return "Scroll Lock"
	case 0xA0:
		return "Left Shift"
	case 0xA1:
		return "Right Shift"
	case 0xA2:
		return "Left Ctrl"
	case 0xA3:
		return "Right Ctrl"
	case 0xA4:
		return "Left Alt"
	case 0xA5:
		return "Right Alt"
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
	if label, forbidden := ForbiddenOutputKeyLabel(binding.VK); forbidden {
		return fmt.Errorf("%s must not be a system key (%s)", name, label)
	}
	return nil
}

// ForbiddenOutputKeyLabel reports whether vk is blocked as an automated output key and returns its label.
func ForbiddenOutputKeyLabel(vk int) (string, bool) {
	return forbiddenOutputKey(vk)
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
	if binding.VK > 0 && !keyNameMatchesVK(binding.Name, binding.VK) {
		return fmt.Errorf("%s name does not match virtual-key code", name)
	}
	return nil
}

func keyNameMatchesVK(name string, vk int) bool {
	if name == KeyDisplayName(vk) {
		return true
	}
	if name == fmt.Sprintf("VK_%d", vk) {
		return acceptsLegacyFallbackName(vk)
	}
	switch vk {
	case 0xA0, 0xA1:
		return name == "Shift"
	case 0xA2, 0xA3:
		return name == "Ctrl"
	case 0xA4, 0xA5:
		return name == "Alt"
	}
	return false
}

func acceptsLegacyFallbackName(vk int) bool {
	return vk == 0x90 || vk == 0x91 || (vk >= 0xA0 && vk <= 0xA5)
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
