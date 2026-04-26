package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return ParseTOML(data)
}

func SaveFile(path string, cfg Config) error {
	data, err := MarshalTOML(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func MarshalTOML(cfg Config) ([]byte, error) {
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writeKey(&buf, "start", cfg.Start)
	writeKey(&buf, "stop", cfg.Stop)
	writeKey(&buf, "pause", cfg.Pause)
	buf.WriteByte('\n')
	writeKey(&buf, "menu_inventory", cfg.Menu.Inventory)
	writeKey(&buf, "menu_skills", cfg.Menu.Skills)
	writeKey(&buf, "menu_follower", cfg.Menu.Follower)
	writeKey(&buf, "menu_map", cfg.Menu.Map)
	writeKey(&buf, "menu_world_map", cfg.Menu.WorldMap)
	writeKey(&buf, "menu_town_portal", cfg.Menu.TownPortal)
	writeKey(&buf, "menu_chat", cfg.Menu.Chat)
	writeKey(&buf, "menu_whisper", cfg.Menu.Whisper)
	for _, skill := range cfg.Skills {
		buf.WriteString("\n[[skills]]\n")
		writeString(&buf, "name", skill.Name)
		writeString(&buf, "key_name", skill.Key.Name)
		writeInt(&buf, "key_vk", skill.Key.VK)
		writeInt(&buf, "interval_ms", skill.IntervalMS)
		writeBool(&buf, "enabled", skill.Enabled)
	}
	return buf.Bytes(), nil
}

func ParseTOML(data []byte) (Config, error) {
	cfg := Default()
	cfg.Skills = nil

	var currentSkill *Skill
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if line == "[[skills]]" {
			cfg.Skills = append(cfg.Skills, Skill{
				Name:       fmt.Sprintf("Skill %d", len(cfg.Skills)+1),
				IntervalMS: DefaultIntervalMS,
				Enabled:    true,
			})
			currentSkill = &cfg.Skills[len(cfg.Skills)-1]
			continue
		}
		if strings.HasPrefix(line, "[") {
			return Config{}, fmt.Errorf("line %d: unsupported section %q", lineNumber, line)
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("line %d: expected key = value", lineNumber)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return Config{}, fmt.Errorf("line %d: empty key", lineNumber)
		}

		if currentSkill != nil {
			if err := setSkillValue(currentSkill, key, value); err != nil {
				return Config{}, fmt.Errorf("line %d: %w", lineNumber, err)
			}
			continue
		}
		if err := setTopLevelValue(&cfg, key, value); err != nil {
			return Config{}, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func setTopLevelValue(cfg *Config, key string, value string) error {
	switch key {
	case "start_key_name":
		return setString(&cfg.Start.Name, value)
	case "start_key_vk":
		return setInt(&cfg.Start.VK, value)
	case "stop_key_name":
		return setString(&cfg.Stop.Name, value)
	case "stop_key_vk":
		return setInt(&cfg.Stop.VK, value)
	case "pause_key_name":
		return setString(&cfg.Pause.Name, value)
	case "pause_key_vk":
		return setInt(&cfg.Pause.VK, value)
	case "menu_inventory_key_name":
		return setString(&cfg.Menu.Inventory.Name, value)
	case "menu_inventory_key_vk":
		return setInt(&cfg.Menu.Inventory.VK, value)
	case "menu_skills_key_name":
		return setString(&cfg.Menu.Skills.Name, value)
	case "menu_skills_key_vk":
		return setInt(&cfg.Menu.Skills.VK, value)
	case "menu_follower_key_name":
		return setString(&cfg.Menu.Follower.Name, value)
	case "menu_follower_key_vk":
		return setInt(&cfg.Menu.Follower.VK, value)
	case "menu_map_key_name":
		return setString(&cfg.Menu.Map.Name, value)
	case "menu_map_key_vk":
		return setInt(&cfg.Menu.Map.VK, value)
	case "menu_world_map_key_name":
		return setString(&cfg.Menu.WorldMap.Name, value)
	case "menu_world_map_key_vk":
		return setInt(&cfg.Menu.WorldMap.VK, value)
	case "menu_town_portal_key_name":
		return setString(&cfg.Menu.TownPortal.Name, value)
	case "menu_town_portal_key_vk":
		return setInt(&cfg.Menu.TownPortal.VK, value)
	case "menu_chat_key_name":
		return setString(&cfg.Menu.Chat.Name, value)
	case "menu_chat_key_vk":
		return setInt(&cfg.Menu.Chat.VK, value)
	case "menu_whisper_key_name":
		return setString(&cfg.Menu.Whisper.Name, value)
	case "menu_whisper_key_vk":
		return setInt(&cfg.Menu.Whisper.VK, value)
	default:
		return fmt.Errorf("unknown key %q", key)
	}
}

func setSkillValue(skill *Skill, key string, value string) error {
	switch key {
	case "name":
		return setString(&skill.Name, value)
	case "key_name":
		return setString(&skill.Key.Name, value)
	case "key_vk":
		return setInt(&skill.Key.VK, value)
	case "interval_ms":
		return setInt(&skill.IntervalMS, value)
	case "enabled":
		return setBool(&skill.Enabled, value)
	default:
		return fmt.Errorf("unknown skill key %q", key)
	}
}

func writeKey(buf *bytes.Buffer, prefix string, binding KeyBinding) {
	writeString(buf, prefix+"_key_name", binding.Name)
	writeInt(buf, prefix+"_key_vk", binding.VK)
}

func writeString(buf *bytes.Buffer, key string, value string) {
	buf.WriteString(key)
	buf.WriteString(" = ")
	buf.WriteString(strconv.Quote(value))
	buf.WriteByte('\n')
}

func writeInt(buf *bytes.Buffer, key string, value int) {
	buf.WriteString(key)
	buf.WriteString(" = ")
	buf.WriteString(strconv.Itoa(value))
	buf.WriteByte('\n')
}

func writeBool(buf *bytes.Buffer, key string, value bool) {
	buf.WriteString(key)
	buf.WriteString(" = ")
	buf.WriteString(strconv.FormatBool(value))
	buf.WriteByte('\n')
}

func setString(target *string, value string) error {
	parsed, err := strconv.Unquote(value)
	if err != nil {
		return fmt.Errorf("expected quoted string: %w", err)
	}
	*target = parsed
	return nil
}

func setInt(target *int, value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("expected integer: %w", err)
	}
	*target = parsed
	return nil
}

func setBool(target *bool, value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("expected boolean: %w", err)
	}
	*target = parsed
	return nil
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			inString = !inString
		case r == '#' && !inString:
			return line[:i]
		}
	}
	return line
}
