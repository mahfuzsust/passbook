package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCreateMenuHasFolderOption(t *testing.T) {
	var found bool
	for _, item := range createMenuItems {
		if item.isFolder && item.key == "d" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a folder creation option in the create menu")
	}
}

func TestCreateMenuSelectFolderOpensFolderCreate(t *testing.T) {
	m := &Model{}
	nm, _ := m.updateCreateMenu("d")
	if nm.overlay != overlayFolderCreate {
		t.Fatalf("expected folder create overlay, got %v", nm.overlay)
	}
}

func TestCreateMenuSelectLoginOpensEditor(t *testing.T) {
	m := &Model{}
	nm, _ := m.updateCreateMenu("l")
	if nm.overlay != overlayEditor {
		t.Fatalf("expected editor overlay for login entry, got %v", nm.overlay)
	}
	if nm.editor.entryType != TypeLogin {
		t.Fatalf("expected login entry type, got %v", nm.editor.entryType)
	}
}

func TestCreateMenuEnterUsesCursorSelection(t *testing.T) {
	m := &Model{}
	// find the folder item's index to move the cursor there
	idx := -1
	for i, item := range createMenuItems {
		if item.isFolder {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("expected folder item to exist")
	}
	m.createMenu.cursor = idx
	nm, _ := m.updateCreateMenu("enter")
	if nm.overlay != overlayFolderCreate {
		t.Fatalf("expected enter on folder cursor position to open folder create, got %v", nm.overlay)
	}
}

func TestCtrlNStillOpensFolderCreate(t *testing.T) {
	m := &Model{screen: screenMain}
	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	if nm.overlay != overlayFolderCreate {
		t.Fatalf("expected ctrl+n to open folder create, got %v", nm.overlay)
	}
}
