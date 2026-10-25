package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config represents persistent application settings for Delvo CLI.
type Config struct {
	Roots         []string `json:"roots"`
	DBPath        string   `json:"db_path"`
	TrashDir      string   `json:"trash_dir"`
	Workers       int      `json:"workers"`
	DebounceMs    int      `json:"debounce_ms"`
	WatchEnabled  bool     `json:"watch_enabled"`
	ShowHidden    bool     `json:"show_hidden"`
	CaseSensitive bool     `json:"case_sensitive"`
	Theme         string   `json:"theme"`
}

// DefaultRoots returns accessible default roots for Termux / Linux.
func DefaultRoots() []string {
	var roots []string

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		roots = append(roots, home)

		// Termux shared storage via termux-setup-storage
		sharedStorage := filepath.Join(home, "storage", "shared")
		if info, err := os.Stat(sharedStorage); err == nil && info.IsDir() {
			roots = append(roots, sharedStorage)
		}
	} else {
		// Fallback to current working directory
		if cwd, err := os.Getwd(); err == nil {
			roots = append(roots, cwd)
		}
	}

	return roots
}

// DefaultDBPath returns the path to the local SQLite database.
func DefaultDBPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			dataHome = filepath.Join(home, ".local", "share")
		} else {
			dataHome = ".delvo"
		}
	}
	return filepath.Join(dataHome, "delvo", "delvo.db")
}

// DefaultTrashDir returns the path to Delvo's local trash directory.
func DefaultTrashDir() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			dataHome = filepath.Join(home, ".local", "share")
		} else {
			dataHome = ".delvo"
		}
	}
	return filepath.Join(dataHome, "delvo", "trash")
}

// DefaultConfigPath returns the path to the configuration file.
func DefaultConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			configHome = filepath.Join(home, ".config")
		} else {
			configHome = ".delvo"
		}
	}
	return filepath.Join(configHome, "delvo", "config.json")
}

// NewDefaultConfig returns a Config struct populated with sensible defaults.
func NewDefaultConfig() *Config {
	workers := runtime.NumCPU() * 2
	if workers < 2 {
		workers = 2
	} else if workers > 16 {
		workers = 16
	}

	return &Config{
		Roots:         DefaultRoots(),
		DBPath:        DefaultDBPath(),
		TrashDir:      DefaultTrashDir(),
		Workers:       workers,
		DebounceMs:    80,
		WatchEnabled:  true,
		ShowHidden:    false,
		CaseSensitive: false,
		Theme:         "dark-navy",
	}
}

// Load loads the configuration from disk, creating defaults if not present.
func Load() (*Config, error) {
	configPath := DefaultConfigPath()

	cfg := NewDefaultConfig()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Ensure directory exists and write default config
		if err := cfg.Save(); err != nil {
			return cfg, nil // Ignore save error on first run, use in-memory defaults
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}

	// Sanity checks
	if len(cfg.Roots) == 0 {
		cfg.Roots = DefaultRoots()
	}
	if cfg.DBPath == "" {
		cfg.DBPath = DefaultDBPath()
	}
	if cfg.TrashDir == "" {
		cfg.TrashDir = DefaultTrashDir()
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.DebounceMs <= 0 {
		cfg.DebounceMs = 80
	}

	return cfg, nil
}

// Save writes the current configuration to disk.
func (c *Config) Save() error {
	configPath := DefaultConfigPath()
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
