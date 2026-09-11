package ui

import (
	"strings"
	"testing"
)

func TestEntryTypeIcon(t *testing.T) {
	cases := map[EntryType]string{
		TypeLogin: "🔐",
		TypeCard:  "💳",
		TypeNote:  "📝",
		TypeFile:  "📎",
	}
	for typ, want := range cases {
		if got := entryTypeIcon(string(typ)); got != want {
			t.Fatalf("%v: expected %q, got %q", typ, want, got)
		}
	}
	if got := entryTypeIcon("unknown"); got != "📄" {
		t.Fatalf("expected default icon, got %q", got)
	}
}

func TestRefreshTreeListsFoldersAndEntries(t *testing.T) {
	s := newTestStore(t)
	folderID, _ := s.CreateFolder("Work")
	_, _ = s.SaveEntry(folderID, &Entry{Type: string(TypeLogin), Title: "Inside"})
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "RootNote"})

	var tr treeState
	tr.refreshTree(s, "")

	var sawFolder, sawRoot bool
	for _, item := range tr.items {
		if item.ref.IsFolder && strings.Contains(item.label, "Work") {
			sawFolder = true
		}
		if !item.ref.IsFolder && strings.Contains(item.label, "RootNote") {
			sawRoot = true
		}
	}
	if !sawFolder || !sawRoot {
		t.Fatalf("expected folder and root entry in tree, got %+v", tr.items)
	}
}

func TestRefreshTreeFiltersByTitle(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "Alpha"})
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "Beta"})

	var tr treeState
	tr.refreshTree(s, "alp")

	if len(tr.items) != 1 || !strings.Contains(tr.items[0].label, "Alpha") {
		t.Fatalf("expected only Alpha to match filter, got %+v", tr.items)
	}
}

func TestTreeMoveUpDown(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "A"})
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "B"})
	var tr treeState
	tr.refreshTree(s, "")

	tr.moveDown()
	if tr.flatCursor != 1 {
		t.Fatalf("expected cursor 1, got %d", tr.flatCursor)
	}
	tr.moveDown()
	if tr.flatCursor != 1 {
		t.Fatalf("expected cursor to stay at last item, got %d", tr.flatCursor)
	}
	tr.moveUp()
	if tr.flatCursor != 0 {
		t.Fatalf("expected cursor 0, got %d", tr.flatCursor)
	}
	tr.moveUp()
	if tr.flatCursor != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", tr.flatCursor)
	}
}

func TestTreeToggleOrSelect(t *testing.T) {
	s := newTestStore(t)
	folderID, _ := s.CreateFolder("Work")
	_, _ = s.SaveEntry(folderID, &Entry{Type: string(TypeNote), Title: "Inside"})
	var tr treeState
	tr.refreshTree(s, "")

	ref, isToggle := tr.toggleOrSelect()
	if !isToggle || !ref.IsFolder {
		t.Fatalf("expected folder toggle at cursor 0")
	}

	tr.flatCursor = len(tr.items) - 1
	entRef, isToggle2 := tr.toggleOrSelect()
	if isToggle2 || entRef.IsFolder {
		t.Fatalf("expected entry selection, not a toggle")
	}
}

func TestTreeToggleOrSelectOutOfBounds(t *testing.T) {
	var tr treeState
	ref, isToggle := tr.toggleOrSelect()
	if isToggle || ref != (nodeRef{}) {
		t.Fatalf("expected zero-value result for an empty tree")
	}
}

func TestTreeCurrentRefAndSelectRef(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "A"})
	id2, _ := s.SaveEntry(0, &Entry{Type: string(TypeNote), Title: "B"})
	var tr treeState
	tr.refreshTree(s, "")

	tr.selectRef(nodeRef{IsFolder: false, ID: id2})
	if tr.currentRef().ID != id2 {
		t.Fatalf("expected selectRef to move cursor to id %d, got %+v", id2, tr.currentRef())
	}
}

func TestTreeCurrentRefOutOfBounds(t *testing.T) {
	var tr treeState
	if ref := tr.currentRef(); ref != (nodeRef{}) {
		t.Fatalf("expected zero-value ref for empty tree, got %+v", ref)
	}
}

func TestTreeRender(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.SaveEntry(0, &Entry{Type: string(TypeLogin), Title: "Site"})
	var tr treeState
	tr.refreshTree(s, "")
	out := tr.render(20)
	if !strings.Contains(out, "Site") {
		t.Fatalf("expected rendered tree to contain entry title, got %q", out)
	}
}

func TestListFolders(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.CreateFolder("Zeta")
	_, _ = s.CreateFolder("Alpha")
	names := listFolders(s)
	if len(names) != 2 || names[0] != "Alpha" || names[1] != "Zeta" {
		t.Fatalf("expected alphabetically sorted folder names, got %v", names)
	}
}
