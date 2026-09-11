package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewEditorModelSelectsEntrysExistingFolder(t *testing.T) {
	s := newTestStore(t)
	folderID, err := s.CreateFolder("Work")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	entryID, err := s.SaveEntry(folderID, &Entry{Type: string(TypeNote), Title: "Note"})
	if err != nil {
		t.Fatalf("save entry: %v", err)
	}
	ent, err := s.LoadEntry(entryID)
	if err != nil {
		t.Fatalf("load entry: %v", err)
	}

	e := newEditorModel(ent, entryID, folderID, s)
	if got := e.folderOptions[e.folderIdx]; got != "Work" {
		t.Fatalf("expected folder dropdown to default to %q, got %q", "Work", got)
	}
}

func TestNewEditorModelDefaultsToRootFolder(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateFolder("Work"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	ent := NewEntry(TypeNote)

	e := newEditorModel(ent, 0, 0, s)
	if e.folderIdx != 0 || e.folderOptions[0] != "— (root)" {
		t.Fatalf("expected root to be selected by default, got idx=%d options=%v", e.folderIdx, e.folderOptions)
	}
}

func TestOpenEditorUsesEntrysOwnFolderNotBrowsedFolder(t *testing.T) {
	s := newTestStore(t)
	entryFolderID, err := s.CreateFolder("Entry Folder")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	browsedFolderID, err := s.CreateFolder("Other Folder")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	entryID, err := s.SaveEntry(entryFolderID, &Entry{Type: string(TypeNote), Title: "Note"})
	if err != nil {
		t.Fatalf("save entry: %v", err)
	}
	ent, err := s.LoadEntry(entryID)
	if err != nil {
		t.Fatalf("load entry: %v", err)
	}

	m := &Model{store: s}
	// simulate the tree currently browsing a different folder while an
	// entry from another folder is open for editing (loadEntry resets
	// currentFolderID to 0, but the entry itself still belongs elsewhere).
	m.main.currentFolderID = browsedFolderID
	m.main.currentEntryID = entryID
	m.main.currentEnt = ent

	_ = m.openEditor(ent)

	if got := m.editor.folderOptions[m.editor.folderIdx]; got != "Entry Folder" {
		t.Fatalf("expected editor to default to the entry's own folder %q, got %q", "Entry Folder", got)
	}
}

func TestOpenEditorForNewEntryUsesBrowsedFolder(t *testing.T) {
	s := newTestStore(t)
	folderID, err := s.CreateFolder("Work")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}

	m := &Model{store: s}
	m.main.currentFolderID = folderID
	m.main.currentEntryID = 0

	_ = m.newEntry(TypeNote)

	if got := m.editor.folderOptions[m.editor.folderIdx]; got != "Work" {
		t.Fatalf("expected new entry to default to the browsed folder %q, got %q", "Work", got)
	}
}

