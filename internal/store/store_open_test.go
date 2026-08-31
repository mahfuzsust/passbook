package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenFreshVault(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")

	s, err := Open(dbPath, "MyStr0ng!Pass")
	if err != nil {
		t.Fatalf("Open fresh vault: %v", err)
	}
	s.Close()
}

func TestOpenWithOrphanWAL(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")

	if err := os.WriteFile(dbPath+"-wal", []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath+"-shm", []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dbPath, "MyStr0ng!Pass")
	if err != nil {
		t.Fatalf("Open with orphan WAL/SHM: %v", err)
	}
	s.Close()
}

func TestOpenWithZeroByteDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")

	if err := os.WriteFile(dbPath, nil, 0600); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dbPath, "MyStr0ng!Pass")
	if err != nil {
		t.Fatalf("Open zero-byte db: %v", err)
	}
	s.Close()
}

func TestOpenSpecialCharPassword(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")

	s, err := Open(dbPath, "p@ss&word=test#foo+bar/baz")
	if err != nil {
		t.Fatalf("Open with special password: %v", err)
	}
	s.Close()
}
