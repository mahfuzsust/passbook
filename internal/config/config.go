package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type AppConfig struct {
	DataDir string `json:"data_dir"`
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".passbook", "config.json")
}

func LoadOrInit() AppConfig {
	cfg := AppConfig{DataDir: "~/.passbook/data"}

	data, err := os.ReadFile(configPath())
	if err == nil {
		var loaded AppConfig
		if json.Unmarshal(data, &loaded) == nil && loaded.DataDir != "" {
			cfg.DataDir = loaded.DataDir
		}
	}

	_ = Save(cfg)
	return cfg
}

func Save(cfg AppConfig) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
