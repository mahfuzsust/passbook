package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type AppConfig struct {
	DataDir string `json:"data_dir"`
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".passbook", "config.json")
}

func getAttachmentDir() string {
	return filepath.Join(expandPath(dataDir), "_attachments")
}

func loadConfig() {
	data, err := os.ReadFile(getConfigPath())
	if err == nil {
		var cfg AppConfig
		if json.Unmarshal(data, &cfg) == nil && cfg.DataDir != "" {
			dataDir = cfg.DataDir
		}
	}
	saveConfig()
}

func saveConfig() {
	path := getConfigPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	cfg := AppConfig{DataDir: dataDir}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(path, data, 0600)
}
