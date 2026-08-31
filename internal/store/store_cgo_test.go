package store

import "testing"

func TestIsCGODisabled(t *testing.T) {
	if IsCGODisabled(nil) {
		t.Fatal("nil error should not be CGO disabled")
	}
	stub := errString("Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub")
	if !IsCGODisabled(stub) {
		t.Fatal("expected CGO stub error to be detected")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
