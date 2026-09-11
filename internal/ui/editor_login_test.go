package ui

import "testing"

func TestCollectLoginFields(t *testing.T) {
	ent := &Entry{Type: string(TypeLogin)}
	e := editorModel{entryType: TypeLogin}
	e.username.SetValue("user")
	e.password.SetValue("newpass")
	e.link.SetValue("http://example.com")
	e.totpSecret.SetValue("ABC123")

	collectLoginFields(ent, e, "oldpass")
	if ent.Username != "user" || ent.Password != "newpass" || ent.Link != "http://example.com" || ent.TotpSecret != "ABC123" {
		t.Fatalf("unexpected values in entry after collect")
	}
	if len(ent.History) != 1 || ent.History[0].Password != "oldpass" {
		t.Fatalf("expected password history to be appended")
	}
}

func TestCollectLoginFieldsNoHistoryWhenSame(t *testing.T) {
	ent := &Entry{Type: string(TypeLogin)}
	e := editorModel{entryType: TypeLogin}
	e.username.SetValue("user")
	e.password.SetValue("same")
	e.link.SetValue("http://example.com")
	e.totpSecret.SetValue("ABC123")

	collectLoginFields(ent, e, "same")
	if len(ent.History) != 0 {
		t.Fatalf("did not expect password history when unchanged")
	}
}
