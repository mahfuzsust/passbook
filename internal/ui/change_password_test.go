package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"passbook/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

func newChangePwdTestModel(t *testing.T, currentPwd string) *Model {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "passbook.db")
	s, err := store.Open(dbPath, currentPwd)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	m := &Model{store: s, dbPath: dbPath, width: 80, height: 24, overlay: overlayChangePwd}
	m.changePwd = newChangePwdModel()
	return m
}

func TestNewChangePwdModel(t *testing.T) {
	c := newChangePwdModel()
	if c.focus != cpfCurrent {
		t.Fatalf("expected initial focus on current password field")
	}
	if !c.current.Focused() {
		t.Fatalf("expected current field to be focused")
	}
}

func TestChangePwdFocusCycle(t *testing.T) {
	c := newChangePwdModel()
	c.focusNext()
	if c.focus != cpfNew {
		t.Fatalf("expected cpfNew, got %v", c.focus)
	}
	c.focusNext()
	if c.focus != cpfConfirm {
		t.Fatalf("expected cpfConfirm, got %v", c.focus)
	}
	c.focusNext()
	if c.focus != cpfButtons {
		t.Fatalf("expected cpfButtons, got %v", c.focus)
	}
	c.focusNext()
	if c.focus != cpfCurrent {
		t.Fatalf("expected wraparound to cpfCurrent, got %v", c.focus)
	}
	c.focusPrev()
	if c.focus != cpfButtons {
		t.Fatalf("expected wraparound back to cpfButtons, got %v", c.focus)
	}
}

func TestUpdateChangePwdKeyNavigationAndTyping(t *testing.T) {
	m := newChangePwdTestModel(t, "Original$Pass1!")

	nm, _ := m.updateChangePwdKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	*m = nm
	if m.changePwd.current.Value() != "x" {
		t.Fatalf("expected typed char in current field, got %q", m.changePwd.current.Value())
	}

	nm, _ = m.updateChangePwdKey(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.changePwd.focus != cpfNew {
		t.Fatalf("expected focus to move to new password field")
	}
	nm, _ = m.updateChangePwdKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	*m = nm
	if m.changePwd.strength == "" {
		t.Fatalf("expected strength indicator to update for new password field")
	}

	nm, _ = m.updateChangePwdKey(tea.KeyMsg{Type: tea.KeyEsc})
	*m = nm
	if m.overlay != overlayNone {
		t.Fatalf("expected esc to close overlay")
	}
}

func TestUpdateChangePwdKeyButtonNavigation(t *testing.T) {
	m := newChangePwdTestModel(t, "Original$Pass1!")
	m.changePwd.focus = cpfButtons

	nm, _ := m.updateChangePwdKey(tea.KeyMsg{Type: tea.KeyRight})
	*m = nm
	if m.changePwd.btnFocus != 1 {
		t.Fatalf("expected btnFocus 1, got %d", m.changePwd.btnFocus)
	}
	nm, _ = m.updateChangePwdKey(tea.KeyMsg{Type: tea.KeyLeft})
	*m = nm
	if m.changePwd.btnFocus != 0 {
		t.Fatalf("expected btnFocus 0, got %d", m.changePwd.btnFocus)
	}
}

func TestDoChangePasswordValidation(t *testing.T) {
	m := newChangePwdTestModel(t, "Original$Pass1!")

	m.doChangePassword()
	if !strings.Contains(m.changePwd.status, "required") {
		t.Fatalf("expected required-fields message, got %q", m.changePwd.status)
	}

	m.changePwd.current.SetValue("Original$Pass1!")
	m.changePwd.newPwd.SetValue("New$Pass1!")
	m.changePwd.confirm.SetValue("Different$Pass1!")
	m.doChangePassword()
	if !strings.Contains(m.changePwd.status, "not match") {
		t.Fatalf("expected mismatch message, got %q", m.changePwd.status)
	}

	m.changePwd.current.SetValue("Wrong$Pass1!")
	m.changePwd.newPwd.SetValue("New$Pass1!")
	m.changePwd.confirm.SetValue("New$Pass1!")
	m.doChangePassword()
	if !strings.Contains(m.changePwd.status, "incorrect") {
		t.Fatalf("expected incorrect current password message, got %q", m.changePwd.status)
	}

	m.changePwd.current.SetValue("Original$Pass1!")
	m.changePwd.newPwd.SetValue("Original$Pass1!")
	m.changePwd.confirm.SetValue("Original$Pass1!")
	m.doChangePassword()
	if !strings.Contains(m.changePwd.status, "different") {
		t.Fatalf("expected same-password message, got %q", m.changePwd.status)
	}

	m.changePwd.current.SetValue("Original$Pass1!")
	m.changePwd.newPwd.SetValue("weak")
	m.changePwd.confirm.SetValue("weak")
	m.doChangePassword()
	if !strings.Contains(m.changePwd.status, "weak") {
		t.Fatalf("expected weak password message, got %q", m.changePwd.status)
	}
}

func TestDoChangePasswordSuccess(t *testing.T) {
	m := newChangePwdTestModel(t, "Original$Pass1!")
	m.changePwd.current.SetValue("Original$Pass1!")
	m.changePwd.newPwd.SetValue("Br4nd$NewPassphrase!")
	m.changePwd.confirm.SetValue("Br4nd$NewPassphrase!")

	m.doChangePassword()
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to close on success, status=%q", m.changePwd.status)
	}
}

func TestViewChangePwdRenders(t *testing.T) {
	m := newChangePwdTestModel(t, "Original$Pass1!")
	got := m.viewChangePwd()
	if !strings.Contains(got, "Change Master Password") {
		t.Fatalf("expected title in view, got %q", got)
	}
}
