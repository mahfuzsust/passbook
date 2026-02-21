package ui

import "testing"

func TestAddFileFieldsAddsFormItems(t *testing.T) {
	resetEditorTestState()
	ent := &Entry{Type: string(TypeFile)}
	before := uiEditorForm.GetFormItemCount()
	addFileFields(ent)
	after := uiEditorForm.GetFormItemCount()
	if after <= before {
		t.Fatalf("expected form items to be added")
	}
}
