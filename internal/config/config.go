package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

// Config represents the ThreadMine configuration
type Config struct {
	file *ini.File
}

// Load reads the configuration file from ~/.threadmine/config
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".threadmine", "config")

	// If config file doesn't exist, return empty config (not an error)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{file: ini.Empty()}, nil
	}

	file, err := ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	return &Config{file: file}, nil
}

// GetString retrieves a string value from the config
// section.key format (e.g., "fetch.slack.workspace")
func (c *Config) GetString(key string) string {
	section, keyName := c.parseKey(key)
	if section == "" {
		return ""
	}

	sec := c.file.Section(section)
	if sec == nil {
		return ""
	}

	return sec.Key(keyName).String()
}

// GetInt retrieves an integer value from the config
func (c *Config) GetInt(key string) (int, error) {
	val := c.GetString(key)
	if val == "" {
		return 0, nil
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s: %w", key, err)
	}

	return intVal, nil
}

// GetBool retrieves a boolean value from the config
func (c *Config) GetBool(key string) bool {
	val := c.GetString(key)
	if val == "" {
		return false
	}

	val = strings.ToLower(val)
	return val == "true" || val == "yes" || val == "1" || val == "on"
}

// HasKey checks if a key exists in the config
func (c *Config) HasKey(key string) bool {
	section, keyName := c.parseKey(key)
	if section == "" {
		return false
	}

	sec := c.file.Section(section)
	if sec == nil {
		return false
	}

	return sec.HasKey(keyName)
}

// parseKey splits a dotted key into section and key name
// e.g., "fetch.slack.workspace" -> ("fetch.slack", "workspace")
// For Git config compatibility, we use the last dot as the separator
func (c *Config) parseKey(key string) (string, string) {
	lastDot := strings.LastIndex(key, ".")
	if lastDot == -1 {
		return "", ""
	}

	section := key[:lastDot]
	keyName := key[lastDot+1:]

	return section, keyName
}

// GetStringWithFallback retrieves a string value with a fallback default
func (c *Config) GetStringWithFallback(key, fallback string) string {
	if c.HasKey(key) {
		return c.GetString(key)
	}
	return fallback
}

// GetIntWithFallback retrieves an int value with a fallback default
func (c *Config) GetIntWithFallback(key string, fallback int) int {
	if c.HasKey(key) {
		val, err := c.GetInt(key)
		if err == nil {
			return val
		}
	}
	return fallback
}
