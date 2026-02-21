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

	entryPath := filepath.Join(dir, "logins", "GitHub.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
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

	entryPath := filepath.Join(dir, "notes", "My Note.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
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

	key := deriveTestKey(t, dir, password)

	entryA := decryptEntry(t, filepath.Join(dir, "logins", "Site A.pb"), key)
	if entryA.Username != "u1" {
		t.Fatalf("expected u1, got %s", entryA.Username)
	}

	entryB := decryptEntry(t, filepath.Join(dir, "logins", "Site B.pb"), key)
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

	if _, err := os.Stat(filepath.Join(dir, "logins", "Dup.pb")); err != nil {
		t.Fatalf("expected Dup.pb: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "logins", "Dup_1.pb")); err != nil {
		t.Fatalf("expected Dup_1.pb: %v", err)
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

	// When name is empty, it should use the URL as the title.
	// URL contains ":" and "/" which get sanitized.
	files, err := os.ReadDir(filepath.Join(dir, "logins"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}
