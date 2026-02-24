package crypto

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	// Should not panic on nil or empty slices.
	WipeBytes(nil)
	WipeBytes([]byte{})
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

func TestSaveAndLoadVaultParams(t *testing.T) {
	dir := t.TempDir()
	p := VaultParams{
		Version:  1,
		Salt:     bytes.Repeat([]byte{0xBB}, 32),
		Time:     8,
		MemoryKB: 512 * 1024,
		Threads:  8,
		KDF:      "argon2id",
		Cipher:   "aes-256-gcm",
	}

	if err := SaveVaultParams(dir, p); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}

	loaded, err := LoadVaultParams(dir)
	if err != nil {
		t.Fatalf("LoadVaultParams error: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected non-nil params")
	}
	if !bytes.Equal(p.Salt, loaded.Salt) {
		t.Fatalf("salt mismatch")
	}
	if loaded.Time != 8 || loaded.MemoryKB != 512*1024 || loaded.Threads != 8 {
		t.Fatalf("params mismatch: got time=%d memory=%d threads=%d", loaded.Time, loaded.MemoryKB, loaded.Threads)
	}
	if loaded.Version != 1 || loaded.KDF != "argon2id" || loaded.Cipher != "aes-256-gcm" {
		t.Fatalf("metadata mismatch: version=%d kdf=%s cipher=%s", loaded.Version, loaded.KDF, loaded.Cipher)
	}
}

func TestLoadVaultParamsBackfillsZeroes(t *testing.T) {
	dir := t.TempDir()
	salt := bytes.Repeat([]byte{0xCC}, 32)
	p := VaultParams{Salt: salt}
	if err := SaveVaultParams(dir, p); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}

	loaded, err := LoadVaultParams(dir)
	if err != nil {
		t.Fatalf("LoadVaultParams error: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected non-nil params")
	}
	if loaded.Time != RecommendedTime || loaded.MemoryKB != RecommendedMemory || loaded.Threads != RecommendedThreads {
		t.Fatalf("expected zeroed params to be back-filled with recommended values")
	}
	if loaded.Version != 1 || loaded.KDF != "argon2id" || loaded.Cipher != "aes-256-gcm" {
		t.Fatalf("expected defaults to be back-filled")
	}
}

func TestLoadVaultParamsMissing(t *testing.T) {
	dir := t.TempDir()
	loaded, err := LoadVaultParams(dir)
	if err != nil {
		t.Fatalf("expected no error for missing vault params, got: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil params for missing file")
	}
}

func TestNeedsRehash(t *testing.T) {
	current := VaultParams{
		Salt:     bytes.Repeat([]byte{0x01}, 32),
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
	}
	if current.NeedsRehash() {
		t.Fatalf("should not need rehash at recommended values")
	}

	weaker := current
	weaker.Time = RecommendedTime - 1
	if !weaker.NeedsRehash() {
		t.Fatalf("should need rehash when time is below recommended")
	}

	weaker = current
	weaker.MemoryKB = RecommendedMemory - 1
	if !weaker.NeedsRehash() {
		t.Fatalf("should need rehash when memory is below recommended")
	}

	weaker = current
	weaker.Threads = RecommendedThreads - 1
	if !weaker.NeedsRehash() {
		t.Fatalf("should need rehash when threads is below recommended")
	}

	stronger := current
	stronger.Time = RecommendedTime + 2
	if stronger.NeedsRehash() {
		t.Fatalf("should not need rehash when params are stronger")
	}
}

func TestDefaultVaultParams(t *testing.T) {
	p, err := DefaultVaultParams()
	if err != nil {
		t.Fatalf("DefaultVaultParams error: %v", err)
	}
	if len(p.Salt) != 32 {
		t.Fatalf("expected 32-byte salt, got %d", len(p.Salt))
	}
	if p.Time != RecommendedTime || p.MemoryKB != RecommendedMemory || p.Threads != RecommendedThreads {
		t.Fatalf("expected recommended params")
	}
	if p.Version != 1 {
		t.Fatalf("expected version 1, got %d", p.Version)
	}
	if p.KDF != "argon2id" || p.Cipher != "aes-256-gcm" {
		t.Fatalf("expected kdf=argon2id cipher=aes-256-gcm, got kdf=%s cipher=%s", p.KDF, p.Cipher)
	}
	if p.MasterKeyPurpose != "passbook:master:v1" {
		t.Fatalf("expected MasterKeyPurpose=passbook:master:v1, got %s", p.MasterKeyPurpose)
	}
	if p.VaultKeyPurpose != "passbook:vault:v1" {
		t.Fatalf("expected VaultKeyPurpose=passbook:vault:v1, got %s", p.VaultKeyPurpose)
	}
}

