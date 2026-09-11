package ui

import (
	"strings"
	"testing"
)

func TestNewModals(t *testing.T) {
	d := newDeleteModal("Item")
	if d.mode != "delete" || !strings.Contains(d.text, "Item") {
		t.Fatalf("unexpected delete modal: %+v", d)
	}
	h := newHistoryModal([]PasswordHistory{{Password: "p1", Date: "d1"}})
	if h.mode != "history" || len(h.history) != 1 {
		t.Fatalf("unexpected history modal: %+v", h)
	}
	e := newErrorModal("oops")
	if e.mode != "error" || e.text != "oops" {
		t.Fatalf("unexpected error modal: %+v", e)
	}
}

func TestUpdateModalNavigation(t *testing.T) {
	m := &Model{overlay: overlayDelete}
	m.modals = newDeleteModal("X")

	nm, _ := m.updateModal("right")
	*m = nm
	if m.modals.btnFocus != 1 {
		t.Fatalf("expected btnFocus 1, got %d", m.modals.btnFocus)
	}
	nm, _ = m.updateModal("left")
	*m = nm
	if m.modals.btnFocus != 0 {
		t.Fatalf("expected btnFocus 0, got %d", m.modals.btnFocus)
	}
	nm, _ = m.updateModal("esc")
	*m = nm
	if m.overlay != overlayNone {
		t.Fatalf("expected esc to close overlay")
	}
}

func TestUpdateModalDeleteConfirm(t *testing.T) {
	s := newTestStore(t)
	entryID, _ := s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "ToDelete"})
	m := &Model{store: s, overlay: overlayDelete}
	m.main.currentEntryID = entryID
	m.modals = newDeleteModal("ToDelete")

	nm, _ := m.updateModal("enter")
	*m = nm
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to close after confirming delete")
	}
	if _, err := s.LoadEntry(entryID); err == nil {
		t.Fatalf("expected entry to be deleted")
	}
}

func TestUpdateModalFolderDeleteConfirm(t *testing.T) {
	s := newTestStore(t)
	folderID, _ := s.CreateFolder("ToDelete")
	m := &Model{store: s, overlay: overlayFolderDelete}
	m.folder = folderModel{folderID: folderID}
	m.modals = newDeleteModal("ToDelete")

	nm, _ := m.updateModal("enter")
	*m = nm
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to close")
	}
	if f, _ := s.GetFolder(folderID); f != nil {
		t.Fatalf("expected folder to be deleted")
	}
}

func TestModalViewsRender(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.modals = newDeleteModal("Thing")
	if !strings.Contains(m.viewDeleteModal(), "Thing") {
		t.Fatalf("expected delete modal view to mention item name")
	}
	m.modals = newHistoryModal([]PasswordHistory{{Password: "p1", Date: "2024-01-01"}})
	if !strings.Contains(m.viewHistory(), "p1") {
		t.Fatalf("expected history view to show password entries")
	}
	m.modals = newErrorModal("bad thing happened")
	if !strings.Contains(m.viewErrorModal(), "bad thing happened") {
		t.Fatalf("expected error view to show message")
	}
}
