package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAnsiFg(t *testing.T) {
	cases := map[string]string{"9": "31", "11": "33", "12": "34", "10": "32", "unknown": "37"}
	for in, want := range cases {
		if got := ansiFg(in); got != want {
			t.Fatalf("%q: expected %q, got %q", in, want, got)
		}
	}
}

func TestFormatStrengthBarEmpty(t *testing.T) {
	if got := formatStrengthBar(""); got != "" {
		t.Fatalf("expected empty bar for empty password, got %q", got)
	}
}

func TestFormatStrengthBarNonEmpty(t *testing.T) {
	got := formatStrengthBar("Sup3r$ecureLongPassphrase!!")
	if got == "" {
		t.Fatalf("expected a rendered strength bar for a strong password")
	}
}

func TestRenderProgressBar(t *testing.T) {
	if got := renderProgressBar(1, 10); got != strings.Repeat("█", 10) {
		t.Fatalf("expected fully filled bar, got %q", got)
	}
	if got := renderProgressBar(0, 10); got != strings.Repeat("░", 10) {
		t.Fatalf("expected fully empty bar, got %q", got)
	}
	if got := renderProgressBar(-1, 10); got != strings.Repeat("░", 10) {
		t.Fatalf("expected negative fraction clamped to empty, got %q", got)
	}
	if got := renderProgressBar(2, 10); got != strings.Repeat("█", 10) {
		t.Fatalf("expected fraction >1 clamped to full, got %q", got)
	}
	mid := renderProgressBar(0.5, 10)
	if len([]rune(mid)) != 10 {
		t.Fatalf("expected bar to stay at requested width, got %q (%d runes)", mid, len([]rune(mid)))
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		500:             "500 B",
		2048:            "2.0 KB",
		5 * 1024 * 1024: "5.0 MB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Fatalf("%d: expected %q, got %q", in, want, got)
		}
	}
}

func TestNewEntryHelper(t *testing.T) {
	e := NewEntry(TypeCard)
	if e.Type != string(TypeCard) {
		t.Fatalf("expected card type, got %q", e.Type)
	}
}

func TestUpdateOverlayKeyDispatchesToHistoryAndError(t *testing.T) {
	m := &Model{overlay: overlayHistory}
	nm, _ := m.updateOverlayKey(tea.KeyMsg{Type: tea.KeyEsc})
	if nm.overlay != overlayNone {
		t.Fatalf("expected esc to close history overlay")
	}

	m2 := &Model{overlay: overlayError}
	nm2, _ := m2.updateOverlayKey(tea.KeyMsg{Type: tea.KeyEnter})
	if nm2.overlay != overlayNone {
		t.Fatalf("expected enter to close error overlay")
	}
}

func TestUpdateOverlayKeyNoneIsNoOp(t *testing.T) {
	m := &Model{overlay: overlayNone}
	nm, cmd := m.updateOverlayKey(tea.KeyMsg{Type: tea.KeyEnter})
	if nm.overlay != overlayNone || cmd != nil {
		t.Fatalf("expected no-op for overlayNone")
	}
}

func TestRenderOverlayDispatchesPerOverlay(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.modals = newErrorModal("boom")
	m.overlay = overlayError
	if !strings.Contains(m.renderOverlay(), "boom") {
		t.Fatalf("expected error overlay content")
	}

	m.overlay = overlayHistory
	m.modals = newHistoryModal(nil)
	if !strings.Contains(m.renderOverlay(), "History") {
		t.Fatalf("expected history overlay content")
	}

	m.overlay = overlayNone
	if m.renderOverlay() != "" {
		t.Fatalf("expected empty string for overlayNone")
	}
}

func TestDismissOverlay(t *testing.T) {
	m := &Model{overlay: overlayError}
	m.dismissOverlay()
	if m.overlay != overlayNone {
		t.Fatalf("expected overlay to be dismissed")
	}
}

func TestNewEntryOpensEditorWithType(t *testing.T) {
	m := &Model{store: newTestStore(t)}
	_ = m.newEntry(TypeCard)
	if m.overlay != overlayEditor || m.editor.entryType != TypeCard {
		t.Fatalf("expected editor overlay with card type, got overlay=%v type=%v", m.overlay, m.editor.entryType)
	}
}

func TestFolderModelUpdate(t *testing.T) {
	f := newFolderCreateModel()
	nf, _ := f.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if nf.nameInput.Value() != "x" {
		t.Fatalf("expected typed character, got %q", nf.nameInput.Value())
	}
}

func TestTreeIsFolderExpandedDefaultsTrue(t *testing.T) {
	var tr treeState
	if !tr.isFolderExpanded(1) {
		t.Fatalf("expected unknown folders to default to expanded")
	}
}

func TestTreeIsFolderExpandedRespectsExistingState(t *testing.T) {
	tr := treeState{items: []treeItem{{ref: nodeRef{IsFolder: true, ID: 1}, expanded: false}}}
	if tr.isFolderExpanded(1) {
		t.Fatalf("expected folder 1 to be collapsed")
	}
}

func TestViewQuickCopy(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.quickCopy = quickCopyModel{items: []quickCopyItem{{label: "Username", key: "u"}}}
	got := m.viewQuickCopy()
	if !strings.Contains(got, "Username") || !strings.Contains(got, "Quick Copy") {
		t.Fatalf("expected quick copy view content, got %q", got)
	}
}

func TestViewCreateMenu(t *testing.T) {
	m := Model{width: 80, height: 24}
	m.createMenu = newCreateMenuModel()
	got := m.viewCreateMenu()
	if !strings.Contains(got, "Login") || !strings.Contains(got, "Folder") {
		t.Fatalf("expected create menu items, got %q", got)
	}
}

func TestViewFileBrowser(t *testing.T) {
	dir := t.TempDir()
	m := Model{width: 80, height: 24}
	m.fileBrowser = newFileBrowserModel(dir)
	got := m.viewFileBrowser()
	if !strings.Contains(got, "Select File") {
		t.Fatalf("expected file browser title, got %q", got)
	}
}

func TestRenderLoginViewContentAndFormatTOTPDisplay(t *testing.T) {
	m := Model{main: mainModel{currentEnt: &Entry{
		Type: string(TypeLogin), Username: "alice", Password: "secret",
		Link: "http://example.com", TotpSecret: "JBSWY3DPEHPK3PXP",
	}}}
	var b strings.Builder
	renderLoginViewContent(&b, m)
	out := b.String()
	if !strings.Contains(out, "alice") || !strings.Contains(out, "http://example.com") {
		t.Fatalf("expected login fields rendered, got %q", out)
	}

	code, bar := formatTOTPDisplay("JBSWY3DPEHPK3PXP")
	if code == "" || bar == "" {
		t.Fatalf("expected non-empty totp code and bar")
	}
	if badCode, badBar := formatTOTPDisplay("not-valid-base32!!"); badCode != "" || badBar != "" {
		t.Fatalf("expected empty code/bar for invalid secret, got %q %q", badCode, badBar)
	}
}
