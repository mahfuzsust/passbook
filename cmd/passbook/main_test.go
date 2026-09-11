package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"passbook/internal/config"
)

func TestMoveDBFilesMovesExistingFiles(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(oldDir, "passbook.db"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "passbook.db-wal"), []byte("wal"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := moveDBFiles(oldDir, newDir); err != nil {
		t.Fatalf("moveDBFiles: %v", err)
	}

	if _, err := os.Stat(filepath.Join(newDir, "passbook.db")); err != nil {
		t.Fatalf("expected passbook.db to be moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, "passbook.db-wal")); err != nil {
		t.Fatalf("expected passbook.db-wal to be moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(oldDir, "passbook.db")); !os.IsNotExist(err) {
		t.Fatalf("expected source file to be gone, err=%v", err)
	}
}

func TestMoveDBFilesNoFilesToMove(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	if err := moveDBFiles(oldDir, newDir); err != nil {
		t.Fatalf("expected no error when there is nothing to move, got %v", err)
	}
}

func TestMoveDBFilesDestinationExists(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(oldDir, "passbook.db"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "passbook.db"), []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := moveDBFiles(oldDir, newDir); err == nil {
		t.Fatalf("expected an error when destination already exists")
	}
}

func TestSetupICloudAlreadyEnabled(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("setupICloud exits the process on non-macOS")
	}
	t.Setenv("HOME", t.TempDir())

	if err := config.Save(config.AppConfig{DataDir: iCloudDataDir}); err != nil {
		t.Fatal(err)
	}

	setupICloud() // should print "already enabled" and return without exiting
}

func TestSetupICloudMovesExistingVault(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("setupICloud exits the process on non-macOS")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldDataDir := filepath.Join(home, "olddata")
	if err := config.Save(config.AppConfig{DataDir: oldDataDir}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(oldDataDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDataDir, "passbook.db"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	setupICloud()

	cfg := config.LoadOrInit()
	if cfg.DataDir != iCloudDataDir {
		t.Fatalf("expected config to point at iCloud dir, got %q", cfg.DataDir)
	}
	newDataDir := config.ExpandPath(iCloudDataDir)
	if _, err := os.Stat(filepath.Join(newDataDir, "passbook.db")); err != nil {
		t.Fatalf("expected db to be moved to iCloud dir: %v", err)
	}
}
