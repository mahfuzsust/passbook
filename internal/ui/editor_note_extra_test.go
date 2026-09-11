package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newNoteEditorTestModel() *Model {
	ent := NewEntry(TypeNote)
	m := &Model{}
	m.editor = newEditorModel(ent, 0, 0, nil)
	m.overlay = overlayEditor
	m.editor.focusOrder = m.editor.buildFocusOrder()
	// focus the notes textarea directly
	for i, t := range m.editor.focusOrder {
		if t == eftNotes {
			m.editor.focusPos = i
		}
	}
	m.editor.applyFocus()
	return m
}

func TestEnterInNotesInsertsNewlineInsteadOfMovingFocus(t *testing.T) {
	m := newNoteEditorTestModel()
	m.editor.notes.SetValue("line1")

	nm, _ := m.updateEditorKey(tea.KeyMsg{Type: tea.KeyEnter})
	*m = nm

	if m.editor.focusTarget != eftNotes {
		t.Fatalf("expected focus to remain on notes textarea, got %v", m.editor.focusTarget)
	}
	if !strings.Contains(m.editor.notes.Value(), "\n") {
		t.Fatalf("expected a newline to be inserted, got %q", m.editor.notes.Value())
	}
}

func TestUpDownInNotesDoNotChangeFocus(t *testing.T) {
	m := newNoteEditorTestModel()
	m.editor.notes.SetValue("line1\nline2\nline3")

	nm, _ := m.updateEditorKey(tea.KeyMsg{Type: tea.KeyDown})
	*m = nm
	if m.editor.focusTarget != eftNotes {
		t.Fatalf("expected down arrow to stay within notes, got %v", m.editor.focusTarget)
	}

	nm, _ = m.updateEditorKey(tea.KeyMsg{Type: tea.KeyUp})
	*m = nm
	if m.editor.focusTarget != eftNotes {
		t.Fatalf("expected up arrow to stay within notes, got %v", m.editor.focusTarget)
	}
}

func TestTabStillMovesOutOfNotes(t *testing.T) {
	m := newNoteEditorTestModel()
	nm, _ := m.updateEditorKey(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.editor.focusTarget == eftNotes {
		t.Fatalf("expected tab to move focus away from notes")
	}
}
