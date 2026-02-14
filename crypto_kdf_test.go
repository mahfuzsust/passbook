package main

import "testing"

func TestDeriveKey_Argon2Deterministic(t *testing.T) {
	kdfSalt = []byte("0123456789abcdef")
	kdfTime = 1
	kdfMemoryKB = 32 * 1024
	kdfThreads = 1

	k1 := deriveKey("password")
	k2 := deriveKey("password")
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatalf("argon2 key not deterministic")
		}
	}
}
