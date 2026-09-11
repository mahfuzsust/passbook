package ui

import (
	"strings"
	"testing"
)

func TestSyncSelectionFromTreeFolder(t *testing.T) {
	s := newTestStore(t)
	folderID, _ := s.CreateFolder("Work")
	m := &Model{store: s}
	m.main = newMainModel()
	m.main.refreshTree(s, "")
	m.main.tree.selectRef(nodeRef{IsFolder: true, ID: folderID})

	m.syncSelectionFromTree()
	if m.main.currentFolderID != folderID {
		t.Fatalf("expected currentFolderID %d, got %d", folderID, m.main.currentFolderID)
	}
	if m.main.showContent {
		t.Fatalf("did not expect content to be shown for a folder selection")
	}
}

func TestSyncSelectionFromTreeEntry(t *testing.T) {
	s := newTestStore(t)
	entryID, _ := s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "N"})
	m := &Model{store: s}
	m.main = newMainModel()
	m.main.refreshTree(s, "")
	m.main.tree.selectRef(nodeRef{IsFolder: false, ID: entryID})

	m.syncSelectionFromTree()
	if m.main.currentEntryID != entryID || !m.main.showContent {
		t.Fatalf("expected entry to be loaded, got id=%d showContent=%v", m.main.currentEntryID, m.main.showContent)
	}
}

func TestLoadEntry(t *testing.T) {
	s := newTestStore(t)
	entryID, _ := s.SaveEntry(0, &Entry{Type: string(TypeLogin), Title: "Site", Username: "u"})
	m := &Model{store: s}
	m.main = newMainModel()

	m.loadEntry(entryID)
	if m.main.currentEnt == nil || m.main.currentEnt.Username != "u" {
		t.Fatalf("expected entry to be loaded, got %+v", m.main.currentEnt)
	}
	if !m.main.showContent || m.main.showSensitive {
		t.Fatalf("expected showContent true and showSensitive reset to false")
	}
}

func TestLoadEntryMissingIsNoOp(t *testing.T) {
	s := newTestStore(t)
	m := &Model{store: s}
	m.main = newMainModel()
	m.loadEntry(99999)
	if m.main.currentEnt != nil {
		t.Fatalf("expected no entry to be loaded for a missing id")
	}
}

func TestViewMainRendersTreeAndDetail(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "Hello"})
	m := Model{store: s, width: 100, height: 30}
	m.main = newMainModel()
	m.main.refreshTree(s, "")

	got := m.viewMain()
	if !strings.Contains(got, "Hello") {
		t.Fatalf("expected tree to show entry title, got %q", got)
	}
}

func TestViewDetailPaneShowsKeybindingsWhenNoEntry(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.main = newMainModel()
	got := m.viewDetailPane()
	if !strings.Contains(got, "Keybindings") {
		t.Fatalf("expected keybindings help, got %q", got)
	}
}

func TestViewDetailPaneShowsEntryWhenSelected(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.main = newMainModel()
	m.main.showContent = true
	m.main.currentEnt = &Entry{Type: string(TypeNote), Title: "Note", CustomText: "hi"}
	got := m.viewDetailPane()
	if !strings.Contains(got, "Note") {
		t.Fatalf("expected entry view, got %q", got)
	}
}

func TestRenderKeybindings(t *testing.T) {
	got := renderKeybindings()
	if !strings.Contains(got, "Ctrl+A") || !strings.Contains(got, "u/c/l/t") {
		t.Fatalf("expected keybindings list, got %q", got)
	}
}
