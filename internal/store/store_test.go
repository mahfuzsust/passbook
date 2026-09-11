package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "passbook.db"), "MyStr0ng!Pass")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestFolderCRUD(t *testing.T) {
	s := newTestStore(t)

	id, err := s.CreateFolder("Work")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero folder id")
	}

	folders, err := s.ListFolders()
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	if len(folders) != 1 || folders[0].Name != "Work" {
		t.Fatalf("unexpected folders: %+v", folders)
	}

	f, err := s.GetFolder(id)
	if err != nil || f == nil || f.Name != "Work" {
		t.Fatalf("get folder: %+v, err=%v", f, err)
	}

	f2, err := s.GetFolderByName("Work")
	if err != nil || f2 == nil || f2.ID != id {
		t.Fatalf("get folder by name: %+v, err=%v", f2, err)
	}

	if err := s.RenameFolder(id, "Personal"); err != nil {
		t.Fatalf("rename folder: %v", err)
	}
	renamed, _ := s.GetFolder(id)
	if renamed == nil || renamed.Name != "Personal" {
		t.Fatalf("expected renamed folder, got %+v", renamed)
	}

	if err := s.DeleteFolder(id); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	if deleted, _ := s.GetFolder(id); deleted != nil {
		t.Fatalf("expected folder to be gone, got %+v", deleted)
	}
}

func TestGetFolderMissingReturnsNil(t *testing.T) {
	s := newTestStore(t)
	f, err := s.GetFolder(999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		t.Fatalf("expected nil folder, got %+v", f)
	}

	f2, err := s.GetFolderByName("does-not-exist")
	if err != nil || f2 != nil {
		t.Fatalf("expected nil folder by name, got %+v err=%v", f2, err)
	}
}

func TestEntryCRUD(t *testing.T) {
	s := newTestStore(t)
	folderID, _ := s.CreateFolder("Web")

	entryID, err := s.SaveEntry(folderID, &EntryFull{
		Type: "Login", Title: "GitHub", Username: "alice", Password: "secret",
		Link: "https://github.com", TotpSecret: "ABC",
	})
	if err != nil {
		t.Fatalf("save entry: %v", err)
	}

	loaded, err := s.LoadEntry(entryID)
	if err != nil {
		t.Fatalf("load entry: %v", err)
	}
	if loaded.Title != "GitHub" || loaded.Username != "alice" || loaded.FolderID != folderID {
		t.Fatalf("unexpected loaded entry: %+v", loaded)
	}

	meta, err := s.GetEntryMeta(entryID)
	if err != nil || meta == nil || meta.Title != "GitHub" {
		t.Fatalf("get entry meta: %+v err=%v", meta, err)
	}

	entries, err := s.ListEntries(folderID)
	if err != nil || len(entries) != 1 {
		t.Fatalf("list entries: %+v err=%v", entries, err)
	}

	all, err := s.ListAllEntries()
	if err != nil || len(all) != 1 {
		t.Fatalf("list all entries: %+v err=%v", all, err)
	}

	if err := s.UpdateEntryFull(entryID, folderID, &EntryFull{
		Type: "Login", Title: "GitHub Updated", Username: "bob", Password: "newsecret",
	}); err != nil {
		t.Fatalf("update entry: %v", err)
	}
	updated, _ := s.LoadEntry(entryID)
	if updated.Title != "GitHub Updated" || updated.Username != "bob" {
		t.Fatalf("expected updated fields, got %+v", updated)
	}

	if !s.HasEntries() {
		t.Fatalf("expected HasEntries to be true")
	}
	if s.CountEntriesInFolder(folderID) != 1 {
		t.Fatalf("expected 1 entry in folder")
	}

	if err := s.DeleteEntry(entryID); err != nil {
		t.Fatalf("delete entry: %v", err)
	}
	if s.HasEntries() {
		t.Fatalf("expected no entries after delete")
	}
}

func TestLoadEntryMissingReturnsError(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.LoadEntry(12345); err == nil {
		t.Fatalf("expected an error loading a missing entry")
	}
}

func TestGetEntryMetaMissingReturnsNil(t *testing.T) {
	s := newTestStore(t)
	meta, err := s.GetEntryMeta(12345)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != nil {
		t.Fatalf("expected nil meta, got %+v", meta)
	}
}

func TestEntryExistsInFolder(t *testing.T) {
	s := newTestStore(t)
	entryID, _ := s.SaveEntry(0, &EntryFull{Type: "Note", Title: "Dup"})

	if !s.EntryExistsInFolder(0, "Dup") {
		t.Fatalf("expected entry to exist in root folder")
	}
	if s.EntryExistsInFolder(0, "Missing") {
		t.Fatalf("did not expect a nonexistent title to be found")
	}
	if s.EntryExistsInFolderExcluding(0, "Dup", entryID) {
		t.Fatalf("expected exclusion of the entry's own id")
	}
	otherID, _ := s.SaveEntry(0, &EntryFull{Type: "Note", Title: "Other"})
	if !s.EntryExistsInFolderExcluding(0, "Dup", otherID) {
		t.Fatalf("expected the excluding check to still find Dup when excluding a different id")
	}
}

func TestPasswordHistory(t *testing.T) {
	s := newTestStore(t)
	entryID, err := s.SaveEntry(0, &EntryFull{
		Type: "Login", Title: "Site",
		History: []PasswordHistory{{Password: "old1", Date: "2024-01-01"}},
	})
	if err != nil {
		t.Fatalf("save entry: %v", err)
	}
	loaded, _ := s.LoadEntry(entryID)
	if len(loaded.History) != 1 || loaded.History[0].Password != "old1" {
		t.Fatalf("expected saved history, got %+v", loaded.History)
	}

	if err := s.UpdateEntryFull(entryID, 0, &EntryFull{
		Type: "Login", Title: "Site",
		History: []PasswordHistory{
			{Password: "old1", Date: "2024-01-01"},
			{Password: "old2", Date: "2024-02-01"},
		},
	}); err != nil {
		t.Fatalf("update entry: %v", err)
	}
	loaded2, _ := s.LoadEntry(entryID)
	if len(loaded2.History) != 2 {
		t.Fatalf("expected replaced history with 2 entries, got %+v", loaded2.History)
	}
}