func newFolderEditorTestModel(t *testing.T) *Model {
	t.Helper()
	s := newTestStore(t)
	if _, err := s.CreateFolder("Home"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if _, err := s.CreateFolder("Work"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	m := &Model{store: s}
	_ = m.newEntry(TypeNote)
	// focus the folder field directly, bypassing the auto-open hook that
	// only fires through updateEditorKey's tab/arrow handling.
	for i, target := range m.editor.focusOrder {
		if target == eftFolder {
			m.editor.focusPos = i
		}
	}
	m.editor.applyFocus()
	return m
}

func TestTabbingIntoFolderFieldAutoOpensOverlay(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateFolder("Home"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	m := &Model{store: s}
	_ = m.newEntry(TypeNote)
	if m.editor.focusTarget != eftTitle {
		t.Fatalf("expected editor to start focused on the title field, got %v", m.editor.focusTarget)
	}

	nm, _ := m.updateEditorKey(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm

	if m.editor.focusTarget != eftFolder {
		t.Fatalf("expected tab to move focus to the folder field, got %v", m.editor.focusTarget)
	}
	if m.overlay != overlayFolderSelect {
		t.Fatalf("expected tabbing into the folder field to auto-open the picker, got overlay %v", m.overlay)
	}
	if m.folderSelect.cursor != m.editor.folderIdx {
		t.Fatalf("expected overlay cursor to start at the current folder selection")
	}
}

func TestShiftTabIntoFolderFieldAutoOpensOverlay(t *testing.T) {
	m := newFolderEditorTestModel(t)
	// move off the folder field first, then shift-tab back onto it
	nm, _ := m.updateEditorKey(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.editor.focusTarget == eftFolder {
		t.Fatalf("test setup: expected focus to move away from folder field")
	}

	nm, _ = m.updateEditorKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	*m = nm

	if m.editor.focusTarget != eftFolder {
		t.Fatalf("expected shift+tab to return focus to the folder field, got %v", m.editor.focusTarget)
	}
	if m.overlay != overlayFolderSelect {
		t.Fatalf("expected shift+tab into the folder field to auto-open the picker, got overlay %v", m.overlay)
	}
}

func TestFolderSelectOverlayListsAllFolders(t *testing.T) {
	m := newFolderEditorTestModel(t)
	m.overlay = overlayFolderSelect
	m.folderSelect = folderSelectModel{cursor: 0}

	view := m.viewFolderSelect()
	for _, want := range []string{"— (root)", "Home", "Work"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected folder list to contain %q, got %q", want, view)
		}
	}
}

func TestFolderSelectOverlayEnterAppliesChoice(t *testing.T) {
	m := newFolderEditorTestModel(t)
	m.overlay = overlayFolderSelect

	// find "Work" in the options and select it
	workIdx := -1
	for i, opt := range m.editor.folderOptions {
		if opt == "Work" {
			workIdx = i
		}
	}
	if workIdx < 0 {
		t.Fatalf("expected Work folder in options: %v", m.editor.folderOptions)
	}
	m.folderSelect = folderSelectModel{cursor: workIdx}

	nm, _ := m.updateFolderSelectKey("enter")
	*m = nm

	if m.overlay != overlayEditor {
		t.Fatalf("expected to return to the editor overlay, got %v", m.overlay)
	}
	if m.editor.folderIdx != workIdx {
		t.Fatalf("expected folderIdx to be updated to %d, got %d", workIdx, m.editor.folderIdx)
	}
}

func TestFolderSelectOverlayEscCancelsWithoutChange(t *testing.T) {
	m := newFolderEditorTestModel(t)
	m.overlay = overlayFolderSelect
	original := m.editor.folderIdx
	m.folderSelect = folderSelectModel{cursor: original + 1}

	nm, _ := m.updateFolderSelectKey("esc")
	*m = nm

	if m.overlay != overlayEditor {
		t.Fatalf("expected esc to return to the editor overlay, got %v", m.overlay)
	}
	if m.editor.folderIdx != original {
		t.Fatalf("expected folderIdx to remain %d after esc, got %d", original, m.editor.folderIdx)
	}
}

func TestFolderSelectOverlayNavigation(t *testing.T) {
	m := newFolderEditorTestModel(t)
	m.overlay = overlayFolderSelect
	m.folderSelect = folderSelectModel{cursor: 0}

	nm, _ := m.updateFolderSelectKey("down")
	*m = nm
	if m.folderSelect.cursor != 1 {
		t.Fatalf("expected cursor to move down, got %d", m.folderSelect.cursor)
	}
	nm, _ = m.updateFolderSelectKey("up")
	*m = nm
	if m.folderSelect.cursor != 0 {
		t.Fatalf("expected cursor to move up, got %d", m.folderSelect.cursor)
	}
	nm, _ = m.updateFolderSelectKey("up")
	*m = nm
	if m.folderSelect.cursor != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", m.folderSelect.cursor)
	}
}
