package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttachmentEncryptDecryptRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	dataDir = tmp
	if err := os.MkdirAll(getAttachmentDir(), 0700); err != nil {
		t.Fatalf("mkdir attachments: %v", err)
	}

	kdfSalt = []byte("0123456789abcdef")
	kdfTime = 1
	kdfMemoryKB = 32 * 1024
	kdfThreads = 1
	masterKey = deriveKey("pw")

	plaintext := []byte("hello attachment")
	enc, err := encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	id := "att1"
	if err := os.WriteFile(filepath.Join(getAttachmentDir(), id), enc, 0600); err != nil {
		t.Fatalf("write encrypted: %v", err)
	}

	readEnc, err := os.ReadFile(filepath.Join(getAttachmentDir(), id))
	if err != nil {
		t.Fatalf("read encrypted: %v", err)
	}
	dec, err := decrypt(readEnc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(dec) != string(plaintext) {
		t.Fatalf("want %q got %q", plaintext, dec)
	}
}