func TestAttachmentCRUD(t *testing.T) {
	s := newTestStore(t)
	entryID, _ := s.SaveEntry(0, &EntryFull{Type: "File", Title: "Files"})

	if err := s.WriteAttachment("att1", entryID, "a.txt", 5, []byte("hello")); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	data, err := s.ReadAttachment("att1")
	if err != nil || string(data) != "hello" {
		t.Fatalf("read attachment: %q err=%v", data, err)
	}

	// upsert should overwrite
	if err := s.WriteAttachment("att1", entryID, "a.txt", 5, []byte("world")); err != nil {
		t.Fatalf("upsert attachment: %v", err)
	}
	data2, _ := s.ReadAttachment("att1")
	if string(data2) != "world" {
		t.Fatalf("expected upserted content, got %q", data2)
	}

	loaded, _ := s.LoadEntry(entryID)
	if len(loaded.Attachments) != 1 || loaded.Attachments[0].FileName != "a.txt" {
		t.Fatalf("expected attachment metadata on entry, got %+v", loaded.Attachments)
	}

	if err := s.DeleteAttachment("att1"); err != nil {
		t.Fatalf("delete attachment: %v", err)
	}
	if _, err := s.ReadAttachment("att1"); err == nil {
		t.Fatalf("expected error reading deleted attachment")
	}
}

func TestAttachmentCascadeDeleteWithEntry(t *testing.T) {
	s := newTestStore(t)
	entryID, _ := s.SaveEntry(0, &EntryFull{Type: "File", Title: "Files"})
	_ = s.WriteAttachment("att1", entryID, "a.txt", 5, []byte("hello"))

	if err := s.DeleteEntry(entryID); err != nil {
		t.Fatalf("delete entry: %v", err)
	}
	if _, err := s.ReadAttachment("att1"); err == nil {
		t.Fatalf("expected attachment to cascade-delete with its entry")
	}
}

func TestPinConfigCRUD(t *testing.T) {
	s := newTestStore(t)

	if s.PinConfigExists() {
		t.Fatalf("did not expect a pin config on a fresh store")
	}
	cfg, err := s.ReadPinConfig()
	if err != nil || cfg != nil {
		t.Fatalf("expected nil pin config, got %+v err=%v", cfg, err)
	}

	want := &PinConfig{Mode: "pin", PinKey: []byte{1, 2, 3}, PinTag: "tag", TotpSecret: "SECRET"}
	if err := s.WritePinConfig(want); err != nil {
		t.Fatalf("write pin config: %v", err)
	}
	if !s.PinConfigExists() {
		t.Fatalf("expected pin config to exist after write")
	}
	got, err := s.ReadPinConfig()
	if err != nil || got == nil {
		t.Fatalf("read pin config: %+v err=%v", got, err)
	}
	if got.Mode != "pin" || got.PinTag != "tag" || got.TotpSecret != "SECRET" {
		t.Fatalf("unexpected pin config: %+v", got)
	}

	// upsert should overwrite
	want.Mode = "totp"
	if err := s.WritePinConfig(want); err != nil {
		t.Fatalf("update pin config: %v", err)
	}
	got2, _ := s.ReadPinConfig()
	if got2.Mode != "totp" {
		t.Fatalf("expected updated mode, got %+v", got2)
	}
}

func TestRekey(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")
	s, err := Open(dbPath, "OldPassw0rd!")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := s.CreateFolder("Test"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if err := s.Rekey("NewPassw0rd!"); err != nil {
		t.Fatalf("rekey: %v", err)
	}
	s.Close()

	if err := VerifyKey(dbPath, "NewPassw0rd!"); err != nil {
		t.Fatalf("expected new key to work: %v", err)
	}
	if err := VerifyKey(dbPath, "OldPassw0rd!"); err == nil {
		t.Fatalf("expected old key to be rejected after rekey")
	}
}

func TestRekeyWithQuoteInPassword(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")
	s, err := Open(dbPath, "OldPassw0rd!")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := s.Rekey("o'brien's pass"); err != nil {
		t.Fatalf("rekey with quote: %v", err)
	}
	s.Close()

	if err := VerifyKey(dbPath, "o'brien's pass"); err != nil {
		t.Fatalf("expected quoted key to work: %v", err)
	}
}

func TestDBExistsAndRemoveDBFiles(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")
	if DBExists(dbPath) {
		t.Fatalf("did not expect db to exist yet")
	}
	s, err := Open(dbPath, "MyStr0ng!Pass")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s.Close()
	if !DBExists(dbPath) {
		t.Fatalf("expected db to exist after Open")
	}
	RemoveDBFiles(dbPath)
	if DBExists(dbPath) {
		t.Fatalf("expected db file to be removed")
	}
}

func TestVerifyKeyWrongPassword(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "passbook.db")
	s, err := Open(dbPath, "CorrectPass1!")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s.Close()

	if err := VerifyKey(dbPath, "WrongPass1!"); err == nil {
		t.Fatalf("expected wrong password to fail verification")
	}
}

func TestCloseNilDB(t *testing.T) {
	var s Store
	if err := s.Close(); err != nil {
		t.Fatalf("expected Close on zero-value store to be a no-op, got %v", err)
	}
}
