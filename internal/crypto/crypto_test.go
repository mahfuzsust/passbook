package crypto

import (
	"bytes"
	"testing"
)

func TestWipeBytes(t *testing.T) {
	key := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	WipeBytes(key)
	for i, b := range key {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: got %x", i, b)
		}
	}
}

func TestWipeBytesNil(t *testing.T) {
	WipeBytes(nil)
	WipeBytes([]byte{})
}

func TestGeneratePinKey(t *testing.T) {
	k1, err := GeneratePinKey()
	if err != nil {
		t.Fatalf("GeneratePinKey error: %v", err)
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}

	k2, err := GeneratePinKey()
	if err != nil {
		t.Fatalf("GeneratePinKey error: %v", err)
	}
	if bytes.Equal(k1, k2) {
		t.Fatalf("expected unique keys")
	}
}

func TestComputePinTagDeterministic(t *testing.T) {
	key := bytes.Repeat([]byte{0xAA}, 32)
	tag1 := ComputePinTag(key, "123456")
	tag2 := ComputePinTag(key, "123456")
	if tag1 != tag2 {
		t.Fatalf("expected deterministic pin tag")
	}
	if len(tag1) != 64 {
		t.Fatalf("expected 64-char hex tag, got %d", len(tag1))
	}
}

func TestComputePinTagDiffersForDiffPins(t *testing.T) {
	key := bytes.Repeat([]byte{0xAA}, 32)
	tag1 := ComputePinTag(key, "123456")
	tag2 := ComputePinTag(key, "654321")
	if tag1 == tag2 {
		t.Fatalf("expected different tags for different PINs")
	}
}

func TestVerifyPinTagCorrect(t *testing.T) {
	key := bytes.Repeat([]byte{0xBB}, 32)
	pin := "123456"
	tag := ComputePinTag(key, pin)
	if !VerifyPinTag(key, pin, tag) {
		t.Fatalf("expected pin tag to verify")
	}
}

func TestVerifyPinTagWrongPin(t *testing.T) {
	key := bytes.Repeat([]byte{0xBB}, 32)
	tag := ComputePinTag(key, "123456")
	if VerifyPinTag(key, "654321", tag) {
		t.Fatalf("expected pin tag verification to fail with wrong pin")
	}
}

func TestVerifyPinTagWrongKey(t *testing.T) {
	key1 := bytes.Repeat([]byte{0xBB}, 32)
	key2 := bytes.Repeat([]byte{0xCC}, 32)
	pin := "123456"
	tag := ComputePinTag(key1, pin)
	if VerifyPinTag(key2, pin, tag) {
		t.Fatalf("expected pin tag verification to fail with wrong key")
	}
}
