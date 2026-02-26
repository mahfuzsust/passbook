package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write1PasswordJSON(t *testing.T, items []onePasswordItem) string {
	t.Helper()
	export := onePasswordExport{
		Accounts: []onePasswordAccount{
			{Vaults: []onePasswordVault{
				{Items: items},
			}},
		},
	}
	data, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(t.TempDir(), "export.1pux")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestImport1PasswordLogin(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := write1PasswordJSON(t, []onePasswordItem{
		{
			Title:    "GitHub",
			Category: "001",
			URLs:     []onePasswordURL{{URL: "https://github.com"}},
			Fields: []onePasswordField{
				{Designation: "username", Value: "user@example.com"},
				{Designation: "password", Value: "secret123"},
			},
			Sections: []onePasswordSection{
				{Fields: []onePasswordField{
					{Type: "OTP", Value: "JBSWY3DPEHPK3PXP"},
				}},
			},
			Notes: "My GitHub account",
		},
	})

	if err := Import1Password(jsonPath, password, cfg); err != nil {
		t.Fatalf("Import1Password: %v", err)
	}

	entryPath := filepath.Join(dir, "GitHub.pb")
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

func TestImport1PasswordCard(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := write1PasswordJSON(t, []onePasswordItem{
		{
			Title:    "Visa",
			Category: "002",
			Fields: []onePasswordField{
				{Name: "ccnum", Value: "4111111111111111"},
				{Name: "cvv", Value: "123"},
				{Name: "expiry", Value: "12/2028"},
				{Name: "cardholder", Value: "John Doe"},
			},
		},
	})

	if err := Import1Password(jsonPath, password, cfg); err != nil {
		t.Fatalf("Import1Password: %v", err)
	}

	entryPath := filepath.Join(dir, "Visa.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if entry.Type != "Card" {
		t.Fatalf("expected Card type, got %s", entry.Type)
	}
	if entry.CardNumber != "4111111111111111" {
		t.Fatalf("expected card number, got %s", entry.CardNumber)
	}
	if entry.Cvv != "123" {
		t.Fatalf("expected CVV, got %s", entry.Cvv)
	}
	if entry.Expiry != "12/2028" {
		t.Fatalf("expected expiry, got %s", entry.Expiry)
	}
	if !strings.Contains(entry.CustomText, "Cardholder: John Doe") {
		t.Fatalf("expected cardholder in notes, got %s", entry.CustomText)
	}
}

func TestImport1PasswordSecureNote(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := write1PasswordJSON(t, []onePasswordItem{
		{
			Title:    "My Note",
			Category: "003",
			Notes:    "Secret note content.",
		},
	})

	if err := Import1Password(jsonPath, password, cfg); err != nil {
		t.Fatalf("Import1Password: %v", err)
	}

	entryPath := filepath.Join(dir, "My Note.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if entry.Type != "Note" {
		t.Fatalf("expected Note type, got %s", entry.Type)
	}
	if entry.CustomText != "Secret note content." {
		t.Fatalf("expected notes, got %s", entry.CustomText)
	}
}

func TestImport1PasswordExtraFields(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := write1PasswordJSON(t, []onePasswordItem{
		{
			Title:    "WithExtra",
			Category: "001",
			Fields: []onePasswordField{
				{Designation: "username", Value: "u"},
				{Designation: "password", Value: "p"},
			},
			Sections: []onePasswordSection{
				{Fields: []onePasswordField{
					{Name: "API Key", Value: "abc123"},
				}},
			},
		},
	})

	if err := Import1Password(jsonPath, password, cfg); err != nil {
		t.Fatalf("Import1Password: %v", err)
	}

	entryPath := filepath.Join(dir, "WithExtra.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if !strings.Contains(entry.CustomText, "API Key: abc123") {
		t.Fatalf("expected extra field in notes, got %q", entry.CustomText)
	}
}

func TestImport1PasswordUnknownCategoryAsNote(t *testing.T) {
	password := "testpass"
	dir, cfg := setupTestVault(t, password)

	jsonPath := write1PasswordJSON(t, []onePasswordItem{
		{
			Title:    "Identity",
			Category: "099",
			Notes:    "Some identity info",
		},
	})

	if err := Import1Password(jsonPath, password, cfg); err != nil {
		t.Fatalf("Import1Password: %v", err)
	}

	entryPath := filepath.Join(dir, "Identity.pb")
	key := deriveTestKey(t, dir, password)

	entry := decryptEntry(t, entryPath, key)
	if entry.Type != "Note" {
		t.Fatalf("expected Note type for unknown category, got %s", entry.Type)
	}
}
