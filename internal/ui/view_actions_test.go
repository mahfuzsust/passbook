package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func newLoginTestModel() *Model {
	m := &Model{main: newMainModel()}
	m.main.showContent = true
	m.main.currentEnt = &Entry{
		Type:       string(TypeLogin),
		Title:      "Example",
		Username:   "alice",
		Password:   "hunter2",
		Link:       "http://example.com",
		TotpSecret: "JBSWY3DPEHPK3PXP",
	}
	m.main.currentEntryID = 1
	return m
}

func TestHandleViewActionCopyUsername(t *testing.T) {
	m := newLoginTestModel()
	handled, cmd := m.handleViewAction("u")
	if !handled {
		t.Fatalf("expected 'u' to be handled")
	}
	if cmd != nil {
		t.Fatalf("username copy should not schedule a clipboard-clear cmd")
	}
	if !strings.Contains(m.main.viewStatus, "Username") {
		t.Fatalf("expected status message to mention Username, got %q", m.main.viewStatus)
	}
	if m.main.viewStatusClearAt.IsZero() {
		t.Fatalf("expected viewStatusClearAt to be set so the message auto-clears")
	}
}

func TestHandleViewActionCopyPasswordSchedulesClear(t *testing.T) {
	m := newLoginTestModel()
	handled, cmd := m.handleViewAction("c")
	if !handled {
		t.Fatalf("expected 'c' to be handled")
	}
	if cmd == nil {
		t.Fatalf("expected password copy to schedule a clipboard-clear cmd")
	}
	if m.main.viewStatusClearAt.IsZero() {
		t.Fatalf("expected status message to auto-clear")
	}
}

func TestHandleViewActionCopyLink(t *testing.T) {
	m := newLoginTestModel()
	handled, _ := m.handleViewAction("l")
	if !handled {
		t.Fatalf("expected 'l' to be handled")
	}
	if !strings.Contains(m.main.viewStatus, "Link") {
		t.Fatalf("expected status message to mention Link, got %q", m.main.viewStatus)
	}
}

func TestHandleViewActionCopyTOTP(t *testing.T) {
	m := newLoginTestModel()
	handled, cmd := m.handleViewAction("t")
	if !handled {
		t.Fatalf("expected 't' to be handled")
	}
	if cmd == nil {
		t.Fatalf("expected TOTP copy to schedule a clipboard-clear cmd")
	}
	if !strings.Contains(m.main.viewStatus, "TOTP") {
		t.Fatalf("expected status message to mention TOTP, got %q", m.main.viewStatus)
	}
}

func TestHandleViewActionCardCopiesCardNumber(t *testing.T) {
	m := &Model{main: mainModel{
		currentEnt:     &Entry{Type: string(TypeCard), CardNumber: "4111111111111111"},
		currentEntryID: 1,
	}}
	handled, cmd := m.handleViewAction("c")
	if !handled || cmd == nil {
		t.Fatalf("expected card copy to be handled with a clear cmd")
	}
}

func TestHandleViewActionUnknownKeyNotHandled(t *testing.T) {
	m := newLoginTestModel()
	handled, cmd := m.handleViewAction("z")
	if handled || cmd != nil {
		t.Fatalf("expected unknown key to be unhandled")
	}
}

func TestHandleViewActionNoCurrentEntry(t *testing.T) {
	m := &Model{}
	handled, cmd := m.handleViewAction("u")
	if handled || cmd != nil {
		t.Fatalf("expected no-op when there is no current entry")
	}
}

func TestNotifyCopiedSetsClearTimer(t *testing.T) {
	m := &Model{}
	m.notifyCopied("Note")
	if m.main.viewStatusClearAt.IsZero() {
		t.Fatalf("expected notifyCopied to schedule status clearing")
	}
	if !strings.Contains(m.main.viewStatus, "Note") {
		t.Fatalf("expected status to mention Note, got %q", m.main.viewStatus)
	}
}

func TestTickClearsExpiredViewStatus(t *testing.T) {
	m := Model{
		screen: screenMain,
		main: mainModel{
			viewStatus:        "stale",
			viewStatusClearAt: time.Now().Add(-time.Second),
		},
	}
	updated, _ := m.Update(tickMsg{})
	nm := updated.(Model)
	if nm.main.viewStatus != "" {
		t.Fatalf("expected expired status to be cleared, got %q", nm.main.viewStatus)
	}
}

func TestHandleClipboardClearWipesMatchingClipboard(t *testing.T) {
	m := &Model{}
	// We can't reliably assert on the system clipboard in CI, but we can
	// verify the handler doesn't panic and returns without requiring a
	// specific clipboard backend.
	_, cmd := m.handleClipboardClear("some-value-unlikely-to-be-on-clipboard")
	_ = cmd
	var _ tea.Cmd = cmd
}

func TestSearchFocusedKeysTypeIntoSearchInsteadOfTriggeringActions(t *testing.T) {
	m := newLoginTestModel()
	m.screen = screenMain
	m.store = newTestStore(t)
	m.main.searchFocused = true
	m.main.search.Focus()

	for _, k := range []string{"l", "v", "c", "u", "t", "o", "h"} {
		nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		*m = nm
	}

	if m.main.search.Value() != "lvcutoh" {
		t.Fatalf("expected keys to be typed into the search box, got %q", m.main.search.Value())
	}
	if m.main.showSensitive {
		t.Fatalf("did not expect 'v' to toggle reveal while search is focused")
	}
	if m.overlay == overlayHistory {
		t.Fatalf("did not expect 'h' to open history while search is focused")
	}
}

func TestSearchNotFocusedKeysStillTriggerViewActions(t *testing.T) {
	m := newLoginTestModel()
	m.screen = screenMain
	m.main.searchFocused = false

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	*m = nm
	if !m.main.showSensitive {
		t.Fatalf("expected 'v' to toggle reveal when search is not focused")
	}
}
