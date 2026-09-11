package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewFolderModels(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.CreateFolder("Work")

	c := newFolderCreateModel()
	if c.mode != "create" {
		t.Fatalf("expected create mode")
	}

	r := newFolderRenameModel(s, id)
	if r.mode != "rename" || r.nameInput.Value() != "Work" {
		t.Fatalf("expected rename model prefilled with folder name, got %q", r.nameInput.Value())
	}

	_, _ = s.SaveEntry(id, &Entry{Type: string(TypeNote), Title: "N"})
	d := newFolderDeleteModel(s, id)
	if !strings.Contains(d.deleteText, "1 item") {
		t.Fatalf("expected delete text to mention item count, got %q", d.deleteText)
	}

	emptyID, _ := s.CreateFolder("Empty")
	d2 := newFolderDeleteModel(s, emptyID)
	if !strings.Contains(d2.deleteText, "empty folder") {
		t.Fatalf("expected empty folder text, got %q", d2.deleteText)
	}
}

func TestIsValidFolderName(t *testing.T) {
	valid := []string{"Work", "My Folder", "2024"}
	invalid := []string{"", ".", "..", ".hidden", "_temp", "a/b", "a:b", "a<b", "a>b", "a\"b", "a\\b", "a|b", "a?b", "a*b"}
	for _, v := range valid {
		if !isValidFolderName(v) {
			t.Fatalf("expected %q to be valid", v)
		}
	}
	for _, v := range invalid {
		if isValidFolderName(v) {
			t.Fatalf("expected %q to be invalid", v)
		}
	}
}

func TestDoFolderCreate(t *testing.T) {
	s := newTestStore(t)
	m := &Model{store: s, overlay: overlayFolderCreate}
	m.folder = newFolderCreateModel()
	m.folder.nameInput.SetValue("NewFolder")

	m.doFolderCreate()
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to close after create")
	}
	f, _ := s.GetFolderByName("NewFolder")
	if f == nil {
		t.Fatalf("expected folder to be created")
	}
}

func TestDoFolderCreateRejectsInvalidName(t *testing.T) {
	s := newTestStore(t)
	m := &Model{store: s, overlay: overlayFolderCreate}
	m.folder = newFolderCreateModel()
	m.folder.nameInput.SetValue(".hidden")

	m.doFolderCreate()
	if m.overlay != overlayFolderCreate {
		t.Fatalf("expected overlay to remain open for invalid name")
	}
}

func TestDoFolderRename(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.CreateFolder("Old")
	m := &Model{store: s, overlay: overlayFolderRename}
	m.folder = newFolderRenameModel(s, id)
	m.folder.nameInput.SetValue("New")

	m.doFolderRename()
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to close after rename")
	}
	f, _ := s.GetFolder(id)
	if f.Name != "New" {
		t.Fatalf("expected folder renamed, got %q", f.Name)
	}
}

func TestDoFolderDelete(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.CreateFolder("ToDelete")
	m := &Model{store: s, main: mainModel{currentFolderID: id, showContent: true}}
	m.folder = folderModel{folderID: id}

	m.doFolderDelete()
	if f, _ := s.GetFolder(id); f != nil {
		t.Fatalf("expected folder to be deleted")
	}
	if m.main.currentFolderID != 0 || m.main.showContent {
		t.Fatalf("expected main state reset after delete")
	}
}

func TestDoFolderDeleteNoOpWithZeroID(t *testing.T) {
	s := newTestStore(t)
	m := &Model{store: s}
	m.folder = folderModel{folderID: 0}
	m.doFolderDelete() // should not panic or error
}

func TestUpdateFolderKeyEscAndEnter(t *testing.T) {
	s := newTestStore(t)
	m := &Model{store: s, overlay: overlayFolderCreate}
	m.folder = newFolderCreateModel()

	nm, _ := m.updateFolderKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	*m = nm
	if m.folder.nameInput.Value() != "A" {
		t.Fatalf("expected typed character in name input, got %q", m.folder.nameInput.Value())
	}

	nm, _ = m.updateFolderKey(tea.KeyMsg{Type: tea.KeyEnter})
	*m = nm
	if m.overlay != overlayNone {
		t.Fatalf("expected enter to create the folder and close overlay")
	}
}

func TestUpdateFolderKeyEsc(t *testing.T) {
	m := &Model{overlay: overlayFolderCreate}
	m.folder = newFolderCreateModel()
	nm, _ := m.updateFolderKey(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.overlay != overlayNone {
		t.Fatalf("expected esc to close overlay")
	}
}

func TestUpdateFolderDeleteNavigationAndConfirm(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.CreateFolder("Del")
	m := &Model{store: s, overlay: overlayFolderDelete}
	m.folder = folderModel{folderID: id}

	nm, _ := m.updateFolderDelete("right")
	*m = nm
	if m.folder.btnFocus != 1 {
		t.Fatalf("expected btnFocus 1, got %d", m.folder.btnFocus)
	}
	nm, _ = m.updateFolderDelete("left")
	*m = nm
	if m.folder.btnFocus != 0 {
		t.Fatalf("expected btnFocus 0, got %d", m.folder.btnFocus)
	}
	nm, _ = m.updateFolderDelete("enter")
	*m = nm
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to close")
	}
	if f, _ := s.GetFolder(id); f != nil {
		t.Fatalf("expected folder deleted on confirm")
	}
}

func TestUpdateFolderDeleteEsc(t *testing.T) {
	m := &Model{overlay: overlayFolderDelete}
	nm, _ := m.updateFolderDelete("esc")
	if nm.overlay != overlayNone {
		t.Fatalf("expected esc to close overlay")
	}
}

func TestFolderViewsRender(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.CreateFolder("Work")
	m := Model{width: 80, height: 24, store: s}
	m.folder = newFolderCreateModel()
	if !strings.Contains(m.viewFolderCreate(), "New Folder") {
		t.Fatalf("expected create title")
	}
	m.folder = newFolderRenameModel(s, id)
	if !strings.Contains(m.viewFolderRename(), "Rename Folder") {
		t.Fatalf("expected rename title")
	}
	m.folder = newFolderDeleteModel(s, id)
	if !strings.Contains(m.viewFolderDelete(), "Delete") {
		t.Fatalf("expected delete button text")
	}
}
