package ui

// addNoteFields adds note-specific form fields to the editor.
// Notes only use the shared Notes textarea, so no extra fields are needed.
func addNoteFields(ent *Entry) {
	ent.Attachments = nil
}

// collectNoteFields reads note form values into the entry (no-op for notes).
func collectNoteFields(_ *Entry) {}

// renderNoteView renders the note-type view pane content.
// Notes have no type-specific rows beyond the shared notes/attachments section.
func renderNoteView() {
	uiCurrentEnt.Attachments = nil
}
