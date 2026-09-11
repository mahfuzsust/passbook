package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateCreateMenuNavigation(t *testing.T) {
	m := &Model{store: newTestStore(t)}
	m.createMenu = newCreateMenuModel()

	nm, _ := m.updateCreateMenu("down")
	*m = nm
	if m.createMenu.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.createMenu.cursor)
	}
	nm, _ = m.updateCreateMenu("up")
	*m = nm
	if m.createMenu.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.createMenu.cursor)
	}
	nm, _ = m.updateCreateMenu("up")
	*m = nm
	if m.createMenu.cursor != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", m.createMenu.cursor)
	}
}

func TestUpdateCreateMenuEsc(t *testing.T) {
	m := &Model{overlay: overlayCreateMenu}
	m.createMenu = newCreateMenuModel()
	nm, _ := m.updateCreateMenu("esc")
	if nm.overlay != overlayNone {
		t.Fatalf("expected esc to close overlay")
	}
}

func TestUpdateCreateMenuEnterSelectsCursor(t *testing.T) {
	m := &Model{store: newTestStore(t)}
	m.createMenu = newCreateMenuModel()
	m.createMenu.cursor = 2 // Note

	nm, _ := m.updateCreateMenu("enter")
	*m = nm
	if m.editor.entryType != TypeNote {
		t.Fatalf("expected note entry type, got %v", m.editor.entryType)
	}
}

func TestEditorFocusUpdateFocusedAllFields(t *testing.T) {
	e := &editorModel{
		title:      newTextInput("Title", false),
		username:   newTextInput("Username", false),
		password:   newTextInput("Password", true),
		link:       newTextInput("Link", false),
		totpSecret: newTextInput("TOTP", false),
		cardNumber: newTextInput("Card", false),
		expiry:     newTextInput("Expiry", false),
		cvv:        newTextInput("CVV", false),
	}

	cases := []struct {
		target editorFocusTarget
		get    func() string
	}{
		{eftTitle, func() string { return e.title.Value() }},
		{eftUsername, func() string { return e.username.Value() }},
		{eftPassword, func() string { return e.password.Value() }},
		{eftLink, func() string { return e.link.Value() }},
		{eftTotpSecret, func() string { return e.totpSecret.Value() }},
		{eftCardNumber, func() string { return e.cardNumber.Value() }},
		{eftExpiry, func() string { return e.expiry.Value() }},
		{eftCVV, func() string { return e.cvv.Value() }},
	}
	for _, c := range cases {
		e.focusTarget = c.target
		e.blurAll()
		switch c.target {
		case eftTitle:
			e.title.Focus()
		case eftUsername:
			e.username.Focus()
		case eftPassword:
			e.password.Focus()
		case eftLink:
			e.link.Focus()
		case eftTotpSecret:
			e.totpSecret.Focus()
		case eftCardNumber:
			e.cardNumber.Focus()
		case eftExpiry:
			e.expiry.Focus()
		case eftCVV:
			e.cvv.Focus()
		}
		e.updateFocused(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
		if c.get() != "x" {
			t.Fatalf("target %v: expected 'x' typed, got %q", c.target, c.get())
		}
	}
}

func TestShowQuickCopyCardAndNote(t *testing.T) {
	m := &Model{main: mainModel{
		currentEnt:     &Entry{Type: string(TypeCard), CardNumber: "4111", CVV: "123"},
		currentEntryID: 1,
	}}
	m.showQuickCopy()
	if m.overlay != overlayQuickCopy || len(m.quickCopy.items) != 2 {
		t.Fatalf("expected 2 quick copy items for card, got %d", len(m.quickCopy.items))
	}

	m2 := &Model{main: mainModel{
		currentEnt:     &Entry{Type: string(TypeNote), CustomText: "hello"},
		currentEntryID: 1,
	}}
	m2.showQuickCopy()
	if m2.overlay != overlayQuickCopy || len(m2.quickCopy.items) != 1 {
		t.Fatalf("expected 1 quick copy item for note, got %d", len(m2.quickCopy.items))
	}
}
