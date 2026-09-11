package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShowQuickCopyBuildsLoginItems(t *testing.T) {
	m := &Model{main: mainModel{
		currentEnt: &Entry{
			Type:       string(TypeLogin),
			Username:   "alice",
			Password:   "hunter2",
			Link:       "http://example.com",
			TotpSecret: "JBSWY3DPEHPK3PXP",
		},
		currentEntryID: 1,
	}}
	m.showQuickCopy()
	if m.overlay != overlayQuickCopy {
		t.Fatalf("expected quick copy overlay to open")
	}
	keys := map[string]bool{}
	for _, item := range m.quickCopy.items {
		keys[item.key] = true
	}
	for _, want := range []string{"u", "c", "l", "t"} {
		if !keys[want] {
			t.Fatalf("expected quick copy item with key %q, got items %+v", want, m.quickCopy.items)
		}
	}
}

func TestShowQuickCopyNoEntryDoesNothing(t *testing.T) {
	m := &Model{}
	m.showQuickCopy()
	if m.overlay == overlayQuickCopy {
		t.Fatalf("did not expect quick copy overlay without a current entry")
	}
}

func TestUpdateQuickCopySelectByKeyRunsActionAndCloses(t *testing.T) {
	ran := false
	m := &Model{}
	m.overlay = overlayQuickCopy
	m.quickCopy = quickCopyModel{items: []quickCopyItem{
		{label: "Username", key: "u", action: func(m *Model) tea.Cmd {
			ran = true
			return nil
		}},
	}}
	nm, _ := m.updateQuickCopy("u")
	if !ran {
		t.Fatalf("expected action to run")
	}
	if nm.overlay != overlayNone {
		t.Fatalf("expected overlay to close after action")
	}
}

func TestUpdateQuickCopyEnterRunsSelectedAction(t *testing.T) {
	ran := false
	m := &Model{}
	m.overlay = overlayQuickCopy
	m.quickCopy = quickCopyModel{cursor: 1, items: []quickCopyItem{
		{label: "Username", key: "u", action: func(m *Model) tea.Cmd { return nil }},
		{label: "Password", key: "c", action: func(m *Model) tea.Cmd {
			ran = true
			return nil
		}},
	}}
	nm, _ := m.updateQuickCopy("enter")
	if !ran {
		t.Fatalf("expected enter to run the highlighted item's action, not just close the overlay")
	}
	if nm.overlay != overlayNone {
		t.Fatalf("expected overlay to close after enter")
	}
}

func TestUpdateQuickCopyPropagatesCmd(t *testing.T) {
	m := &Model{}
	m.overlay = overlayQuickCopy
	wantCmd := func() tea.Msg { return clipboardClearMsg{text: "x"} }
	m.quickCopy = quickCopyModel{items: []quickCopyItem{
		{label: "Password", key: "c", action: func(m *Model) tea.Cmd { return wantCmd }},
	}}
	_, cmd := m.updateQuickCopy("c")
	if cmd == nil {
		t.Fatalf("expected cmd to be propagated from the action")
	}
	if _, ok := cmd().(clipboardClearMsg); !ok {
		t.Fatalf("expected propagated cmd to produce clipboardClearMsg")
	}
}

func TestUpdateQuickCopyNavigation(t *testing.T) {
	m := &Model{}
	m.quickCopy = quickCopyModel{items: []quickCopyItem{
		{key: "u"}, {key: "c"}, {key: "l"},
	}}
	nm, _ := m.updateQuickCopy("down")
	if nm.quickCopy.cursor != 1 {
		t.Fatalf("expected cursor to move down, got %d", nm.quickCopy.cursor)
	}
	nm, _ = nm.updateQuickCopy("down")
	if nm.quickCopy.cursor != 2 {
		t.Fatalf("expected cursor to move down again, got %d", nm.quickCopy.cursor)
	}
	nm, _ = nm.updateQuickCopy("down")
	if nm.quickCopy.cursor != 2 {
		t.Fatalf("expected cursor to stay at last item, got %d", nm.quickCopy.cursor)
	}
	nm, _ = nm.updateQuickCopy("up")
	if nm.quickCopy.cursor != 1 {
		t.Fatalf("expected cursor to move up, got %d", nm.quickCopy.cursor)
	}
}

func TestUpdateQuickCopyEsc(t *testing.T) {
	m := &Model{overlay: overlayQuickCopy}
	nm, _ := m.updateQuickCopy("esc")
	if nm.overlay != overlayNone {
		t.Fatalf("expected esc to close overlay")
	}
}
