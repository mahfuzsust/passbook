package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newMainKeyTestModel(t *testing.T) *Model {
	t.Helper()
	s := newTestStore(t)
	m := &Model{store: s, screen: screenMain, width: 80, height: 24}
	m.main = newMainModel()
	m.main.refreshTree(s, "")
	return m
}

func TestUpdateMainKeyCtrlA(t *testing.T) {
	m := newMainKeyTestModel(t)
	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlA})
	if nm.overlay != overlayCreateMenu {
		t.Fatalf("expected create menu overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlEEntersEditorForEntry(t *testing.T) {
	m := newMainKeyTestModel(t)
	entryID, _ := m.store.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "N"})
	m.main.currentEntryID = entryID
	m.main.currentEnt = &Entry{ID: entryID, Type: string(TypeNote), Title: "N"}

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlE})
	if nm.overlay != overlayEditor {
		t.Fatalf("expected editor overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlERenamesFolder(t *testing.T) {
	m := newMainKeyTestModel(t)
	folderID, _ := m.store.CreateFolder("Work")
	m.main.currentFolderID = folderID

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlE})
	if nm.overlay != overlayFolderRename {
		t.Fatalf("expected folder rename overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlDDeletesEntry(t *testing.T) {
	m := newMainKeyTestModel(t)
	entryID, _ := m.store.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "N"})
	m.main.currentEntryID = entryID
	m.main.currentEnt = &Entry{ID: entryID, Type: string(TypeNote), Title: "N"}

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	if nm.overlay != overlayDelete {
		t.Fatalf("expected delete overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlDDeletesFolder(t *testing.T) {
	m := newMainKeyTestModel(t)
	folderID, _ := m.store.CreateFolder("Work")
	m.main.currentFolderID = folderID

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	if nm.overlay != overlayFolderDelete {
		t.Fatalf("expected folder delete overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlN(t *testing.T) {
	m := newMainKeyTestModel(t)
	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	if nm.overlay != overlayFolderCreate {
		t.Fatalf("expected folder create overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlF(t *testing.T) {
	m := newMainKeyTestModel(t)
	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlF})
	if !nm.main.searchFocused {
		t.Fatalf("expected search to become focused")
	}
}

func TestUpdateMainKeyCtrlY(t *testing.T) {
	m := newMainKeyTestModel(t)
	m.main.currentEnt = &Entry{Type: string(TypeNote), Title: "N", CustomText: "hi"}
	m.main.currentEntryID = 1
	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlY})
	if nm.overlay != overlayQuickCopy {
		t.Fatalf("expected quick copy overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlP(t *testing.T) {
	m := newMainKeyTestModel(t)
	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlP})
	if nm.overlay != overlayChangePwd {
		t.Fatalf("expected change password overlay, got %v", nm.overlay)
	}
}

func TestUpdateMainKeyCtrlQQuits(t *testing.T) {
	m := newMainKeyTestModel(t)
	_, cmd := m.updateMainKey(tea.KeyMsg{Type: tea.KeyCtrlQ})
	if cmd == nil {
		t.Fatalf("expected quit cmd")
	}
}

func TestUpdateMainKeyEscBlursSearch(t *testing.T) {
	m := newMainKeyTestModel(t)
	m.main.searchFocused = false
	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.main.searchFocused {
		t.Fatalf("expected search to remain unfocused")
	}
}

func TestUpdateMainKeyUpDownMovesTreeAndLoadsEntry(t *testing.T) {
	m := newMainKeyTestModel(t)
	_, _ = m.store.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "A"})
	_, _ = m.store.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "B"})
	m.main.refreshTree(m.store, "")

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyDown})
	*m = nm
	if !m.main.showContent {
		t.Fatalf("expected entry content to show after moving down")
	}
	nm, _ = m.updateMainKey(tea.KeyMsg{Type: tea.KeyUp})
	*m = nm
}

func TestUpdateMainKeyEnterTogglesFolder(t *testing.T) {
	m := newMainKeyTestModel(t)
	folderID, _ := m.store.CreateFolder("Work")
	_, _ = m.store.SaveEntry(folderID, &Entry{Type: string(TypeNote), Title: "Inside"})
	m.main.refreshTree(m.store, "")

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyEnter})
	*m = nm
	if m.main.currentFolderID != folderID {
		t.Fatalf("expected folder to be selected, got %d", m.main.currentFolderID)
	}
}

func TestUpdateMainKeySearchTypingRefreshesTree(t *testing.T) {
	m := newMainKeyTestModel(t)
	_, _ = m.store.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "Alpha"})
	m.main.searchFocused = true
	m.main.search.Focus()

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	*m = nm
	if m.main.search.Value() != "a" {
		t.Fatalf("expected typed character in search box, got %q", m.main.search.Value())
	}
}

func TestUpdateMainKeySearchEnterUnfocuses(t *testing.T) {
	m := newMainKeyTestModel(t)
	m.main.searchFocused = true
	m.main.search.Focus()

	nm, _ := m.updateMainKey(tea.KeyMsg{Type: tea.KeyEnter})
	*m = nm
	if m.main.searchFocused {
		t.Fatalf("expected enter to unfocus search")
	}
}