func TestHashVaultParams(t *testing.T) {
	p := VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0x11}, 32),
		Time: 6, MemoryKB: 256 * 1024, Threads: 4,
		KDF: "argon2id", Cipher: "aes-256-gcm",
	}
	h1, err := HashVaultParams(p)
	if err != nil {
		t.Fatalf("HashVaultParams error: %v", err)
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d", len(h1))
	}

	h2, _ := HashVaultParams(p)
	if h1 != h2 {
		t.Fatalf("expected deterministic hash")
	}

	p.Time = 8
	h3, _ := HashVaultParams(p)
	if h1 == h3 {
		t.Fatalf("expected different hash for different params")
	}
}

func TestHashMatchesDiskBytes(t *testing.T) {
	dir := t.TempDir()
	p := VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0x44}, 32),
		Time: 6, MemoryKB: 256 * 1024, Threads: 4,
		KDF: "argon2id", Cipher: "aes-256-gcm",
	}

	if err := SaveVaultParams(dir, p); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}

	diskBytes, err := os.ReadFile(filepath.Join(dir, ".vault_params"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	structHash, _ := HashVaultParams(p)
	diskHash := HashVaultParamsBytes(diskBytes)
	if structHash != diskHash {
		t.Fatalf("hash mismatch: struct=%s disk=%s", structHash, diskHash)
	}
}

// newTestVaultParams returns valid VaultParams for use in tests.
func newTestVaultParams() VaultParams {
	return VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0x22}, 32),
		Time: 6, MemoryKB: 256 * 1024, Threads: 4,
		KDF: "argon2id", Cipher: "aes-256-gcm",
		MasterKeyPurpose: "passbook:master:v1",
		VaultKeyPurpose:  "passbook:vault:v1",
	}
}

// setupTestSecret creates .vault_params and .secret in dir.
func setupTestSecret(t *testing.T, dir string, key []byte, vp VaultParams) {
	t.Helper()
	if err := SaveVaultParams(dir, vp); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}
	if _, err := EnsureSecret(dir, key, vp); err != nil {
		t.Fatalf("EnsureSecret error: %v", err)
	}
}

func TestEnsureSecretStoresHash(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x55}, 32)
	vp := newTestVaultParams()

	setupTestSecret(t, dir, key, vp)

	if err := VerifyVaultParamsHash(dir, key); err != nil {
		t.Fatalf("VerifyVaultParamsHash should pass: %v", err)
	}

	tampered := vp
	tampered.Time = 999
	if err := SaveVaultParams(dir, tampered); err != nil {
		t.Fatalf("SaveVaultParams (tampered) error: %v", err)
	}
	if err := VerifyVaultParamsHash(dir, key); err == nil {
		t.Fatalf("expected error for tampered vault params file")
	}
}

func TestSecretAADRejectsTamperedVaultParams(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x88}, 32)
	vp := newTestVaultParams()

	setupTestSecret(t, dir, key, vp)

	if _, err := loadKDFSecret(dir, key); err != nil {
		t.Fatalf("loadKDFSecret should succeed: %v", err)
	}

	tampered := vp
	tampered.Time = 999
	if err := SaveVaultParams(dir, tampered); err != nil {
		t.Fatalf("SaveVaultParams (tampered) error: %v", err)
	}

	// AAD mismatch → GCM auth fails.
	if _, err := loadKDFSecret(dir, key); err == nil {
		t.Fatalf("expected loadKDFSecret to fail with tampered vault params")
	}

	// Direct GCM-level test.
	hash, _ := HashVaultParams(vp)
	secretBytes, _ := os.ReadFile(filepath.Join(dir, ".secret"))
	wrongAAD := []byte(HashVaultParamsBytes([]byte(`{"tampered":true}`)))
	correctAAD := []byte(hash)
	if _, err := DecryptAES256GCM(secretBytes, key, wrongAAD); err == nil {
		t.Fatalf("expected GCM auth failure with wrong AAD")
	}
	if _, err := DecryptAES256GCM(secretBytes, key, correctAAD); err != nil {
		t.Fatalf("expected GCM success with correct AAD: %v", err)
	}
}

