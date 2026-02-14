package crypto

import "testing"

func TestDeriveKey_Argon2Deterministic(t *testing.T) {
	p := KDFParams{
		Salt:     []byte("0123456789abcdef"),
		Time:     1,
		MemoryKB: 32 * 1024,
		Threads:  1,
	}

	k1 := DeriveKey("password", p)
	k2 := DeriveKey("password", p)
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatalf("argon2 key not deterministic")
		}
	}
}
