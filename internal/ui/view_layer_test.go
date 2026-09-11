package ui

import (
	"strings"
	"testing"
)

func TestRenderEntryViewSetsTitle(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		main: mainModel{
			showContent: true,
			currentEnt:  &Entry{Type: string(TypeNote), Title: "My Note"},
		},
	}
	got := renderEntryView(m)
	if !strings.Contains(got, "My Note") {
		t.Fatalf("expected title in view, got %q", got)
	}
}

func TestRenderEntryViewMasksCardNumber(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		main: mainModel{
			showContent: true,
			currentEnt:  &Entry{Type: string(TypeCard), Title: "Card", CardNumber: "1234567812345678", Expiry: "12/34", CVV: "123"},
		},
	}
	got := renderEntryView(m)
	if !strings.Contains(got, "**** **** **** 5678") {
		t.Fatalf("expected masked card number, got %q", got)
	}
}

func TestRenderEntryViewShowsNotes(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		main: mainModel{
			showContent: true,
			currentEnt:  &Entry{Type: string(TypeNote), Title: "Note", CustomText: "hello"},
		},
	}
	got := renderEntryView(m)
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected notes text, got %q", got)
	}
}

func TestRenderEntryViewShowsAttachments(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		main: mainModel{
			showContent: true,
			currentEnt: &Entry{
				Type:        string(TypeFile),
				Title:       "Files",
				Attachments: []Attachment{{ID: "1", FileName: "a.txt", Size: 10}},
			},
		},
	}
	got := renderEntryView(m)
	if !strings.Contains(got, "a.txt") {
		t.Fatalf("expected attachment in view, got %q", got)
	}
}
