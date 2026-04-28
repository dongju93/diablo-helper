package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type SaveOptions struct {
	AllowNonTOMLExtension bool
}

func LoadFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return Config{}, err
	}
	if fi.Size() > MaxConfigFileBytes {
		return Config{}, fmt.Errorf("config file too large (%d bytes, max %d)", fi.Size(), MaxConfigFileBytes)
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return Config{}, err
	}
	return ParseTOML(data)
}

func SaveFile(path string, cfg Config) error {
	return SaveFileWithOptions(path, cfg, SaveOptions{})
}

func SaveFileWithOptions(path string, cfg Config, opts SaveOptions) error {
	data, err := MarshalTOML(cfg)
	if err != nil {
		return err
	}
	if err := validateSavePath(path, opts); err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o600)
}

func HasTOMLExtension(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".toml")
}

func validateSavePath(path string, opts SaveOptions) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is empty")
	}
	if !opts.AllowNonTOMLExtension && !HasTOMLExtension(path) {
		return fmt.Errorf("config file extension must be .toml")
	}
	if err := rejectReparsePath(path); err != nil {
		return err
	}
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if err = tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = rejectReparsePath(path); err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	return syncDir(dir)
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer d.Close()
	_ = d.Sync()
	return nil
}

func rejectReparsePath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := rejectReparsePathComponent(abs); err != nil {
		return err
	}
	for dir := filepath.Dir(abs); dir != "."; dir = filepath.Dir(dir) {
		if err := rejectReparsePathComponent(dir); err != nil {
			return err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil
}

func rejectReparsePathComponent(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fileInfoIsReparsePoint(info) {
		return fmt.Errorf("config path must not use a symlink or reparse point: %s", path)
	}
	return nil
}

func fileInfoIsReparsePoint(info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	sys := info.Sys()
	if sys == nil {
		return false
	}
	value := reflect.ValueOf(sys)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return false
	}
	attributes := value.FieldByName("FileAttributes")
	if !attributes.IsValid() || !attributes.CanUint() {
		return false
	}
	return attributes.Uint()&0x400 != 0
}

