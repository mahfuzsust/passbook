package ui

import "testing"

func TestAddNoteFieldsClearsAttachments(t *testing.T) {
	resetEditorTestState()
	ent := &Entry{Type: string(TypeNote), Attachments: []*Attachment{{Id: "1"}}}
	addNoteFields(ent)
	if ent.Attachments != nil {
		t.Fatalf("expected attachments to be cleared")
	}
}