func TestDeriveRootKeyDeterministic(t *testing.T) {
	p := VaultParams{
		Salt:     bytes.Repeat([]byte{0xAB}, 32),
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
	}
	k1 := DeriveRootKey("password", p)
	k2 := DeriveRootKey("password", p)
	if !bytes.Equal(k1, k2) {
		t.Fatalf("expected deterministic root key derivation")
	}
	if len(k1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(k1))
	}

	k3 := DeriveRootKey("password2", p)
	if bytes.Equal(k1, k3) {
		t.Fatalf("expected different key for different password")
	}

	p2 := VaultParams{
		Salt:     bytes.Repeat([]byte{0xCD}, 32),
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
	}
	k4 := DeriveRootKey("password", p2)
	if bytes.Equal(k1, k4) {
		t.Fatalf("expected different key for different salt")
	}
}

func TestDeriveHKDFKey(t *testing.T) {
	rootKey := bytes.Repeat([]byte{0x42}, 32)

	mk, err := DeriveHKDFKey(rootKey, masterKeyPurpose)
	if err != nil {
		t.Fatalf("DeriveHKDFKey master error: %v", err)
	}
	if len(mk) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(mk))
	}

	vk, err := DeriveHKDFKey(rootKey, vaultKeyPurpose)
	if err != nil {
		t.Fatalf("DeriveHKDFKey vault error: %v", err)
	}
	if bytes.Equal(mk, vk) {
		t.Fatalf("expected different keys for different purposes")
	}

	mk2, _ := DeriveHKDFKey(rootKey, masterKeyPurpose)
	if !bytes.Equal(mk, mk2) {
		t.Fatalf("expected deterministic HKDF output")
	}
}

func TestDeriveKeys(t *testing.T) {
	p := VaultParams{
		Salt:             bytes.Repeat([]byte{0x99}, 32),
		Time:             RecommendedTime,
		MemoryKB:         RecommendedMemory,
		Threads:          RecommendedThreads,
		MasterKeyPurpose: "passbook:master:v1",
		VaultKeyPurpose:  "passbook:vault:v1",
	}
	mk, vk, err := DeriveKeys("password", p)
	if err != nil {
		t.Fatalf("DeriveKeys error: %v", err)
	}
	if len(mk) != 32 || len(vk) != 32 {
		t.Fatalf("expected 32-byte keys")
	}
	if bytes.Equal(mk, vk) {
		t.Fatalf("master and vault keys should differ")
	}

	mk2, vk2, _ := DeriveKeys("password", p)
	if !bytes.Equal(mk, mk2) || !bytes.Equal(vk, vk2) {
		t.Fatalf("expected deterministic key derivation")
	}

	mk3, _, _ := DeriveKeys("other", p)
	if bytes.Equal(mk, mk3) {
		t.Fatalf("expected different keys for different password")
	}
}

func TestDeriveKeysRejectsEmptyPurpose(t *testing.T) {
	p := VaultParams{
		Salt:     bytes.Repeat([]byte{0x99}, 32),
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
	}
	_, _, err := DeriveKeys("password", p)
	if err == nil {
		t.Fatalf("expected error for empty purpose strings")
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
	_, err := EncryptAES256GCM([]byte("data"), []byte("short"), nil)
	if err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestEncryptAES256GCMRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x33}, 32)
	plaintext := []byte("secret")

	ciphertext, err := EncryptAES256GCM(plaintext, key, nil)
	if err != nil {
		t.Fatalf("EncryptAES256GCM error: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatalf("ciphertext should not equal plaintext")
	}

	got, err := DecryptAES256GCM(ciphertext, key, nil)
	if err != nil {
		t.Fatalf("DecryptAES256GCM error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext mismatch")
	}
}

func TestDecryptAES256GCMRejectsBadKeyLength(t *testing.T) {
	key := []byte("short")
	_, err := DecryptAES256GCM([]byte("cipher"), key, nil)
	if err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestEncryptAES256GCMWithAAD(t *testing.T) {
	key := bytes.Repeat([]byte{0x77}, 32)
	plaintext := []byte("authenticated data test")
	aad := []byte("vault-params-hash-hex")

	ciphertext, err := EncryptAES256GCM(plaintext, key, aad)
	if err != nil {
		t.Fatalf("EncryptAES256GCM with AAD error: %v", err)
	}

	got, err := DecryptAES256GCM(ciphertext, key, aad)
	if err != nil {
		t.Fatalf("DecryptAES256GCM with AAD error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext mismatch")
	}

	if _, err := DecryptAES256GCM(ciphertext, key, []byte("wrong-aad")); err == nil {
		t.Fatalf("expected error when decrypting with wrong AAD")
	}

	if _, err := DecryptAES256GCM(ciphertext, key, nil); err == nil {
		t.Fatalf("expected error when decrypting with nil AAD (was encrypted with AAD)")
	}
}

func TestDecryptAES256GCMRejectsShortCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x44}, 32)
	_, err := DecryptAES256GCM([]byte{0x01, 0x02}, key, nil)
	if err == nil {
		t.Fatalf("expected error for short ciphertext")
	}
}

