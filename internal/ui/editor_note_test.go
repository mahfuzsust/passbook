package ui

import "testing"

func TestNewNoteEntry(t *testing.T) {
	ent := NewEntry(TypeNote)
	if ent.Type != string(TypeNote) {
		t.Fatalf("expected note type")
	}
}
