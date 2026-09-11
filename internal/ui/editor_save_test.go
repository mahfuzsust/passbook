package ui

import (
	"strings"
	"testing"
)

func newEditorSaveTestModel(t *testing.T, typ EntryType) *Model {
	t.Helper()
	s := newTestStore(t)
	m := &Model{store: s, width: 80, height: 24}
	m.main = newMainModel()
	_ = m.newEntry(typ)
	return m
}

func TestSaveEntryEmptyTitleNoOp(t *testing.T) {
	m := newEditorSaveTestModel(t, TypeNote)
	m.editor.title.SetValue("")
	m.saveEntry()
	if m.store.HasEntries() {
		t.Fatalf("did not expect an entry to be created with an empty title")
	}
}

func TestSaveEntryCreatesNewLoginEntry(t *testing.T) {
	m := newEditorSaveTestModel(t, TypeLogin)
	m.editor.title.SetValue("GitHub")
	m.editor.username.SetValue("alice")
	m.editor.password.SetValue("hunter2")

	m.saveEntry()
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to close after save")
	}
	if !m.store.HasEntries() {
		t.Fatalf("expected entry to be persisted")
	}
	if m.main.currentEnt == nil || m.main.currentEnt.Username != "alice" {
		t.Fatalf("expected the new entry to be loaded, got %+v", m.main.currentEnt)
	}
}

func TestSaveEntryRejectsDuplicateTitleInSameFolder(t *testing.T) {
	m := newEditorSaveTestModel(t, TypeNote)
	m.editor.title.SetValue("Dup")
	m.saveEntry()

	_ = m.newEntry(TypeNote)
	m.editor.title.SetValue("Dup")
	m.saveEntry()

	if m.overlay != overlayError {
		t.Fatalf("expected duplicate title to show an error overlay, got %v", m.overlay)
	}
}

func TestSaveEntryRejectsInvalidCardFields(t *testing.T) {
	m := newEditorSaveTestModel(t, TypeCard)
	m.editor.title.SetValue("Visa")
	m.editor.cardNumber.SetValue("not-digits")
	m.editor.expiry.SetValue("13/99")
	m.editor.cvv.SetValue("12")

	m.saveEntry()
	if m.store.HasEntries() {
		t.Fatalf("did not expect an entry to be saved with invalid card fields")
	}
}

func TestSaveEntryUpdatesExistingEntry(t *testing.T) {
	m := newEditorSaveTestModel(t, TypeNote)
	m.editor.title.SetValue("Original")
	m.saveEntry()
	entryID := m.main.currentEntryID

	_ = m.openEditor(m.main.currentEnt)
	m.editor.title.SetValue("Renamed")
	m.saveEntry()

	if m.main.currentEnt.Title != "Renamed" {
		t.Fatalf("expected entry to be renamed, got %q", m.main.currentEnt.Title)
	}
	if m.main.currentEntryID != entryID {
		t.Fatalf("expected the same entry id to persist across edits")
	}
}

func TestViewEditorRendersEachEntryType(t *testing.T) {
	for _, typ := range []EntryType{TypeLogin, TypeCard, TypeNote, TypeFile} {
		m := newEditorSaveTestModel(t, typ)
		got := m.viewEditor()
		if !strings.Contains(got, "Add Entry") {
			t.Fatalf("%v: expected editor title, got %q", typ, got)
		}
	}
}

func TestRenderLoginAndCardEditorFields(t *testing.T) {
	m := newEditorSaveTestModel(t, TypeLogin)
	m.editor.username.SetValue("bob")
	if got := m.renderLoginEditorFields(); !strings.Contains(got, "Username") {
		t.Fatalf("expected username label, got %q", got)
	}

	c := newEditorSaveTestModel(t, TypeCard)
	if got := c.renderCardEditorFields(); !strings.Contains(got, "Card Number") {
		t.Fatalf("expected card number label, got %q", got)
	}
}
