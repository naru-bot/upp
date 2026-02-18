package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Defaults   Defaults          `yaml:"defaults"`
	Display    Display           `yaml:"display"`
	Headers    map[string]string `yaml:"headers,omitempty"`
}

type Defaults struct {
	Interval    int    `yaml:"interval"`       // default check interval in seconds
	Type        string `yaml:"type"`           // default check type
	Timeout     int    `yaml:"timeout"`        // HTTP timeout in seconds
	RetryCount  int    `yaml:"retry_count"`    // retries before marking down
	UserAgent   string `yaml:"user_agent"`
}

type Display struct {
	Color   bool   `yaml:"color"`
	Format  string `yaml:"format"` // table, json, compact
	Verbose bool   `yaml:"verbose"`
}

var current *Config

func Default() *Config {
	return &Config{
		Defaults: Defaults{
			Interval:   300,
			Type:       "http",
			Timeout:    30,
			RetryCount: 1,
			UserAgent:  "watchdog/1.0",
		},
		Display: Display{
			Color:   true,
			Format:  "table",
			Verbose: false,
		},
	}
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "watchdog", "config.yml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "watchdog", "config.yml")
}

func Load() *Config {
	if current != nil {
		return current
	}

	current = Default()

	data, err := os.ReadFile(configPath())
	if err != nil {
		return current
	}

	yaml.Unmarshal(data, current)
	return current
}

func Save(cfg *Config) error {
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0755)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Get() *Config {
	if current == nil {
		return Load()
	}
	return current
}
