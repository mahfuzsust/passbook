package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newPassgenTestModel() *Model {
	ent := NewEntry(TypeLogin)
	m := &Model{}
	m.editor = newEditorModel(ent, 0, 0, nil)
	m.overlay = overlayPassgen
	m.passgen = newPassgenModel()
	m.passgen.updatePreview()
	return m
}

func TestNewPassgenModelHasSaneDefaults(t *testing.T) {
	p := newPassgenModel()
	if !p.upper || !p.lower || !p.special {
		t.Fatalf("expected all character classes enabled by default")
	}
	if p.length.Value() != "28" {
		t.Fatalf("expected default length 28, got %q", p.length.Value())
	}
}

func TestPassgenGeneratesPreviewOnOpen(t *testing.T) {
	m := newPassgenTestModel()
	if m.passgen.preview == "" {
		t.Fatalf("expected a password preview to be generated on open")
	}
	if len(m.passgen.preview) != 28 {
		t.Fatalf("expected default length 28, got %d", len(m.passgen.preview))
	}
}

func TestPassgenTabNavigationReachesUseButton(t *testing.T) {
	m := newPassgenTestModel()
	// length(0) -> upper(1) -> lower(2) -> special(3) -> refresh(4) -> use(5)
	for i := 0; i < 5; i++ {
		nm, _ := m.updatePassgenKey(tea.KeyMsg{Type: tea.KeyTab})
		*m = nm
	}
	if m.passgen.cursor != 5 {
		t.Fatalf("expected tab to reach the Use button (cursor 5), got %d", m.passgen.cursor)
	}
}

func TestPassgenEnterOnUseAppliesPasswordAndCloses(t *testing.T) {
	m := newPassgenTestModel()
	preview := m.passgen.preview
	m.passgen.cursor = 5 // Use button

	nm, _ := m.updatePassgenKey(tea.KeyMsg{Type: tea.KeyEnter})
	*m = nm

	if m.overlay != overlayEditor {
		t.Fatalf("expected editor overlay after using generated password, got %v", m.overlay)
	}
	if m.editor.password.Value() != preview {
		t.Fatalf("expected password field to be set to %q, got %q", preview, m.editor.password.Value())
	}
}

func TestPassgenEnterOnRefreshRegeneratesWithoutClosing(t *testing.T) {
	m := newPassgenTestModel()
	m.passgen.cursor = 4 // Refresh button

	nm, _ := m.updatePassgenKey(tea.KeyMsg{Type: tea.KeyEnter})
	*m = nm

	if m.overlay != overlayPassgen {
		t.Fatalf("expected to remain on the passgen overlay after refresh, got %v", m.overlay)
	}
	if m.passgen.preview == "" {
		t.Fatalf("expected a regenerated preview")
	}
}

func TestPassgenSpaceTogglesCharacterClasses(t *testing.T) {
	m := newPassgenTestModel()
	m.passgen.cursor = 3 // special
	nm, _ := m.updatePassgenKey(tea.KeyMsg{Type: tea.KeySpace})
	*m = nm
	if m.passgen.special {
		t.Fatalf("expected special to be toggled off")
	}
}

func TestPassgenEscReturnsToEditor(t *testing.T) {
	m := newPassgenTestModel()
	nm, _ := m.updatePassgenKey(tea.KeyMsg{Type: tea.KeyEsc})
	*m = nm
	if m.overlay != overlayEditor {
		t.Fatalf("expected esc to return to editor overlay")
	}
}

func TestPassgenLeftRightMoveBetweenButtons(t *testing.T) {
	m := newPassgenTestModel()
	m.passgen.cursor = 4
	nm, _ := m.updatePassgenKey(tea.KeyMsg{Type: tea.KeyRight})
	*m = nm
	if m.passgen.cursor != 5 {
		t.Fatalf("expected right arrow to move from refresh to use, got %d", m.passgen.cursor)
	}
	nm, _ = m.updatePassgenKey(tea.KeyMsg{Type: tea.KeyLeft})
	*m = nm
	if m.passgen.cursor != 4 {
		t.Fatalf("expected left arrow to move back to refresh, got %d", m.passgen.cursor)
	}
}

func TestCtrlGOpensPassgenForLoginOnly(t *testing.T) {
	ent := NewEntry(TypeLogin)
	m := &Model{}
	m.editor = newEditorModel(ent, 0, 0, nil)
	m.overlay = overlayEditor

	nm, _ := m.updateEditorKey(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = &nm
	if m.overlay != overlayPassgen {
		t.Fatalf("expected ctrl+g to open the password generator")
	}
	if m.passgen.preview == "" {
		t.Fatalf("expected preview to already be populated when opening")
	}
}

func TestCtrlGDoesNothingForNonLoginEntries(t *testing.T) {
	ent := NewEntry(TypeCard)
	m := &Model{}
	m.editor = newEditorModel(ent, 0, 0, nil)
	m.overlay = overlayEditor

	nm, _ := m.updateEditorKey(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = &nm
	if m.overlay != overlayEditor {
		t.Fatalf("expected ctrl+g to be a no-op for non-login entries")
	}
}

func TestPassgenViewHighlightsFocusedCheckbox(t *testing.T) {
	m := newPassgenTestModel()
	m.width, m.height = 80, 24

	m.passgen.cursor = 1 // A-Z checkbox
	upperView := m.viewPassgen()

	m.passgen.cursor = 2 // a-z checkbox
	lowerView := m.viewPassgen()

	if upperView == lowerView {
		t.Fatalf("expected focus indicator to change when moving between checkboxes")
	}
}
