package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Config struct {
	Version  int
	Disabled []string
}

func DefaultConfig() Config {
	return Config{Version: 1, Disabled: []string{}}
}

func (c *Config) Normalize() {
	seen := make(map[string]bool, len(c.Disabled))
	result := make([]string, 0, len(c.Disabled))
	for _, name := range c.Disabled {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	sort.Strings(result)
	c.Disabled = result
}

func (c Config) DisabledSet() map[string]bool {
	set := make(map[string]bool, len(c.Disabled))
	for _, name := range c.Disabled {
		set[name] = true
	}
	return set
}

func LoadConfig(path string) (Config, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg, err := ParseConfig(string(data))
	if err != nil {
		return Config{}, true, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, true, nil
}

func ParseConfig(content string) (Config, error) {
	cfg := Config{}
	seenVersion := false
	seenDisabled := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripTOMLComment(scanner.Text()))
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("line %d: expected key = value", lineNo)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "version":
			if seenVersion {
				return Config{}, fmt.Errorf("line %d: duplicate version", lineNo)
			}
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return Config{}, fmt.Errorf("line %d: version must be an integer", lineNo)
			}
			cfg.Version = parsed
			seenVersion = true
		case "disabled":
			if seenDisabled {
				return Config{}, fmt.Errorf("line %d: duplicate disabled", lineNo)
			}
			arrayText := value
			for !arrayClosed(arrayText) {
				if !scanner.Scan() {
					return Config{}, fmt.Errorf("line %d: unterminated disabled array", lineNo)
				}
				lineNo++
				next := strings.TrimSpace(stripTOMLComment(scanner.Text()))
				if next != "" {
					arrayText += "\n" + next
				}
			}
			parsed, err := parseStringArray(arrayText)
			if err != nil {
				return Config{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			cfg.Disabled = parsed
			seenDisabled = true
		default:
			return Config{}, fmt.Errorf("line %d: unknown field %q", lineNo, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("scan config: %w", err)
	}
	if !seenVersion {
		return Config{}, fmt.Errorf("missing required field version")
	}
	if cfg.Version != 1 {
		return Config{}, fmt.Errorf("unsupported config version %d; supported versions: 1", cfg.Version)
	}
	if !seenDisabled {
		return Config{}, fmt.Errorf("missing required field disabled")
	}
	return cfg, nil
}

func stripTOMLComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if inString && r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return line[:i]
		}
	}
	return line
}

func arrayClosed(value string) bool {
	inString := false
	escaped := false
	depth := 0
	for _, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if inString && r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch r {
		case '[':
			depth++
		case ']':
			depth--
		}
	}
	return depth == 0 && strings.Contains(value, "[")
}

func parseStringArray(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '[' {
		return nil, fmt.Errorf("disabled must be an array")
	}
	closing := strings.LastIndex(value, "]")
	if closing < 0 {
		return nil, fmt.Errorf("unterminated disabled array")
	}
	if strings.TrimSpace(value[closing+1:]) != "" {
		return nil, fmt.Errorf("unexpected content after disabled array")
	}
	body := value[1:closing]
	result := []string{}
	for i := 0; i < len(body); {
		for i < len(body) && (body[i] == ' ' || body[i] == '\t' || body[i] == '\r' || body[i] == '\n' || body[i] == ',') {
			i++
		}
		if i >= len(body) {
			break
		}
		if body[i] != '"' {
			return nil, fmt.Errorf("disabled entries must be double-quoted strings")
		}
		start := i
		i++
		escaped := false
		for i < len(body) {
			if escaped {
				escaped = false
				i++
				continue
			}
			if body[i] == '\\' {
				escaped = true
				i++
				continue
			}
			if body[i] == '"' {
				i++
				break
			}
			i++
		}
		if i > len(body) || body[i-1] != '"' {
			return nil, fmt.Errorf("unterminated string in disabled array")
		}
		decoded, err := strconv.Unquote(body[start:i])
		if err != nil {
			return nil, fmt.Errorf("invalid string in disabled array: %w", err)
		}
		if decoded == "" {
			return nil, fmt.Errorf("disabled skill names must not be empty")
		}
		result = append(result, decoded)
		for i < len(body) && (body[i] == ' ' || body[i] == '\t' || body[i] == '\r' || body[i] == '\n') {
			i++
		}
		if i < len(body) && body[i] != ',' {
			return nil, fmt.Errorf("expected comma between disabled entries")
		}
	}
	return result, nil
}

func FormatConfig(cfg Config) string {
	cfg.Normalize()
	var b strings.Builder
	b.WriteString("# Project-local overrides for user-scoped Codex skills.\n")
	b.WriteString("version = 1\n\n")
	b.WriteString("disabled = [\n")
	for _, name := range cfg.Disabled {
		fmt.Fprintf(&b, "  %s,\n", strconv.Quote(name))
	}
	b.WriteString("]\n")
	return b.String()
}

func SaveConfig(path string, cfg Config) error {
	cfg.Normalize()
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d; supported versions: 1", cfg.Version)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".user-skills-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return fmt.Errorf("set temporary config permissions: %w", err)
	}
	if _, err := temp.WriteString(FormatConfig(cfg)); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}