func MarshalTOML(cfg Config) ([]byte, error) {
	cfg.NormalizeForUI()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writeKey(&buf, "start", cfg.Start)
	writeKey(&buf, "stop", cfg.Stop)
	writeKey(&buf, "pause", cfg.Pause)
	writeInt(&buf, "skill_gap_ms", cfg.SkillGapMS)
	buf.WriteByte('\n')
	writeKey(&buf, "clicker_start", cfg.Clicker.Start)
	writeKey(&buf, "clicker_stop", cfg.Clicker.Stop)
	writeKey(&buf, "clicker", cfg.Clicker.Key)
	writeInt(&buf, "clicker_interval_ms", cfg.Clicker.IntervalMS)
	buf.WriteByte('\n')
	writeKey(&buf, "menu_character", cfg.Menu.Character)
	writeKey(&buf, "menu_skill_assign", cfg.Menu.SkillAssign)
	writeKey(&buf, "menu_talents", cfg.Menu.Talents)
	writeKey(&buf, "menu_map", cfg.Menu.Map)
	writeKey(&buf, "menu_journal", cfg.Menu.Journal)
	writeKey(&buf, "menu_social", cfg.Menu.Social)
	writeKey(&buf, "menu_clan", cfg.Menu.Clan)
	writeKey(&buf, "menu_town_portal", cfg.Menu.TownPortal)
	writeKey(&buf, "menu_collection", cfg.Menu.Collection)
	writeKey(&buf, "menu_shop", cfg.Menu.Shop)
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
			if len(cfg.Skills) >= MaxSkills {
				return Config{}, fmt.Errorf("line %d: too many [[skills]] sections (max %d)", lineNumber, MaxSkills)
			}
			cfg.Skills = append(cfg.Skills, Skill{
				Name:       fmt.Sprintf("Skill %d", len(cfg.Skills)+1),
				IntervalMS: DefaultIntervalMS,
				Enabled:    DefaultSkillEnabled,
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

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	cfg.NormalizeForUI()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func setTopLevelValue(cfg *Config, key string, value string) error {
	switch key {
	case "start_key_name":
		return setKeyName(&cfg.Start.Name, key, value)
	case "start_key_vk":
		return setInt(&cfg.Start.VK, value)
	case "stop_key_name":
		return setKeyName(&cfg.Stop.Name, key, value)
	case "stop_key_vk":
		return setInt(&cfg.Stop.VK, value)
	case "pause_key_name":
		return setKeyName(&cfg.Pause.Name, key, value)
	case "pause_key_vk":
		return setInt(&cfg.Pause.VK, value)
	case "skill_gap_ms":
		return setInt(&cfg.SkillGapMS, value)
	case "clicker_start_key_name":
		return setKeyName(&cfg.Clicker.Start.Name, key, value)
	case "clicker_start_key_vk":
		return setInt(&cfg.Clicker.Start.VK, value)
	case "clicker_stop_key_name":
		return setKeyName(&cfg.Clicker.Stop.Name, key, value)
	case "clicker_stop_key_vk":
		return setInt(&cfg.Clicker.Stop.VK, value)
	case "clicker_key_name":
		return setKeyName(&cfg.Clicker.Key.Name, key, value)
	case "clicker_key_vk":
		return setInt(&cfg.Clicker.Key.VK, value)
	case "clicker_interval_ms":
		return setInt(&cfg.Clicker.IntervalMS, value)
	case "menu_character_key_name":
		return setKeyName(&cfg.Menu.Character.Name, key, value)
	case "menu_character_key_vk":
		return setInt(&cfg.Menu.Character.VK, value)
	case "menu_skill_assign_key_name":
		return setKeyName(&cfg.Menu.SkillAssign.Name, key, value)
	case "menu_skill_assign_key_vk":
		return setInt(&cfg.Menu.SkillAssign.VK, value)
	case "menu_talents_key_name":
		return setKeyName(&cfg.Menu.Talents.Name, key, value)
	case "menu_talents_key_vk":
		return setInt(&cfg.Menu.Talents.VK, value)
	case "menu_map_key_name":
		return setKeyName(&cfg.Menu.Map.Name, key, value)
	case "menu_map_key_vk":
		return setInt(&cfg.Menu.Map.VK, value)
	case "menu_journal_key_name":
		return setKeyName(&cfg.Menu.Journal.Name, key, value)
	case "menu_journal_key_vk":
		return setInt(&cfg.Menu.Journal.VK, value)
	case "menu_social_key_name":
		return setKeyName(&cfg.Menu.Social.Name, key, value)
	case "menu_social_key_vk":
		return setInt(&cfg.Menu.Social.VK, value)
	case "menu_clan_key_name":
		return setKeyName(&cfg.Menu.Clan.Name, key, value)
	case "menu_clan_key_vk":
		return setInt(&cfg.Menu.Clan.VK, value)
	case "menu_town_portal_key_name":
		return setKeyName(&cfg.Menu.TownPortal.Name, key, value)
	case "menu_town_portal_key_vk":
		return setInt(&cfg.Menu.TownPortal.VK, value)
	case "menu_collection_key_name":
		return setKeyName(&cfg.Menu.Collection.Name, key, value)
	case "menu_collection_key_vk":
		return setInt(&cfg.Menu.Collection.VK, value)
	case "menu_shop_key_name":
		return setKeyName(&cfg.Menu.Shop.Name, key, value)
	case "menu_shop_key_vk":
		return setInt(&cfg.Menu.Shop.VK, value)
	default:
		return fmt.Errorf("unknown key %q", key)
	}
}

func setSkillValue(skill *Skill, key string, value string) error {
	switch key {
	case "name":
		return setString(&skill.Name, value, "skill name", MaxSkillNameLength)
	case "key_name":
		return setKeyName(&skill.Key.Name, key, value)
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

func setKeyName(target *string, key string, value string) error {
	return setString(target, value, key, MaxKeyNameLength)
}

func setString(target *string, value string, name string, maxLength int) error {
	parsed, err := strconv.Unquote(value)
	if err != nil {
		return fmt.Errorf("expected quoted string: %w", err)
	}
	if err := validateConfigString(name, parsed, maxLength); err != nil {
		return err
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
