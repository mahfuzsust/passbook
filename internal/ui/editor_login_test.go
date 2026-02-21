package ui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestCollectLoginFields(t *testing.T) {
	resetEditorTestState()
	ent := &Entry{Type: string(TypeLogin)}
	addLoginFields(ent)

	uiEditorForm.GetFormItemByLabel("Username").(*tview.InputField).SetText("user")
	uiEditorPasswordField.SetText("newpass")
	uiEditorForm.GetFormItemByLabel("Link").(*tview.InputField).SetText("http://example.com")
	uiEditorForm.GetFormItemByLabel("TOTP Secret").(*tview.InputField).SetText("ABC123")

	collectLoginFields(ent, "oldpass")
	if ent.Username != "user" || ent.Password != "newpass" || ent.Link != "http://example.com" || ent.TotpSecret != "ABC123" {
		t.Fatalf("unexpected values in entry after collect")
	}
	if len(ent.History) != 1 || ent.History[0].Password != "oldpass" {
		t.Fatalf("expected password history to be appended")
	}
}

func TestCollectLoginFieldsNoHistoryWhenSame(t *testing.T) {
	resetEditorTestState()
	ent := &Entry{Type: string(TypeLogin)}
	addLoginFields(ent)

	uiEditorForm.GetFormItemByLabel("Username").(*tview.InputField).SetText("user")
	uiEditorPasswordField.SetText("same")
	uiEditorForm.GetFormItemByLabel("Link").(*tview.InputField).SetText("http://example.com")
	uiEditorForm.GetFormItemByLabel("TOTP Secret").(*tview.InputField).SetText("ABC123")

	collectLoginFields(ent, "same")
	if len(ent.History) != 0 {
		t.Fatalf("did not expect password history when unchanged")
	}
}
