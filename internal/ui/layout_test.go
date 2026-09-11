package ui

import (
	"strings"
	"testing"
)

func TestSplitView(t *testing.T) {
	out := splitView("LEFT", "RIGHT", 80, 24, 0.3, 10, 20)
	if !strings.Contains(out, "LEFT") || !strings.Contains(out, "RIGHT") {
		t.Fatalf("expected both panes rendered, got %q", out)
	}
}

func TestSplitViewRespectsMinRight(t *testing.T) {
	out := splitView("L", "R", 30, 10, 0.9, 5, 20)
	if out == "" {
		t.Fatalf("expected non-empty output even with tight constraints")
	}
}

func TestPadHeight(t *testing.T) {
	got := padHeight("a\nb", 5)
	if len(strings.Split(got, "\n")) != 5 {
		t.Fatalf("expected 5 lines, got %d: %q", len(strings.Split(got, "\n")), got)
	}
}

func TestPadHeightNoOpWhenTallEnough(t *testing.T) {
	content := "a\nb\nc"
	got := padHeight(content, 2)
	if got != content {
		t.Fatalf("expected content unchanged when already tall enough, got %q", got)
	}
}

func TestCenterModal(t *testing.T) {
	out := centerModal("hello", 80, 24, 10, 5, 40, 15, 0.5, 0.5)
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected content to be present in modal, got %q", out)
	}
}
