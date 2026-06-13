package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all hook2feishu configuration.
type Config struct {
	OpenID     string   `json:"open_id"`
	Events     []string `json:"events"`
	QuietHours []int    `json:"quiet_hours"`
	MinDuration int     `json:"min_duration"`
	TZOffset   int      `json:"tz_offset"`
	// lark-cli settings (optional)
	LarkCLIPath    string `json:"lark_cli_path"`
	LarkCLIProfile string `json:"lark_cli_profile"`

	// Tool overrides auto-detection: "claude", "codex", or empty (auto).
	Tool string `json:"tool"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Events:      []string{"Stop", "Notification"},
		TZOffset:    8,
		MinDuration: 0,
	}
}

// Load reads config.json from (in priority order):
//  1. HOOK2FEISHU_CONFIG env var
//  2. config.json next to the executable
//  3. ~/.config/hook2feishu/config.json
func Load() (*Config, error) {
	paths := configPaths()

	var lastErr error
	for _, p := range paths {
		cfg, err := loadPath(p)
		if err == nil {
			return cfg, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("config not found in %v: %w", paths, lastErr)
}

func configPaths() []string {
	var paths []string

	// 1. Env override
	if env := os.Getenv("HOOK2FEISHU_CONFIG"); env != "" {
		paths = append(paths, env)
	}

	// 2. Next to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		paths = append(paths, filepath.Join(dir, "config.json"))
	}

	// 3. User config dir
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "hook2feishu", "config.json"))
	}

	return paths
}

func loadPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.OpenID == "" {
		return nil, fmt.Errorf("%s: missing required field (open_id)", path)
	}
	return cfg, nil
}

// DataDir returns the hook2feishu data directory.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hook2feishu"), nil
}