func TestEnsureSecretPersists(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x55}, 32)
	vp := newTestVaultParams()

	setupTestSecret(t, dir, key, vp)

	p1, err := loadKDFSecret(dir, key)
	if err != nil {
		t.Fatalf("loadKDFSecret error: %v", err)
	}
	if len(p1.Salt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d", len(p1.Salt))
	}

	p2, err := loadKDFSecret(dir, key)
	if err != nil {
		t.Fatalf("loadKDFSecret second call error: %v", err)
	}
	if !bytes.Equal(p1.Salt, p2.Salt) || p1.Time != p2.Time || p1.MemoryKB != p2.MemoryKB || p1.Threads != p2.Threads {
		t.Fatalf("expected stable KDF params across calls")
	}

	sf := filepath.Join(dir, ".secret")
	if _, err := os.Stat(sf); err != nil {
		t.Fatalf("expected secret file to exist: %v", err)
	}
}

func TestWriteKDFSecretAtomicCreatesValidSecret(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x77}, 32)
	vp := newTestVaultParams()

	if err := SaveVaultParams(dir, vp); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}
	hash, _ := HashVaultParams(vp)

	p := KDFParams{Salt: bytes.Repeat([]byte{0x01}, 16), Time: 1, MemoryKB: 64, Threads: 1}
	if err := writeKDFSecretAtomic(dir, p, key, hash); err != nil {
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

func TestLoadKDFSecretRejectsInvalidSecretFile(t *testing.T) {
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

	ciphertext, err := EncryptAES256GCM(plaintext, key, nil)
	if err != nil {
		t.Fatalf("EncryptAES256GCM error: %v", err)
	}
	if _, err := DecryptAES256GCM(ciphertext, wrongKey, nil); err == nil {
		t.Fatalf("expected error when decrypting with wrong key")
	}
}

func TestDecryptAES256GCMFailsWithTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x23}, 32)
	plaintext := []byte("secret")

	ciphertext, err := EncryptAES256GCM(plaintext, key, nil)
	if err != nil {
		t.Fatalf("EncryptAES256GCM error: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xAA
	if _, err := DecryptAES256GCM(ciphertext, key, nil); err == nil {
		t.Fatalf("expected error when decrypting tampered ciphertext")
	}
}

func TestReKeyVaultWritesNewSecret(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0xAA}, 32)
	newKey := bytes.Repeat([]byte{0xBB}, 32)
	vp := newTestVaultParams()

	setupTestSecret(t, dir, key, vp)

	if err := ReKeyVault(dir, newKey); err != nil {
		t.Fatalf("ReKeyVault error: %v", err)
	}

	if _, err := loadKDFSecret(dir, key); err == nil {
		t.Fatalf("expected old master key to fail after re-key")
	}

	p, err := loadKDFSecret(dir, newKey)
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

	if err := ReKeyEntries(dir, oldKey, newKey); err != nil {
		t.Fatalf("ReKeyEntries error: %v", err)
	}

	data, _ := os.ReadFile(entryPath)
	if _, err := Decrypt(oldKey, data); err == nil {
		t.Fatalf("expected old key to fail after re-key")
	}

	got, err := Decrypt(newKey, data)
	if err != nil {
		t.Fatalf("Decrypt with new key error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext mismatch after re-key")
	}

	attData, err := os.ReadFile(attPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
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

	if err := ReKeyEntries(dir, oldKey, newKey); err != nil {
		t.Fatalf("ReKeyEntries error on empty vault: %v", err)
	}
}

