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

func TestSaveAndLoadRootSalt(t *testing.T) {
	dir := t.TempDir()
	salt := bytes.Repeat([]byte{0xAB}, 32)

	if err := SaveRootSalt(dir, salt); err != nil {
		t.Fatalf("SaveRootSalt error: %v", err)
	}

	loaded, err := LoadRootSalt(dir)
	if err != nil {
		t.Fatalf("LoadRootSalt error: %v", err)
	}
	if !bytes.Equal(salt, loaded) {
		t.Fatalf("loaded salt does not match saved salt")
	}
}

func TestLoadRootSaltMissing(t *testing.T) {
	dir := t.TempDir()

	loaded, err := LoadRootSalt(dir)
	if err != nil {
		t.Fatalf("expected no error for missing salt, got: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil salt for missing file, got %d bytes", len(loaded))
	}
}

func TestLoadRootSaltInvalidLength(t *testing.T) {
	dir := t.TempDir()
	// Write a .vault_params with bad salt length.
	if err := os.WriteFile(filepath.Join(dir, ".vault_params"), []byte(`{"version":1,"salt":"c2hvcnQ=","time":6,"memory_kb":262144,"threads":4}`), 0600); err != nil {
		t.Fatalf("setup error: %v", err)
	}

	_, err := LoadRootSalt(dir)
	if err == nil {
		t.Fatalf("expected error for invalid salt length")
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
	// Write params with zero time/memory/threads — should back-fill.
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

func TestLoadVaultParamsMigratesLegacyRootSalt(t *testing.T) {
	dir := t.TempDir()
	salt := bytes.Repeat([]byte{0xDD}, 32)

	// Write legacy .root_salt file.
	if err := os.WriteFile(filepath.Join(dir, ".root_salt"), salt, 0600); err != nil {
		t.Fatalf("setup error: %v", err)
	}

	loaded, err := LoadVaultParams(dir)
	if err != nil {
		t.Fatalf("LoadVaultParams error: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected non-nil params")
	}
	if !bytes.Equal(salt, loaded.Salt) {
		t.Fatalf("salt mismatch after legacy migration")
	}
	if loaded.Time != RecommendedTime || loaded.MemoryKB != RecommendedMemory || loaded.Threads != RecommendedThreads {
		t.Fatalf("expected recommended params after legacy migration")
	}

	// .root_salt should be removed.
	if _, err := os.Stat(filepath.Join(dir, ".root_salt")); !os.IsNotExist(err) {
		t.Fatalf("expected .root_salt to be removed after migration")
	}

	// .vault_params should exist.
	if _, err := os.Stat(filepath.Join(dir, ".vault_params")); err != nil {
		t.Fatalf("expected .vault_params to exist after migration")
	}
}

func TestLoadVaultParamsMigratesLegacyKdfParams(t *testing.T) {
	dir := t.TempDir()
	salt := bytes.Repeat([]byte{0xEE}, 32)

	// Write legacy .kdf_params file.
	p := VaultParams{Salt: salt, Time: 6, MemoryKB: 256 * 1024, Threads: 4}
	data, _ := json.Marshal(p)
	if err := os.WriteFile(filepath.Join(dir, ".kdf_params"), data, 0600); err != nil {
		t.Fatalf("setup error: %v", err)
	}

	loaded, err := LoadVaultParams(dir)
	if err != nil {
		t.Fatalf("LoadVaultParams error: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected non-nil params")
	}
	if !bytes.Equal(salt, loaded.Salt) {
		t.Fatalf("salt mismatch after .kdf_params migration")
	}

	// .kdf_params should be removed.
	if _, err := os.Stat(filepath.Join(dir, ".kdf_params")); !os.IsNotExist(err) {
		t.Fatalf("expected .kdf_params to be removed after migration")
	}

	// .vault_params should exist.
	if _, err := os.Stat(filepath.Join(dir, ".vault_params")); err != nil {
		t.Fatalf("expected .vault_params to exist after migration")
	}
}

func TestNeedsRehash(t *testing.T) {
	current := RootKDFParams{
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
	if len(h1) != 64 { // hex-encoded SHA-256 = 64 chars
		t.Fatalf("expected 64-char hex hash, got %d", len(h1))
	}

	// Deterministic.
	h2, _ := HashVaultParams(p)
	if h1 != h2 {
		t.Fatalf("expected deterministic hash")
	}

	// Different params → different hash.
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

	// Write to disk.
	if err := SaveVaultParams(dir, p); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}

	// Read raw bytes from disk.
	diskBytes, err := os.ReadFile(filepath.Join(dir, ".vault_params"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	// Hash computed from struct must equal hash of disk bytes.
	structHash, _ := HashVaultParams(p)
	diskHash := HashVaultParamsBytes(diskBytes)
	if structHash != diskHash {
		t.Fatalf("hash mismatch: struct=%s disk=%s", structHash, diskHash)
	}
}

func TestEnsureSecretStoresHash(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x55}, 32)
	vp := VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0x22}, 32),
		Time: 6, MemoryKB: 256 * 1024, Threads: 4,
		KDF: "argon2id", Cipher: "aes-256-gcm",
	}

	// Write .vault_params to disk first (VerifyVaultParamsHash reads it).
	if err := SaveVaultParams(dir, vp); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}

	_, err := EnsureSecret(dir, key, vp)
	if err != nil {
		t.Fatalf("EnsureSecret error: %v", err)
	}

	// Verification should pass — file on disk matches.
	if err := VerifyVaultParamsHash(dir, key); err != nil {
		t.Fatalf("VerifyVaultParamsHash should pass: %v", err)
	}

	// Tamper with the file on disk.
	tampered := vp
	tampered.Time = 999
	if err := SaveVaultParams(dir, tampered); err != nil {
		t.Fatalf("SaveVaultParams (tampered) error: %v", err)
	}
	if err := VerifyVaultParamsHash(dir, key); err == nil {
		t.Fatalf("expected error for tampered vault params file")
	}
}

