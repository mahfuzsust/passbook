package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"passbook/internal/config"
	"passbook/internal/crypto"
	"passbook/internal/pb"

	"google.golang.org/protobuf/proto"
)

func setupTestVault(t *testing.T, password string) (string, config.AppConfig) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.AppConfig{DataDir: dir}

	vp, err := crypto.DefaultVaultParams()
	if err != nil {
		t.Fatalf("DefaultVaultParams: %v", err)
	}
	masterKey, _, err := crypto.DeriveKeys(password, vp)
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	if err := crypto.SaveVaultParams(dir, vp); err != nil {
		t.Fatalf("SaveVaultParams: %v", err)
	}
	if _, err := crypto.EnsureSecret(dir, masterKey, vp); err != nil {
		t.Fatalf("EnsureSecret: %v", err)
	}
	crypto.WipeBytes(masterKey)
	return dir, cfg
}

func writeBitwardenJSON(t *testing.T, items []bitwardenItem) string {
	t.Helper()
	export := bitwardenExport{Items: items}
	data, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func decryptEntry(t *testing.T, path string, key []byte) *pb.Entry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	plain, err := crypto.Decrypt(key, data)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	entry := &pb.Entry{}
	if err := proto.Unmarshal(plain, entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return entry
}

func deriveTestKey(t *testing.T, dir, password string) []byte {
	t.Helper()
	vp, err := crypto.LoadVaultParams(dir)
	if err != nil || vp == nil {
		t.Fatalf("LoadVaultParams: %v", err)
	}
	_, vaultKey, err := crypto.DeriveKeys(password, *vp)
	if err != nil {
		t.Fatalf("DeriveKeys: %v", err)
	}
	return vaultKey
}

func TestImportBitwardenLogin(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := writeBitwardenJSON(t, []bitwardenItem{
		{
			Type: 1,
			Name: "GitHub",
			Login: &bitwardenLogin{
				Username: "user@example.com",
				Password: "secret123",
				Totp:     "JBSWY3DPEHPK3PXP",
				URIs:     []bitwardenURI{{URI: "https://github.com"}},
			},
			Notes: "My GitHub account",
		},
	})

	if err := ImportBitwarden(jsonPath, password, cfg); err != nil {
		t.Fatalf("ImportBitwarden: %v", err)
	}

	// Verify the entry was created.
	entryPath := filepath.Join(dir, "logins", "GitHub.pb")
	if _, err := os.Stat(entryPath); err != nil {
		t.Fatalf("expected entry file: %v", err)
	}

	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if entry.Type != "Login" {
		t.Fatalf("expected Login type, got %s", entry.Type)
	}
	if entry.Title != "GitHub" {
		t.Fatalf("expected title GitHub, got %s", entry.Title)
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

func TestImportBitwardenPasswordHistory(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := writeBitwardenJSON(t, []bitwardenItem{
		{
			Type: 1,
			Name: "WithHistory",
			Login: &bitwardenLogin{
				Username: "user",
				Password: "current",
			},
			PasswordHistory: []bitwardenPasswordHistory{
				{LastUsedDate: "2025-01-15T10:00:00.000Z", Password: "old1"},
				{LastUsedDate: "2025-06-20T12:00:00.000Z", Password: "old2"},
			},
		},
	})

	if err := ImportBitwarden(jsonPath, password, cfg); err != nil {
		t.Fatalf("ImportBitwarden: %v", err)
	}

	entryPath := filepath.Join(dir, "logins", "WithHistory.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if len(entry.History) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(entry.History))
	}
	if entry.History[0].Password != "old1" {
		t.Fatalf("expected first history password 'old1', got %s", entry.History[0].Password)
	}
	if entry.History[0].Date != "2025-01-15T10:00:00.000Z" {
		t.Fatalf("expected first history date, got %s", entry.History[0].Date)
	}
	if entry.History[1].Password != "old2" {
		t.Fatalf("expected second history password 'old2', got %s", entry.History[1].Password)
	}
}

func TestImportBitwardenCard(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := writeBitwardenJSON(t, []bitwardenItem{
		{
			Type: 3,
			Name: "Visa",
			Card: &bitwardenCard{
				CardholderName: "John Doe",
				Number:         "4111111111111111",
				ExpMonth:       "12",
				ExpYear:        "2028",
				Code:           "123",
			},
		},
	})

	if err := ImportBitwarden(jsonPath, password, cfg); err != nil {
		t.Fatalf("ImportBitwarden: %v", err)
	}

	entryPath := filepath.Join(dir, "cards", "Visa.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if entry.Type != "Card" {
		t.Fatalf("expected Card type, got %s", entry.Type)
	}
	if entry.CardNumber != "4111111111111111" {
		t.Fatalf("expected card number, got %s", entry.CardNumber)
	}
	if entry.Expiry != "12/2028" {
		t.Fatalf("expected expiry 12/2028, got %s", entry.Expiry)
	}
	if entry.Cvv != "123" {
		t.Fatalf("expected CVV, got %s", entry.Cvv)
	}
}

func TestImportBitwardenNote(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := writeBitwardenJSON(t, []bitwardenItem{
		{
			Type:  2,
			Name:  "Secret Note",
			Notes: "This is my secret note.",
		},
	})

	if err := ImportBitwarden(jsonPath, password, cfg); err != nil {
		t.Fatalf("ImportBitwarden: %v", err)
	}

	entryPath := filepath.Join(dir, "notes", "Secret Note.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if entry.Type != "Note" {
		t.Fatalf("expected Note type, got %s", entry.Type)
	}
	if entry.CustomText != "This is my secret note." {
		t.Fatalf("expected notes, got %s", entry.CustomText)
	}
}

func TestImportBitwardenDuplicateTitles(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := writeBitwardenJSON(t, []bitwardenItem{
		{Type: 1, Name: "Dup", Login: &bitwardenLogin{Username: "a"}},
		{Type: 1, Name: "Dup", Login: &bitwardenLogin{Username: "b"}},
	})

	if err := ImportBitwarden(jsonPath, password, cfg); err != nil {
		t.Fatalf("ImportBitwarden: %v", err)
	}

	// Both should exist — one as Dup.pb, one as Dup_1.pb.
	if _, err := os.Stat(filepath.Join(dir, "logins", "Dup.pb")); err != nil {
		t.Fatalf("expected Dup.pb: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "logins", "Dup_1.pb")); err != nil {
		t.Fatalf("expected Dup_1.pb: %v", err)
	}
}

func TestImportBitwardenSkipsUnknownType(t *testing.T) {
	password := "testpass"
	_, cfg := setupTestVault(t, password)

	jsonPath := writeBitwardenJSON(t, []bitwardenItem{
		{Type: 99, Name: "Unknown"},
	})

	if err := ImportBitwarden(jsonPath, password, cfg); err != nil {
		t.Fatalf("ImportBitwarden: %v", err)
	}
}

func TestImportBitwardenCustomFields(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := writeBitwardenJSON(t, []bitwardenItem{
		{
			Type:  1,
			Name:  "WithFields",
			Login: &bitwardenLogin{Username: "u"},
			Fields: []bitwardenField{
				{Name: "API Key", Value: "abc123", Type: 0},
			},
		},
	})

	if err := ImportBitwarden(jsonPath, password, cfg); err != nil {
		t.Fatalf("ImportBitwarden: %v", err)
	}

	entryPath := filepath.Join(dir, "logins", "WithFields.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if entry.CustomText == "" || !contains(entry.CustomText, "API Key: abc123") {
		t.Fatalf("expected custom fields in notes, got %q", entry.CustomText)
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"normal", "normal"},
		{"has/slash", "has-slash"},
		{"has:colon", "hascolon"},
		{".", "entry"},
		{"..", "entry"},
		{"  spaces  ", "spaces"},
	}
	for _, tt := range tests {
		got := sanitizeTitle(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeTitle(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
