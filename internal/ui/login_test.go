package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"passbook/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewLoginModel(t *testing.T) {
	m := newLoginModel(true)
	if !m.freshInstall {
		t.Fatalf("expected freshInstall true")
	}
	if m.focusField != 0 {
		t.Fatalf("expected initial focus on password field")
	}
}

func newLoginTestModelForFlow(t *testing.T, dbPath string) *Model {
	t.Helper()
	return &Model{
		dbPath: dbPath,
		login:  newLoginModel(!store.DBExists(dbPath)),
		width:  80, height: 24,
	}
}

func TestUpdateLoginKeyEscQuits(t *testing.T) {
	m := newLoginTestModelForFlow(t, filepath.Join(t.TempDir(), "passbook.db"))
	_, cmd := m.updateLoginKey(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected esc to quit")
	}
}

func TestUpdateLoginKeyNavigation(t *testing.T) {
	m := newLoginTestModelForFlow(t, filepath.Join(t.TempDir(), "passbook.db"))

	nm, _ := m.updateLoginKey(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.login.focusField != 1 {
		t.Fatalf("expected focus to move to buttons, got %d", m.login.focusField)
	}
	nm, _ = m.updateLoginKey(tea.KeyMsg{Type: tea.KeyRight})
	*m = nm
	if m.login.buttonFocus != 1 {
		t.Fatalf("expected buttonFocus 1, got %d", m.login.buttonFocus)
	}
	nm, _ = m.updateLoginKey(tea.KeyMsg{Type: tea.KeyLeft})
	*m = nm
	if m.login.buttonFocus != 0 {
		t.Fatalf("expected buttonFocus back to 0, got %d", m.login.buttonFocus)
	}
	nm, _ = m.updateLoginKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	*m = nm
	if m.login.focusField != 0 {
		t.Fatalf("expected shift+tab to return focus to password field, got %d", m.login.focusField)
	}
}

func TestUpdateLoginKeyTypingUpdatesStrength(t *testing.T) {
	m := newLoginTestModelForFlow(t, filepath.Join(t.TempDir(), "passbook.db"))
	nm, _ := m.updateLoginKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S3cur3!Pass")})
	*m = nm
	if m.login.strength == "" {
		t.Fatalf("expected strength indicator to update while typing")
	}
}

func TestGoToMainEmptyPasswordNoOp(t *testing.T) {
	m := newLoginTestModelForFlow(t, filepath.Join(t.TempDir(), "passbook.db"))
	m.goToMain("")
	if m.store != nil {
		t.Fatalf("expected no store to be opened for an empty password")
	}
}

func TestGoToMainFreshInstallWeakPasswordRejected(t *testing.T) {
	m := newLoginTestModelForFlow(t, filepath.Join(t.TempDir(), "passbook.db"))
	m.goToMain("weak")
	if m.login.errorMsg == "" {
		t.Fatalf("expected an error message for a weak password")
	}
	if m.store != nil {
		t.Fatalf("expected store to be cleaned up after rejecting a weak password")
	}
}

func TestGoToMainFreshInstallStrongPasswordGoesToPinSetup(t *testing.T) {
	m := newLoginTestModelForFlow(t, filepath.Join(t.TempDir(), "passbook.db"))
	m.goToMain("Sup3r$ecureLongPassphrase!!")
	if m.screen != screenPinSetup {
		t.Fatalf("expected screenPinSetup, got %v (err=%q)", m.screen, m.login.errorMsg)
	}
	if m.store == nil {
		t.Fatalf("expected store to be opened")
	}
	m.store.Close()
}

func TestGoToMainExistingVaultWithPinGoesToVerify(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "passbook.db")
	s, err := store.Open(dbPath, "Sup3r$ecureLongPassphrase!!")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := s.CreateFolder("x"); err != nil {
		t.Fatal(err)
	}
	if err := s.WritePinConfig(&store.PinConfig{Mode: "pin", PinKey: []byte("k"), PinTag: "tag"}); err != nil {
		t.Fatal(err)
	}
	s.Close()

	m := newLoginTestModelForFlow(t, dbPath)
	m.goToMain("Sup3r$ecureLongPassphrase!!")
	if m.screen != screenPinVerify {
		t.Fatalf("expected screenPinVerify, got %v", m.screen)
	}
	m.store.Close()
}

func TestGoToMainExistingVaultWrongPassword(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "passbook.db")
	s, err := store.Open(dbPath, "Correct$Passw0rd!")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	s.Close()

	m := newLoginTestModelForFlow(t, dbPath)
	m.goToMain("Wrong$Passw0rd!")
	if m.login.errorMsg != "Wrong password." {
		t.Fatalf("expected wrong password message, got %q", m.login.errorMsg)
	}
}

func TestLoginErrorMessage(t *testing.T) {
	if got := loginErrorMessage(nil, true); got != "Could not create vault." {
		t.Fatalf("unexpected fresh install message: %q", got)
	}
	if got := loginErrorMessage(nil, false); got != "Wrong password." {
		t.Fatalf("unexpected existing vault message: %q", got)
	}
}

func TestCloseAndCleanupStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "passbook.db")
	s, err := store.Open(dbPath, "Sup3r$ecureLongPassphrase!!")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	m := &Model{store: s, dbPath: dbPath}
	m.closeAndCleanupStore(true)
	if m.store != nil {
		t.Fatalf("expected store to be nil after cleanup")
	}
	if store.DBExists(dbPath) {
		t.Fatalf("expected db file to be removed")
	}
}

func TestViewLoginRendersTitle(t *testing.T) {
	m := Model{width: 80, height: 24, login: newLoginModel(true), freshInstall: true}
	got := m.viewLogin()
	if !strings.Contains(got, "PassBook Setup") {
		t.Fatalf("expected setup title for fresh install, got %q", got)
	}

	m2 := Model{width: 80, height: 24, login: newLoginModel(false), freshInstall: false}
	got2 := m2.viewLogin()
	if !strings.Contains(got2, "PassBook Login") {
		t.Fatalf("expected login title for existing vault, got %q", got2)
	}
}
