package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeLastPassCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "export.csv")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestImportLastPassLogin(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	csv := `url,username,password,totp,extra,name,grouping,fav
https://github.com,user@example.com,secret123,JBSWY3DPEHPK3PXP,My GitHub account,GitHub,,0
`
	csvPath := writeLastPassCSV(t, csv)

	if err := ImportLastPass(csvPath, password, cfg); err != nil {
		t.Fatalf("ImportLastPass: %v", err)
	}

	s := openTestStore(t, dir, password)
	entry := loadEntryFromStore(t, s, "GitHub")
	if entry.Type != "Login" {
		t.Fatalf("expected Login type, got %s", entry.Type)
	}
	if entry.Username != "user@example.com" {
		t.Fatalf("expected username, got %s", entry.Username)
	}
	if entry.Password != "secret123" {
		t.Fatalf("expected password, got %s", entry.Password)
	}
	if entry.TotpSecret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("expected TOTP, got %s", entry.TotpSecret)
	}
	if entry.Link != "https://github.com" {
		t.Fatalf("expected link, got %s", entry.Link)
	}
	if entry.CustomText != "My GitHub account" {
		t.Fatalf("expected notes, got %s", entry.CustomText)
	}
}

func TestImportLastPassSecureNote(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	csv := `url,username,password,totp,extra,name,grouping,fav
http://sn,,,,"Secret note content",My Note,,0
`
	csvPath := writeLastPassCSV(t, csv)

	if err := ImportLastPass(csvPath, password, cfg); err != nil {
		t.Fatalf("ImportLastPass: %v", err)
	}

	s := openTestStore(t, dir, password)
	entry := loadEntryFromStore(t, s, "My Note")
	if entry.Type != "Note" {
		t.Fatalf("expected Note type, got %s", entry.Type)
	}
	if entry.CustomText != "Secret note content" {
		t.Fatalf("expected notes, got %s", entry.CustomText)
	}
}

func TestImportLastPassMultipleEntries(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	csv := `url,username,password,totp,extra,name,grouping,fav
https://a.com,u1,p1,,,Site A,,0
https://b.com,u2,p2,,,Site B,,0
`
	csvPath := writeLastPassCSV(t, csv)

	if err := ImportLastPass(csvPath, password, cfg); err != nil {
		t.Fatalf("ImportLastPass: %v", err)
	}

	s := openTestStore(t, dir, password)
	entryA := loadEntryFromStore(t, s, "Site A")
	if entryA.Username != "u1" {
		t.Fatalf("expected u1, got %s", entryA.Username)
	}

	entryB := loadEntryFromStore(t, s, "Site B")
	if entryB.Username != "u2" {
		t.Fatalf("expected u2, got %s", entryB.Username)
	}
}

func TestImportLastPassDuplicateTitles(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	csv := `url,username,password,totp,extra,name,grouping,fav
https://a.com,u1,p1,,,Dup,,0
https://b.com,u2,p2,,,Dup,,0
`
	csvPath := writeLastPassCSV(t, csv)

	if err := ImportLastPass(csvPath, password, cfg); err != nil {
		t.Fatalf("ImportLastPass: %v", err)
	}

	s := openTestStore(t, dir, password)
	entries, err := s.ListAllEntries()
	if err != nil {
		t.Fatalf("ListAllEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestImportLastPassEmptyCSV(t *testing.T) {
	password := "testpass"
	_, cfg := setupTestVault(t, password)

	csv := `url,username,password,totp,extra,name,grouping,fav
`
	csvPath := writeLastPassCSV(t, csv)

	if err := ImportLastPass(csvPath, password, cfg); err != nil {
		t.Fatalf("ImportLastPass: %v", err)
	}
}

func TestImportLastPassFallbackTitleFromURL(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	csv := `url,username,password,totp,extra,name,grouping,fav
https://example.com,user,pass,,,,, 0
`
	csvPath := writeLastPassCSV(t, csv)

	if err := ImportLastPass(csvPath, password, cfg); err != nil {
		t.Fatalf("ImportLastPass: %v", err)
	}

	s := openTestStore(t, dir, password)
	entries, err := s.ListAllEntries()
	if err != nil {
		t.Fatalf("ListAllEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}
