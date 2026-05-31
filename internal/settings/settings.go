package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Settings holds user-configurable defaults persisted to disk.
type Settings struct {
	Profile string `json:"profile,omitempty"`
	Region  string `json:"region,omitempty"`
	Output  string `json:"output,omitempty"`
}

// Keys lists the valid configuration keys, in display order.
var Keys = []string{"profile", "region", "output"}

// DefaultPath returns the config file path, honoring XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "r53ctl", "config.json"), nil
	}
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "r53ctl", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "r53ctl", "config.json"), nil
}

// Load reads settings from path. A missing file yields the zero value and no error.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, nil
		}
		return Settings{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return s, nil
}

// Save writes settings to path, creating parent directories. Written with 0600 permissions.
func Save(path string, s Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// Set validates and assigns a configuration value by key.
func (s *Settings) Set(key, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case "profile":
		if value == "" {
			return fmt.Errorf("value for %q cannot be empty; use 'config unset %s' to clear", key, key)
		}
		s.Profile = value
	case "region":
		if value == "" {
			return fmt.Errorf("value for %q cannot be empty; use 'config unset %s' to clear", key, key)
		}
		s.Region = value
	case "output":
		if value != "table" && value != "json" {
			return fmt.Errorf("invalid output %q: must be table or json", value)
		}
		s.Output = value
	default:
		return fmt.Errorf("unknown config key %q: valid keys are %s", key, strings.Join(Keys, ", "))
	}
	return nil
}

// Get returns the stored value for key.
func (s Settings) Get(key string) (string, error) {
	switch key {
	case "profile":
		return s.Profile, nil
	case "region":
		return s.Region, nil
	case "output":
		return s.Output, nil
	default:
		return "", fmt.Errorf("unknown config key %q: valid keys are %s", key, strings.Join(Keys, ", "))
	}
}

// Unset clears a configuration value by key.
func (s *Settings) Unset(key string) error {
	switch key {
	case "profile":
		s.Profile = ""
	case "region":
		s.Region = ""
	case "output":
		s.Output = ""
	default:
		return fmt.Errorf("unknown config key %q: valid keys are %s", key, strings.Join(Keys, ", "))
	}
	return nil
}
