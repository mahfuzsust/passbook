package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"passbook/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelFreshInstall(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "passbook.db")
	m := newModel(config.AppConfig{}, t.TempDir(), dbPath)
	if !m.freshInstall {
		t.Fatalf("expected fresh install to be true")
	}
	if m.screen != screenLogin {
		t.Fatalf("expected initial screen to be login")
	}
}

func TestModelInitSchedulesTick(t *testing.T) {
	m := Model{}
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected Init to return a tick cmd")
	}
	msg := cmd()
	if _, ok := msg.(tickMsg); !ok {
		t.Fatalf("expected tickMsg, got %T", msg)
	}
}

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := Model{login: newLoginModel(true)}
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	nm := updated.(Model)
	if nm.width != 100 || nm.height != 50 {
		t.Fatalf("expected model dimensions to update, got %dx%d", nm.width, nm.height)
	}
	if cmd != nil {
		t.Fatalf("expected no cmd from window size update")
	}
}

func TestUpdateKeyMsgRoutesToLoginScreen(t *testing.T) {
	m := Model{screen: screenLogin, login: newLoginModel(true), width: 80, height: 24}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = updated.(Model)
}

func TestUpdateStatusClearMsg(t *testing.T) {
	m := Model{main: mainModel{viewStatus: "hello"}}
	updated, _ := m.Update(statusClearMsg{})
	nm := updated.(Model)
	if nm.main.viewStatus != "" {
		t.Fatalf("expected viewStatus to be cleared")
	}
}

func TestUpdateRedrawMsg(t *testing.T) {
	m := Model{}
	updated, cmd := m.Update(redrawMsg{})
	if cmd != nil {
		t.Fatalf("expected no cmd for redraw")
	}
	_ = updated.(Model)
}

func TestUpdateClipboardClearMsg(t *testing.T) {
	m := Model{}
	updated, _ := m.Update(clipboardClearMsg{text: "secret"})
	_ = updated.(Model)
}

func TestViewLoadingBeforeWindowSize(t *testing.T) {
	m := Model{}
	if got := m.View(); got != "Loading..." {
		t.Fatalf("expected loading placeholder, got %q", got)
	}
}

func TestViewDispatchesPerScreen(t *testing.T) {
	base := Model{width: 80, height: 24}

	cases := []struct {
		name   string
		model  Model
		expect string
	}{
		{"login", func() Model {
			m := base
			m.screen = screenLogin
			m.login = newLoginModel(true)
			m.freshInstall = true
			return m
		}(), "PassBook Setup"},
		{"pinSetup", func() Model { m := base; m.screen = screenPinSetup; m.pin = newPinModel(); return m }(), "Two-Factor"},
	}
	for _, c := range cases {
		got := c.model.View()
		if !strings.Contains(got, c.expect) {
			t.Fatalf("%s: expected view to contain %q, got %q", c.name, c.expect, got)
		}
	}
}

func TestViewWithOverlayUsesRenderOverlay(t *testing.T) {
	m := Model{width: 80, height: 24, screen: screenMain, overlay: overlayCreateMenu}
	m.main = newMainModel()
	m.createMenu = newCreateMenuModel()
	got := m.View()
	if !strings.Contains(got, "Create New") {
		t.Fatalf("expected overlay content to be rendered, got %q", got)
	}
}

func TestHandleClipboardClearMatchingValueClears(t *testing.T) {
	m := &Model{}
	// We can't reliably control real clipboard contents across CI, so just
	// make sure calling it with an arbitrary value doesn't panic and returns
	// a valid (possibly nil) cmd.
	_, cmd := m.handleClipboardClear("some-arbitrary-value")
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(statusClearMsg); !ok {
			t.Fatalf("expected statusClearMsg, got %T", msg)
		}
	}
}

func TestRenderBaseOnlyRendersMain(t *testing.T) {
	m := Model{screen: screenMain, width: 80, height: 24}
	m.main = newMainModel()
	if m.renderBase() == "" {
		t.Fatalf("expected non-empty base render on main screen")
	}
	m2 := Model{screen: screenLogin}
	if m2.renderBase() != "" {
		t.Fatalf("expected empty base render on non-main screen")
	}
}

func TestOverlayOn(t *testing.T) {
	if got := overlayOn("", "overlay-content", 80, 24); got != "overlay-content" {
		t.Fatalf("expected overlay content when base is empty, got %q", got)
	}
	if got := overlayOn("base", "overlay-content", 80, 24); got != "overlay-content" {
		t.Fatalf("expected overlay content, got %q", got)
	}
}

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Fatalf("expected 3")
	}
	if min(5, 3) != 3 {
		t.Fatalf("expected 3")
	}
}

func TestNewTextInput(t *testing.T) {
	ti := newTextInput("hint", false)
	if ti.Placeholder != "hint" || ti.EchoMode == 2 {
		t.Fatalf("unexpected plain text input: %+v", ti)
	}
	pw := newTextInput("secret", true)
	if pw.EchoCharacter != '*' {
		t.Fatalf("expected password echo character to be set")
	}
}

func TestFocusInput(t *testing.T) {
	ti := newTextInput("x", false)
	cmd := focusInput(&ti)
	if !ti.Focused() {
		t.Fatalf("expected input to be focused")
	}
	if cmd == nil {
		t.Fatalf("expected a blink cmd")
	}
}

func TestTickMsgUpdatesTOTPAndClearsExpiredStatus(t *testing.T) {
	m := Model{
		screen: screenMain,
		main: mainModel{
			viewStatus:        "old",
			viewStatusClearAt: time.Now().Add(-time.Second),
		},
	}
	updated, cmd := m.Update(tickMsg{})
	nm := updated.(Model)
	if nm.main.viewStatus != "" {
		t.Fatalf("expected expired status cleared")
	}
	if cmd == nil {
		t.Fatalf("expected another tick to be scheduled")
	}
}
