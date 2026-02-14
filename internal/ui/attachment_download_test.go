package ui

import (
	"os"
	"path/filepath"
	"testing"

	"passbook/internal/config"
)

func TestAttachmentEncryptDecryptRoundTrip(t *testing.T) {
	uiCfg = config.AppConfig{DataDir: t.TempDir()}
	uiDataDir = uiCfg.DataDir
	if err := os.MkdirAll(getAttachmentDir(), 0700); err != nil {
		t.Fatal(err)
	}
	uiKDF.Salt = make([]byte, 16)
	uiKDF.Time = 1
	uiKDF.MemoryKB = 32
	uiKDF.Threads = 1

	uiMasterKey = deriveKey("pw")

	plaintext := []byte("hello")
	enc, err := encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	id := "att1"
	if err := os.WriteFile(filepath.Join(getAttachmentDir(), id), enc, 0600); err != nil {
		t.Fatal(err)
	}
	readEnc, err := os.ReadFile(filepath.Join(getAttachmentDir(), id))
	if err != nil {
		t.Fatal(err)
	}
	dec, err := decrypt(readEnc)
	if err != nil {
		t.Fatal(err)
	}
	if string(dec) != string(plaintext) {
		t.Fatalf("got %q, want %q", string(dec), string(plaintext))
	}
}
