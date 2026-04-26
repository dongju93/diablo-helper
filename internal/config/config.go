package config

import "fmt"

const (
	MaxSkills         = 8
	DefaultIntervalMS = 1000
	MinimumIntervalMS = 10
	MouseLeftVK       = 0x01
)

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

type MenuBinding struct {
	ID      string
	Label   string
	Binding KeyBinding
}

type Config struct {
	Start  KeyBinding
	Stop   KeyBinding
	Pause  KeyBinding
	Menu   MenuKeys
	Skills []Skill
}

func Default() Config {
	cfg := Config{
		Pause: KeyBinding{Name: "Mouse Right", VK: 0x02},
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

func (c *Config) Normalize() {
	if len(c.Skills) > MaxSkills {
		c.Skills = c.Skills[:MaxSkills]
	}
	for len(c.Skills) < MaxSkills {
		index := len(c.Skills) + 1
		c.Skills = append(c.Skills, Skill{
			Name:       fmt.Sprintf("Skill %d", index),
			IntervalMS: DefaultIntervalMS,
			Enabled:    true,
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
	normalizeKey(&c.Start)
	normalizeKey(&c.Stop)
	normalizeKey(&c.Pause)
	normalizeKey(&c.Menu.Inventory)
	normalizeKey(&c.Menu.Skills)
	normalizeKey(&c.Menu.Follower)
	normalizeKey(&c.Menu.Map)
	normalizeKey(&c.Menu.WorldMap)
	normalizeKey(&c.Menu.TownPortal)
	normalizeKey(&c.Menu.Chat)
	normalizeKey(&c.Menu.Whisper)
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
	for i, skill := range c.Skills {
		if skill.IntervalMS < MinimumIntervalMS {
			return fmt.Errorf("skill %d interval must be at least %dms", i+1, MinimumIntervalMS)
		}
		if err := validateKey("skill "+skill.Name, skill.Key); err != nil {
			return err
		}
	}
	for name, binding := range map[string]KeyBinding{
		"start":            c.Start,
		"stop":             c.Stop,
		"pause":            c.Pause,
		"menu inventory":   c.Menu.Inventory,
		"menu skills":      c.Menu.Skills,
		"menu follower":    c.Menu.Follower,
		"menu map":         c.Menu.Map,
		"menu world map":   c.Menu.WorldMap,
		"menu town portal": c.Menu.TownPortal,
		"menu chat":        c.Menu.Chat,
		"menu whisper":     c.Menu.Whisper,
	} {
		if err := validateKey(name, binding); err != nil {
			return err
		}
	}
	return nil
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
	if binding.VK < 0 || binding.VK > 255 {
		return fmt.Errorf("%s key vk must be between 0 and 255", name)
	}
	if binding.VK == 0 && binding.Name != "" {
		return fmt.Errorf("%s key has a name but no virtual-key code", name)
	}
	return nil
}