func TestGenerateCommitNonce(t *testing.T) {
	n1, err := GenerateCommitNonce()
	if err != nil {
		t.Fatalf("GenerateCommitNonce error: %v", err)
	}
	if len(n1) != 32 {
		t.Fatalf("expected 32-byte nonce, got %d", len(n1))
	}

	n2, err := GenerateCommitNonce()
	if err != nil {
		t.Fatalf("GenerateCommitNonce error: %v", err)
	}
	if bytes.Equal(n1, n2) {
		t.Fatalf("expected unique nonces")
	}
}

func TestComputeCommitTagDeterministic(t *testing.T) {
	key := bytes.Repeat([]byte{0xAA}, 32)
	nonce := bytes.Repeat([]byte{0x01}, 32)
	tag1 := ComputeCommitTag(key, nonce)
	tag2 := ComputeCommitTag(key, nonce)
	if tag1 != tag2 {
		t.Fatalf("expected deterministic commit tag")
	}
	if len(tag1) != 64 {
		t.Fatalf("expected 64-char hex tag, got %d", len(tag1))
	}
}

func TestComputeCommitTagDiffersForDiffKeys(t *testing.T) {
	key1 := bytes.Repeat([]byte{0xAA}, 32)
	key2 := bytes.Repeat([]byte{0xBB}, 32)
	nonce := bytes.Repeat([]byte{0x01}, 32)
	tag1 := ComputeCommitTag(key1, nonce)
	tag2 := ComputeCommitTag(key2, nonce)
	if tag1 == tag2 {
		t.Fatalf("expected different commit tags for different keys")
	}
}

func TestComputeCommitTagDiffersForDiffNonces(t *testing.T) {
	key := bytes.Repeat([]byte{0xAA}, 32)
	nonce1 := bytes.Repeat([]byte{0x01}, 32)
	nonce2 := bytes.Repeat([]byte{0x02}, 32)
	tag1 := ComputeCommitTag(key, nonce1)
	tag2 := ComputeCommitTag(key, nonce2)
	if tag1 == tag2 {
		t.Fatalf("expected different commit tags for different nonces")
	}
}

func TestWriteKDFSecretAtomicStoresCommitTag(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0xCC}, 32)
	vp := newTestVaultParams()

	if err := SaveVaultParams(dir, vp); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}
	hash, _ := HashVaultParams(vp)

	if err := writeKDFSecretAtomic(dir, KDFParams{}, key, hash); err != nil {
		t.Fatalf("writeKDFSecretAtomic error: %v", err)
	}

	b, err := os.ReadFile(secretPath(dir))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	plaintext, err := DecryptAES256GCM(b, key, []byte(hash))
	if err != nil {
		t.Fatalf("DecryptAES256GCM error: %v", err)
	}
	var sf secretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if sf.CommitTag == "" {
		t.Fatalf("expected non-empty commit tag in .secret")
	}
	if len(sf.CommitNonce) != 32 {
		t.Fatalf("expected 32-byte commit nonce, got %d", len(sf.CommitNonce))
	}
	expected := ComputeCommitTag(key, sf.CommitNonce)
	if sf.CommitTag != expected {
		t.Fatalf("commit tag mismatch: got %s, want %s", sf.CommitTag, expected)
	}
}

func TestVerifyCommitTagCorrectKey(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0xDD}, 32)
	vp := newTestVaultParams()

	setupTestSecret(t, dir, key, vp)

	if err := VerifyCommitTag(dir, key); err != nil {
		t.Fatalf("VerifyCommitTag should pass with correct key: %v", err)
	}
}

func TestVerifyCommitTagWrongKey(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0xDD}, 32)
	wrongKey := bytes.Repeat([]byte{0xEE}, 32)
	vp := newTestVaultParams()

	setupTestSecret(t, dir, key, vp)

	err := VerifyCommitTag(dir, wrongKey)
	if err == nil {
		t.Fatalf("expected error when verifying commit tag with wrong key")
	}
}

func TestLoadKDFSecretRejectsWrongKeyViaCommitTag(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0xFF}, 32)
	vp := newTestVaultParams()

	setupTestSecret(t, dir, key, vp)

	if _, err := loadKDFSecret(dir, key); err != nil {
		t.Fatalf("loadKDFSecret should succeed with correct key: %v", err)
	}
}

func TestErrWrongPasswordSentinel(t *testing.T) {
	if ErrWrongPassword == nil {
		t.Fatalf("ErrWrongPassword should not be nil")
	}
	if ErrWrongPassword.Error() != "wrong master password" {
		t.Fatalf("unexpected error message: %s", ErrWrongPassword.Error())
	}
}