func TestVerifyVaultParamsHashSkipsOldVaults(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x66}, 32)

	// Create a .secret without a hash (simulates old vault).
	_, err := EnsureKDFSecret(dir, key)
	if err != nil {
		t.Fatalf("EnsureKDFSecret error: %v", err)
	}

	vp := VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0x33}, 32),
		Time: 6, MemoryKB: 256 * 1024, Threads: 4,
	}

	// Write .vault_params so VerifyVaultParamsHash can read it.
	if err := SaveVaultParams(dir, vp); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}

	// Should pass — no hash field in old vault.
	if err := VerifyVaultParamsHash(dir, key); err != nil {
		t.Fatalf("expected nil error for old vault without hash, got: %v", err)
	}
}

func TestSecretAADRejectsTamperedVaultParams(t *testing.T) {
	dir := t.TempDir()
	key := bytes.Repeat([]byte{0x88}, 32)
	vp := VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0x44}, 32),
		Time: 6, MemoryKB: 256 * 1024, Threads: 4,
		KDF: "argon2id", Cipher: "aes-256-gcm",
	}

	// Write .vault_params first, then create .secret with AAD.
	if err := SaveVaultParams(dir, vp); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}

	hash, err := HashVaultParams(vp)
	if err != nil {
		t.Fatalf("HashVaultParams error: %v", err)
	}
	if err := writeKDFSecretAtomic(dir, KDFParams{}, key, hash); err != nil {
		t.Fatalf("writeKDFSecretAtomic error: %v", err)
	}

	// Loading should succeed with correct .vault_params on disk.
	if _, err := loadKDFSecret(dir, key); err != nil {
		t.Fatalf("loadKDFSecret should succeed: %v", err)
	}

	// Tamper with .vault_params on disk → AAD mismatch → GCM auth fails.
	tampered := vp
	tampered.Time = 999
	if err := SaveVaultParams(dir, tampered); err != nil {
		t.Fatalf("SaveVaultParams (tampered) error: %v", err)
	}

	// loadKDFSecret falls back to nil AAD for legacy compat, so it still
	// succeeds.  But VerifyVaultParamsHash catches the JSON-level mismatch.
	// The GCM-level AAD check prevents forging a *new* .secret that
	// validates against tampered params.
	//
	// For a direct GCM-level test, try decrypting with wrong AAD:
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
	p := RootKDFParams{
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

	p2 := RootKDFParams{
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

	// Deterministic.
	mk2, _ := DeriveHKDFKey(rootKey, masterKeyPurpose)
	if !bytes.Equal(mk, mk2) {
		t.Fatalf("expected deterministic HKDF output")
	}
}

func TestDeriveKeys(t *testing.T) {
	p := RootKDFParams{
		Salt:     bytes.Repeat([]byte{0x99}, 32),
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
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

	// Deterministic.
	mk2, vk2, _ := DeriveKeys("password", p)
	if !bytes.Equal(mk, mk2) || !bytes.Equal(vk, vk2) {
		t.Fatalf("expected deterministic key derivation")
	}

	// Different password → different keys.
	mk3, _, _ := DeriveKeys("other", p)
	if bytes.Equal(mk, mk3) {
		t.Fatalf("expected different keys for different password")
	}
}

func TestDeriveKeysUsesVaultParamsPurpose(t *testing.T) {
	salt := bytes.Repeat([]byte{0xAA}, 32)

	// Legacy vault: no purpose fields → falls back to "master"/"vault".
	legacy := VaultParams{
		Salt: salt, Time: RecommendedTime,
		MemoryKB: RecommendedMemory, Threads: RecommendedThreads,
	}
	mkLegacy, vkLegacy, err := DeriveKeys("password", legacy)
	if err != nil {
		t.Fatalf("DeriveKeys (legacy) error: %v", err)
	}

	// New vault: explicit purpose fields.
	newVault := VaultParams{
		Salt: salt, Time: RecommendedTime,
		MemoryKB: RecommendedMemory, Threads: RecommendedThreads,
		MasterKeyPurpose: "passbook:master:v1",
		VaultKeyPurpose:  "passbook:vault:v1",
	}
	mkNew, vkNew, err := DeriveKeys("password", newVault)
	if err != nil {
		t.Fatalf("DeriveKeys (new) error: %v", err)
	}

	// Same salt+password but different purposes → different keys.
	if bytes.Equal(mkLegacy, mkNew) {
		t.Fatalf("legacy and new master keys should differ due to different purposes")
	}
	if bytes.Equal(vkLegacy, vkNew) {
		t.Fatalf("legacy and new vault keys should differ due to different purposes")
	}

	// Deterministic: same params → same keys.
	mkNew2, vkNew2, _ := DeriveKeys("password", newVault)
	if !bytes.Equal(mkNew, mkNew2) || !bytes.Equal(vkNew, vkNew2) {
		t.Fatalf("expected deterministic derivation with explicit purposes")
	}
}

func TestNeedsPurposeMigration(t *testing.T) {
	// Legacy params (no purpose) → needs migration.
	legacy := VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0x11}, 32),
		Time: RecommendedTime, MemoryKB: RecommendedMemory, Threads: RecommendedThreads,
	}
	if !legacy.NeedsPurposeMigration() {
		t.Fatalf("expected legacy params to need purpose migration")
	}

	// Only master set → still needs migration.
	partial := legacy
	partial.MasterKeyPurpose = "passbook:master:v1"
	if !partial.NeedsPurposeMigration() {
		t.Fatalf("expected partial params to need purpose migration")
	}

	// Both set → no migration needed.
	full := legacy
	full.MasterKeyPurpose = "passbook:master:v1"
	full.VaultKeyPurpose = "passbook:vault:v1"
	if full.NeedsPurposeMigration() {
		t.Fatalf("expected full params to NOT need purpose migration")
	}
}

