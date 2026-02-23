package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveKeyDeterministic(t *testing.T) {
	p := KDFParams{
		Salt:     []byte("0123456789abcdef"),
		Time:     2,
		MemoryKB: 32 * 1024,
		Threads:  1,
	}
	k1 := DeriveKey("password", p)
	k2 := DeriveKey("password", p)
	if !bytes.Equal(k1, k2) {
		t.Fatalf("expected deterministic key derivation")
	}

	p.Salt = []byte("fedcba9876543210")
	k3 := DeriveKey("password", p)
	if bytes.Equal(k1, k3) {
		t.Fatalf("expected different key with different salt")
	}
}

func TestDeriveMasterKeyDeterministic(t *testing.T) {
	k1 := DeriveMasterKey("master")
	k2 := DeriveMasterKey("master")
	if !bytes.Equal(k1, k2) {
		t.Fatalf("expected deterministic master key derivation")
	}

	k3 := DeriveMasterKey("master2")
	if bytes.Equal(k1, k3) {
		t.Fatalf("expected different master key with different password")
	}
}

func TestSupportLegacy(t *testing.T) {
	if !SupportLegacy() {
		t.Fatalf("expected supportLegacy to be true by default")
	}
}

func TestDeriveLegacyMasterKeyMatchesDeriveMasterKey(t *testing.T) {
	k1 := DeriveMasterKey("testpass")
	k2 := DeriveLegacyMasterKey("testpass")
	if !bytes.Equal(k1, k2) {
		t.Fatalf("DeriveMasterKey and DeriveLegacyMasterKey should produce identical output")
	}
}

func TestGenerateRootSalt(t *testing.T) {
	s1, err := GenerateRootSalt()
	if err != nil {
		t.Fatalf("GenerateRootSalt error: %v", err)
	}
	if len(s1) != 32 {
		t.Fatalf("expected 32-byte salt, got %d", len(s1))
	}

	s2, err := GenerateRootSalt()
	if err != nil {
		t.Fatalf("GenerateRootSalt error: %v", err)
	}
	if bytes.Equal(s1, s2) {
		t.Fatalf("expected unique salts")
	}
}

func TestDeriveRootKeyDeterministic(t *testing.T) {
	salt := bytes.Repeat([]byte{0xAB}, 32)
	k1 := DeriveRootKey("password", salt)
	k2 := DeriveRootKey("password", salt)
	if !bytes.Equal(k1, k2) {
		t.Fatalf("expected deterministic root key derivation")
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}

	k3 := DeriveRootKey("password2", salt)
	if bytes.Equal(k1, k3) {
		t.Fatalf("expected different key for different password")
	}

	salt2 := bytes.Repeat([]byte{0xCD}, 32)
	k4 := DeriveRootKey("password", salt2)
	if bytes.Equal(k1, k4) {
		t.Fatalf("expected different key for different salt")
	}
}

func TestDeriveHKDFKey(t *testing.T) {
	rootKey := bytes.Repeat([]byte{0x42}, 32)

	mk, err := DeriveHKDFKey(rootKey, "master")
	if err != nil {
		t.Fatalf("DeriveHKDFKey master error: %v", err)
	}
	if len(mk) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(mk))
	}

	vk, err := DeriveHKDFKey(rootKey, "vault")
	if err != nil {
		t.Fatalf("DeriveHKDFKey vault error: %v", err)
	}
	if bytes.Equal(mk, vk) {
		t.Fatalf("expected different keys for different purposes")
	}

	// Deterministic.
	mk2, _ := DeriveHKDFKey(rootKey, "master")
	if !bytes.Equal(mk, mk2) {
		t.Fatalf("expected deterministic HKDF output")
	}
}

func TestDeriveKeys(t *testing.T) {
	salt := bytes.Repeat([]byte{0x99}, 32)
	mk, vk, err := DeriveKeys("password", salt)
	if err != nil {
		t.Fatalf("DeriveKeys error: %v", err)
	}
	if len(mk) != 32 || len(vk) != 32 {
		t.Fatalf("expected 32-byte keys")
	}
	if bytes.Equal(mk, vk) {
		t.Fatalf("master and vault keys should differ")
	}

	// Deterministic.
	mk2, vk2, _ := DeriveKeys("password", salt)
	if !bytes.Equal(mk, mk2) || !bytes.Equal(vk, vk2) {
		t.Fatalf("expected deterministic key derivation")
	}

	// Different password → different keys.
	mk3, _, _ := DeriveKeys("other", salt)
	if bytes.Equal(mk, mk3) {
		t.Fatalf("expected different keys for different password")
	}
}

