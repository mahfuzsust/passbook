package ui

import (
	"os"
	"path/filepath"
	"testing"

	"passbook/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "passbook.db"), "MyStr0ng!Pass")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestDownloadAttachmentSavesToDownloadsFolder(t *testing.T) {
	s := newTestStore(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	entryID, err := s.SaveEntry(0, &Entry{Type: string(TypeFile), Title: "Files"})
	if err != nil {
		t.Fatalf("save entry: %v", err)
	}
	if err := s.WriteAttachment("att1", entryID, "secret.txt", 5, []byte("hello")); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	m := &Model{store: s}
	m.downloadAttachment(Attachment{ID: "att1", FileName: "secret.txt", Size: 5})

	dest := filepath.Join(home, "Downloads", "secret.txt")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("expected file at %s, got error: %v", dest, err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected downloaded content %q, got %q", "hello", data)
	}
	if m.main.viewStatus == "" {
		t.Fatalf("expected a status message after download")
	}
}

func TestDownloadAttachmentAvoidsOverwrite(t *testing.T) {
	s := newTestStore(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	entryID, _ := s.SaveEntry(0, &Entry{Type: string(TypeFile), Title: "Files"})
	_ = s.WriteAttachment("att1", entryID, "secret.txt", 5, []byte("first"))
	_ = s.WriteAttachment("att2", entryID, "secret.txt", 6, []byte("second"))

	m := &Model{store: s}
	m.downloadAttachment(Attachment{ID: "att1", FileName: "secret.txt", Size: 5})
	m.downloadAttachment(Attachment{ID: "att2", FileName: "secret.txt", Size: 6})

	first, err := os.ReadFile(filepath.Join(home, "Downloads", "secret.txt"))
	if err != nil {
		t.Fatalf("expected first file: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(home, "Downloads", "secret (1).txt"))
	if err != nil {
		t.Fatalf("expected second file with suffixed name: %v", err)
	}
	if string(first) != "first" || string(second) != "second" {
		t.Fatalf("unexpected file contents: %q %q", first, second)
	}
}

func TestDownloadAttachmentMissingShowsError(t *testing.T) {
	s := newTestStore(t)
	t.Setenv("HOME", t.TempDir())

	m := &Model{store: s}
	m.downloadAttachment(Attachment{ID: "does-not-exist", FileName: "x.txt"})

	if m.overlay != overlayError {
		t.Fatalf("expected error overlay when attachment can't be read, got %v", m.overlay)
	}
}

func TestHandleViewActionDownloadsNumberedAttachment(t *testing.T) {
	s := newTestStore(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	entryID, _ := s.SaveEntry(0, &Entry{Type: string(TypeFile), Title: "Files"})
	_ = s.WriteAttachment("att1", entryID, "one.txt", 3, []byte("one"))

	ent, err := s.LoadEntry(entryID)
	if err != nil {
		t.Fatalf("load entry: %v", err)
	}
	m := &Model{store: s, main: mainModel{currentEnt: ent, currentEntryID: entryID}}

	handled, _ := m.handleViewAction("1")
	if !handled {
		t.Fatalf("expected digit key to be handled")
	}
	if _, err := os.Stat(filepath.Join(home, "Downloads", "one.txt")); err != nil {
		t.Fatalf("expected attachment to be downloaded: %v", err)
	}
}

func TestUniqueDownloadPathAppendsSuffix(t *testing.T) {
	dir := t.TempDir()
	first := uniqueDownloadPath(dir, "file.txt")
	if first != filepath.Join(dir, "file.txt") {
		t.Fatalf("expected first path unchanged, got %q", first)
	}
	if err := os.WriteFile(first, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	second := uniqueDownloadPath(dir, "file.txt")
	if second != filepath.Join(dir, "file (1).txt") {
		t.Fatalf("expected suffixed path, got %q", second)
	}
}