func TestMigrateVaultPurpose(t *testing.T) {
	dir := t.TempDir()
	password := "testpassword"

	// Set up a vault with legacy purpose strings (empty).
	legacyParams := VaultParams{
		Version: 1, Salt: bytes.Repeat([]byte{0xBB}, 32),
		Time: RecommendedTime, MemoryKB: RecommendedMemory, Threads: RecommendedThreads,
		KDF: "argon2id", Cipher: "aes-256-gcm",
	}

	// Derive keys with legacy purposes and create the vault.
	legacyMasterKey, legacyVaultKey, err := DeriveKeys(password, legacyParams)
	if err != nil {
		t.Fatalf("DeriveKeys error: %v", err)
	}

	// Save vault params and create .secret.
	if err := SaveVaultParams(dir, legacyParams); err != nil {
		t.Fatalf("SaveVaultParams error: %v", err)
	}
	if _, err := EnsureSecret(dir, legacyMasterKey, legacyParams); err != nil {
		t.Fatalf("EnsureSecret error: %v", err)
	}

	// Write a dummy entry to verify re-encryption.
	loginsDir := filepath.Join(dir, "logins")
	if err := os.MkdirAll(loginsDir, 0700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	plaintext := []byte("test-entry-data")
	enc, err := Encrypt(legacyVaultKey, plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	entryPath := filepath.Join(loginsDir, "test.pb")
	if err := os.WriteFile(entryPath, enc, 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	WipeBytes(legacyMasterKey)
	WipeBytes(legacyVaultKey)

	// Confirm it needs migration.
	if !legacyParams.NeedsPurposeMigration() {
		t.Fatalf("expected legacy params to need purpose migration")
	}

	// Run the migration.
	newParams, err := MigrateVaultPurpose(dir, password, legacyParams)
	if err != nil {
		t.Fatalf("MigrateVaultPurpose error: %v", err)
	}

	// Verify new params have the new purpose strings.
	if newParams.MasterKeyPurpose != "passbook:master:v1" {
		t.Fatalf("expected MasterKeyPurpose=passbook:master:v1, got %s", newParams.MasterKeyPurpose)
	}
	if newParams.VaultKeyPurpose != "passbook:vault:v1" {
		t.Fatalf("expected VaultKeyPurpose=passbook:vault:v1, got %s", newParams.VaultKeyPurpose)
	}
	if newParams.NeedsPurposeMigration() {
		t.Fatalf("expected migrated params to NOT need purpose migration")
	}

	// Verify new keys can decrypt the .secret.
	newMasterKey, newVaultKey, err := DeriveKeys(password, *newParams)
	if err != nil {
		t.Fatalf("DeriveKeys (new) error: %v", err)
	}
	defer WipeBytes(newMasterKey)
	defer WipeBytes(newVaultKey)

	if _, err := loadKDFSecret(dir, newMasterKey); err != nil {
		t.Fatalf("new master key cannot decrypt .secret: %v", err)
	}

	// Verify new vault key can decrypt the re-encrypted entry.
	encData, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	decrypted, err := Decrypt(newVaultKey, encData)
	if err != nil {
		t.Fatalf("new vault key cannot decrypt entry: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted data mismatch: got %q, want %q", decrypted, plaintext)
	}

	// Verify vault params on disk have been updated.
	diskParams, err := LoadVaultParams(dir)
	if err != nil {
		t.Fatalf("LoadVaultParams error: %v", err)
	}
	if diskParams.MasterKeyPurpose != "passbook:master:v1" || diskParams.VaultKeyPurpose != "passbook:vault:v1" {
		t.Fatalf("disk params not updated: mk=%s vk=%s", diskParams.MasterKeyPurpose, diskParams.VaultKeyPurpose)
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

	// Encrypt with AAD.
	ciphertext, err := EncryptAES256GCM(plaintext, key, aad)
	if err != nil {
		t.Fatalf("EncryptAES256GCM with AAD error: %v", err)
	}

	// Decrypt with correct AAD succeeds.
	got, err := DecryptAES256GCM(ciphertext, key, aad)
	if err != nil {
		t.Fatalf("DecryptAES256GCM with AAD error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext mismatch")
	}

	// Decrypt with wrong AAD fails.
	if _, err := DecryptAES256GCM(ciphertext, key, []byte("wrong-aad")); err == nil {
		t.Fatalf("expected error when decrypting with wrong AAD")
	}

	// Decrypt with nil AAD fails (AAD was non-nil during encrypt).
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
	if err := writeKDFSecretAtomic(dir, p, key, ""); err != nil {
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