func TestVaultHasEntries(t *testing.T) {
	dir := t.TempDir()
	if VaultHasEntries(dir) {
		t.Fatalf("expected no entries in empty dir")
	}

	loginsDir := filepath.Join(dir, "logins")
	if err := os.MkdirAll(loginsDir, 0700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if VaultHasEntries(dir) {
		t.Fatalf("expected no entries with empty logins dir")
	}

	if err := os.WriteFile(filepath.Join(loginsDir, "test.pb"), []byte("data"), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	if !VaultHasEntries(dir) {
		t.Fatalf("expected entries after adding .pb file")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x11}, 32)
	plaintext := []byte("hello world")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatalf("ciphertext should not equal plaintext")
	}

	got, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext mismatch")
	}
}

func TestDecryptRejectsShortCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x22}, 32)
	_, err := Decrypt(key, []byte{0x01, 0x02})
	if err == nil {
		t.Fatalf("expected error for short ciphertext")
	}
}

func TestEncryptAES256GCMKeyLength(t *testing.T) {
	_, err := EncryptAES256GCM([]byte("data"), []byte("short"))
	if err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestEncryptAES256GCMRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x33}, 32)
	plaintext := []byte("secret")

	ciphertext, err := EncryptAES256GCM(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAES256GCM error: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatalf("ciphertext should not equal plaintext")
	}

	got, err := DecryptAES256GCM(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptAES256GCM error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext mismatch")
	}
}

func TestDecryptAES256GCMRejectsBadKeyLength(t *testing.T) {
	key := []byte("short")
	_, err := DecryptAES256GCM([]byte("cipher"), key)
	if err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestDecryptAES256GCMRejectsShortCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x44}, 32)
	_, err := DecryptAES256GCM([]byte{0x01, 0x02}, key)
	if err == nil {
		t.Fatalf("expected error for short ciphertext")
	}
}

func TestEnsureKDFSecretPersists(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x55}, 32)

	p1, err := EnsureKDFSecret(dir, key)
	if err != nil {
		t.Fatalf("EnsureKDFSecret error: %v", err)
	}
	if len(p1.Salt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d", len(p1.Salt))
	}

	p2, err := EnsureKDFSecret(dir, key)
	if err != nil {
		t.Fatalf("EnsureKDFSecret second call error: %v", err)
	}
	if !bytes.Equal(p1.Salt, p2.Salt) || p1.Time != p2.Time || p1.MemoryKB != p2.MemoryKB || p1.Threads != p2.Threads {
		t.Fatalf("expected stable KDF params across calls")
	}

	secretFile := filepath.Join(dir, ".secret")
	if _, err := os.Stat(secretFile); err != nil {
		t.Fatalf("expected secret file to exist: %v", err)
	}
}

func TestDeriveKeyLength(t *testing.T) {
	p := KDFParams{
		Salt:     []byte("0123456789abcdef"),
		Time:     2,
		MemoryKB: 32 * 1024,
		Threads:  1,
	}
	k := DeriveKey("password", p)
	if len(k) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k))
	}
}

func TestDeriveMasterKeyLength(t *testing.T) {
	k := DeriveMasterKey("master")
	if len(k) != 32 {
		t.Fatalf("expected 32-byte master key, got %d", len(k))
	}
}

func TestEncryptRejectsBadKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("short"), []byte("data"))
	if err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestDecryptRejectsBadKeyLength(t *testing.T) {
	_, err := Decrypt([]byte("short"), []byte("cipher"))
	if err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestEnsureKDFSecretRejectsInvalidSecretFile(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x66}, 32)

	if err := os.WriteFile(filepath.Join(dir, ".secret"), []byte("not-json"), 0600); err != nil {
		t.Fatalf("setup error: %v", err)
	}

	_, err := loadKDFSecret(dir, key)
	if err == nil {
		t.Fatalf("expected loadKDFSecret to fail on invalid data")
	}
}

func TestWriteKDFSecretAtomicCreatesValidSecret(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x77}, 32)

	p := KDFParams{Salt: bytes.Repeat([]byte{0x01}, 16), Time: 1, MemoryKB: 64, Threads: 1}
	if err := writeKDFSecretAtomic(dir, p, key); err != nil {
		t.Fatalf("writeKDFSecretAtomic error: %v", err)
	}

	got, err := loadKDFSecret(dir, key)
	if err != nil {
		t.Fatalf("loadKDFSecret error: %v", err)
	}
	if len(got.Salt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d", len(got.Salt))
	}
}

func TestEnsureKDFParamsDefaults(t *testing.T) {
	p := KDFParams{}
	ensureKDFParams(&p)
	if p.Time == 0 || p.MemoryKB == 0 || p.Threads == 0 {
		t.Fatalf("expected defaults to be applied")
	}
}

func TestDecryptFailsWithWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x10}, 32)
	wrongKey := bytes.Repeat([]byte{0x11}, 32)
	plaintext := []byte("message")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	if _, err := Decrypt(wrongKey, ciphertext); err == nil {
		t.Fatalf("expected error when decrypting with wrong key")
	}
}

func TestDecryptFailsWithTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x12}, 32)
	plaintext := []byte("message")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xFF
	if _, err := Decrypt(key, ciphertext); err == nil {
		t.Fatalf("expected error when decrypting tampered ciphertext")
	}
}

func TestDecryptAES256GCMFailsWithWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x21}, 32)
	wrongKey := bytes.Repeat([]byte{0x22}, 32)
	plaintext := []byte("secret")

	ciphertext, err := EncryptAES256GCM(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAES256GCM error: %v", err)
	}
	if _, err := DecryptAES256GCM(ciphertext, wrongKey); err == nil {
		t.Fatalf("expected error when decrypting with wrong key")
	}
}

func TestDecryptAES256GCMFailsWithTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x23}, 32)
	plaintext := []byte("secret")

	ciphertext, err := EncryptAES256GCM(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAES256GCM error: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xAA
	if _, err := DecryptAES256GCM(ciphertext, key); err == nil {
		t.Fatalf("expected error when decrypting tampered ciphertext")
	}
}

func TestReKeyVaultWritesNewSecret(t *testing.T) {
	dir := t.TempDir()
	oldMasterKey := bytes.Repeat([]byte{0xAA}, 32)
	newMasterKey := bytes.Repeat([]byte{0xBB}, 32)

	// Create initial secret.
	_, err := EnsureKDFSecret(dir, oldMasterKey)
	if err != nil {
		t.Fatalf("EnsureKDFSecret error: %v", err)
	}

	// Re-key vault (writes new .secret).
	if err := ReKeyVault(dir, newMasterKey); err != nil {
		t.Fatalf("ReKeyVault error: %v", err)
	}

	// Old master key should no longer work.
	if _, err := loadKDFSecret(dir, oldMasterKey); err == nil {
		t.Fatalf("expected old master key to fail after re-key")
	}

	// New master key should work.
	p, err := loadKDFSecret(dir, newMasterKey)
	if err != nil {
		t.Fatalf("expected new master key to work: %v", err)
	}
	if len(p.Salt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d", len(p.Salt))
	}
}

func TestReKeyEntriesReEncryptsFiles(t *testing.T) {
	dir := t.TempDir()
	oldKey := bytes.Repeat([]byte{0xCC}, 32)
	newKey := bytes.Repeat([]byte{0xDD}, 32)

	// Create a category directory with an encrypted .pb file.
	loginsDir := filepath.Join(dir, "logins")
	if err := os.MkdirAll(loginsDir, 0700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	plaintext := []byte("test entry data")
	enc, err := Encrypt(oldKey, plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	entryPath := filepath.Join(loginsDir, "test.pb")
	if err := os.WriteFile(entryPath, enc, 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Create an attachment directory with an encrypted file.
	attDir := filepath.Join(dir, "_attachments")
	if err := os.MkdirAll(attDir, 0700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	attPlain := []byte("attachment data")
	attEnc, err := Encrypt(oldKey, attPlain)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	attPath := filepath.Join(attDir, "att1")
	if err := os.WriteFile(attPath, attEnc, 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Re-key entries.
	if err := ReKeyEntries(dir, oldKey, newKey); err != nil {
		t.Fatalf("ReKeyEntries error: %v", err)
	}

	// Old key should no longer decrypt the entry.
	data, _ := os.ReadFile(entryPath)
	if _, err := Decrypt(oldKey, data); err == nil {
		t.Fatalf("expected old key to fail after re-key")
	}

	// New key should decrypt the entry.
	got, err := Decrypt(newKey, data)
	if err != nil {
		t.Fatalf("Decrypt with new key error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext mismatch after re-key")
	}

	// New key should decrypt the attachment.
	attData, _ := os.ReadFile(attPath)
	gotAtt, err := Decrypt(newKey, attData)
	if err != nil {
		t.Fatalf("Decrypt attachment with new key error: %v", err)
	}
	if !bytes.Equal(gotAtt, attPlain) {
		t.Fatalf("attachment plaintext mismatch after re-key")
	}
}

func TestReKeyEntriesSkipsMissingDirs(t *testing.T) {
	dir := t.TempDir()
	oldKey := bytes.Repeat([]byte{0xEE}, 32)
	newKey := bytes.Repeat([]byte{0xFF}, 32)

	// Should succeed even with no category directories.
	if err := ReKeyEntries(dir, oldKey, newKey); err != nil {
		t.Fatalf("ReKeyEntries error on empty vault: %v", err)
	}
}
