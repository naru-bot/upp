package config

import (
	"os"
	"os/user"
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
			UserAgent:  "upp/1.0",
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
		newDir := filepath.Join(xdg, "upp")
		oldDir := filepath.Join(xdg, "watchdog")
		
		// Migrate from old config directory if needed
		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			if _, err := os.Stat(oldDir); err == nil {
				os.Rename(oldDir, newDir)
			}
		}
		
		return filepath.Join(newDir, "config.yml")
	}
	
	home := getHomeDir()
	newDir := filepath.Join(home, ".config", "upp")
	oldDir := filepath.Join(home, ".config", "watchdog")
	
	// Migrate from old config directory if needed
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		if _, err := os.Stat(oldDir); err == nil {
			os.Rename(oldDir, newDir)
		}
	}
	
	return filepath.Join(newDir, "config.yml")
}

// getHomeDir returns the current user's home directory reliably,
// even in contexts where $HOME is not set (e.g. systemd services).
func getHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return "/"
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
