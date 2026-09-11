package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPathTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := ExpandPath("~/data")
	want := filepath.Join(home, "data")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExpandPathNoTilde(t *testing.T) {
	got := ExpandPath("/absolute/path")
	if got != "/absolute/path" {
		t.Fatalf("expected path unchanged, got %q", got)
	}
}

func TestExistsFalseWhenNoConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if Exists() {
		t.Fatalf("did not expect a config file to exist")
	}
}

func TestSaveAndExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := Save(AppConfig{DataDir: "~/.passbook/data"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if !Exists() {
		t.Fatalf("expected config to exist after Save")
	}
}

func TestLoadOrInitCreatesDefaultConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := LoadOrInit()
	if cfg.DataDir != "~/.passbook/data" {
		t.Fatalf("expected default data dir, got %q", cfg.DataDir)
	}
	if !Exists() {
		t.Fatalf("expected LoadOrInit to persist the default config")
	}
}

func TestLoadOrInitLoadsExistingConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := Save(AppConfig{DataDir: "/custom/data"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	cfg := LoadOrInit()
	if cfg.DataDir != "/custom/data" {
		t.Fatalf("expected loaded custom data dir, got %q", cfg.DataDir)
	}
}

func TestLoadOrInitFallsBackOnCorruptConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".passbook", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := LoadOrInit()
	if cfg.DataDir != "~/.passbook/data" {
		t.Fatalf("expected default data dir on corrupt config, got %q", cfg.DataDir)
	}
}

func TestLoadOrInitFallsBackOnEmptyDataDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Save(AppConfig{DataDir: ""}); err != nil {
		t.Fatal(err)
	}

	cfg := LoadOrInit()
	if cfg.DataDir != "~/.passbook/data" {
		t.Fatalf("expected default data dir when saved dir is empty, got %q", cfg.DataDir)
	}
}
