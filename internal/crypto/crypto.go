package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

// ErrWrongPassword is returned when the master password is incorrect.
// This is detected via the HMAC commit tag stored in .secret.
var ErrWrongPassword = errors.New("wrong master password")

func WipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

type KDFParams struct {
	Salt     []byte
	Time     uint32
	MemoryKB uint32
	Threads  uint8
}

type secretFile struct {
	Version         int    `json:"version"`
	Salt            []byte `json:"salt"`
	Time            uint32 `json:"time"`
	MemoryKB        uint32 `json:"memory_kb"`
	Threads         uint8  `json:"threads"`
	KeyLen          uint32 `json:"key_len"`
	KDF             string `json:"kdf"`
	VaultParamsHash string `json:"vault_params_hash,omitempty"` // SHA-256 of .vault_params
	CommitNonce     []byte `json:"commit_nonce,omitempty"`      // random nonce for commit tag HMAC
	CommitTag       string `json:"commit_tag,omitempty"`        // HMAC-SHA256(masterKey, commitNonce) for wrong-password detection
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	VaultID         string `json:"vault_id,omitempty"`
	Reserved01      string `json:"reserved01,omitempty"`
}

const (
	masterKeyPurpose = "passbook:master:v1"
	vaultKeyPurpose  = "passbook:vault:v1"
	masterKeyLegacy  = "master"
	vaultKeyLegacy   = "vault"
)

// ComputeCommitTag returns hex(HMAC-SHA256(masterKey, nonce)).
// The nonce is a random value unique to each vault, stored alongside
// the tag inside .secret for explicit wrong-password detection.
func ComputeCommitTag(masterKey []byte, nonce []byte) string {
	mac := hmac.New(sha256.New, masterKey)
	mac.Write(nonce)
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateCommitNonce returns a 32-byte random nonce for use as the
// HMAC message in commit tag computation.
func GenerateCommitNonce() ([]byte, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating commit nonce: %w", err)
	}
	return nonce, nil
}

// supportLegacy enables backward compatibility with vaults created before the
// HKDF-based key hierarchy. Set to false and remove all code blocks marked
// "--- BEGIN supportLegacy" / "--- END supportLegacy" once every user has migrated.
var supportLegacy = true

// SupportLegacy returns whether legacy fixed-salt key derivation is enabled.
func SupportLegacy() bool { return supportLegacy }

func DeriveKey(password string, p KDFParams) []byte {
	return argon2.IDKey([]byte(password), p.Salt, p.Time, p.MemoryKB, p.Threads, 32)
}

// --- BEGIN supportLegacy ---

// DeriveMasterKey delegates to the legacy fixed-salt derivation.
// Deprecated: only used when supportLegacy is true.
func DeriveMasterKey(masterPassword string) []byte {
	return DeriveLegacyMasterKey(masterPassword)
}

// DeriveLegacyMasterKey derives a master key using the old fixed-salt scheme.
// Kept for backward compatibility with existing vaults (IsMigrated == false).
func DeriveLegacyMasterKey(masterPassword string) []byte {
	salt := []byte("768250f1-214a-4b8d-89b4-5edf1e85f65e")

	return argon2.IDKey(
		[]byte(masterPassword),
		salt,
		6,
		256*1024,
		4,
		32,
	)
}

// --- END supportLegacy ---

func GenerateRootSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating root salt: %w", err)
	}
	return salt, nil
}

const (
	RecommendedTime    uint32 = 6
	RecommendedMemory  uint32 = 256 * 1024 // 256 MB in KiB
	RecommendedThreads uint8  = 4
)

type VaultParams struct {
	Version          int    `json:"version"`                      // schema version (currently 1)
	Salt             []byte `json:"salt"`                         // 32-byte random Argon2id salt
	Time             uint32 `json:"time"`                         // Argon2id time/iterations
	MemoryKB         uint32 `json:"memory_kb"`                    // Argon2id memory in KiB
	Threads          uint8  `json:"threads"`                      // Argon2id parallelism
	KDF              string `json:"kdf,omitempty"`                // identifier, e.g. "argon2id"
	Cipher           string `json:"cipher,omitempty"`             // identifier, e.g. "aes-256-gcm"
	MasterKeyPurpose string `json:"master_key_purpose,omitempty"` // HKDF purpose for master key
	VaultKeyPurpose  string `json:"vault_key_purpose,omitempty"`  // HKDF purpose for vault key
}

type RootKDFParams = VaultParams

func DefaultVaultParams() (VaultParams, error) {
	salt, err := GenerateRootSalt()
	if err != nil {
		return VaultParams{}, err
	}
	return VaultParams{
		Version:          1,
		Salt:             salt,
		Time:             RecommendedTime,
		MemoryKB:         RecommendedMemory,
		Threads:          RecommendedThreads,
		KDF:              "argon2id",
		Cipher:           "aes-256-gcm",
		MasterKeyPurpose: masterKeyPurpose,
		VaultKeyPurpose:  vaultKeyPurpose,
	}, nil
}

// DefaultRootKDFParams is a backward-compatible alias.
// Deprecated: use DefaultVaultParams.
var DefaultRootKDFParams = DefaultVaultParams

// NeedsRehash returns true when any stored parameter is strictly weaker
// (lower) than the current recommended values.
func (p VaultParams) NeedsRehash() bool {
	return p.Time < RecommendedTime ||
		p.MemoryKB < RecommendedMemory ||
		p.Threads < RecommendedThreads
}

func marshalVaultParams(p VaultParams) ([]byte, error) {
	if p.Version == 0 {
		p.Version = 1
	}
	if p.KDF == "" {
		p.KDF = "argon2id"
	}
	if p.Cipher == "" {
		p.Cipher = "aes-256-gcm"
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshalling vault params: %w", err)
	}
	return data, nil
}

func HashVaultParamsBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HashVaultParams(p VaultParams) (string, error) {
	data, err := marshalVaultParams(p)
	if err != nil {
		return "", err
	}
	return HashVaultParamsBytes(data), nil
}

func vaultParamsPath(dataDir string) string {
	return filepath.Join(dataDir, ".vault_params")
}

func legacyKdfParamsPath(dataDir string) string {
	return filepath.Join(dataDir, ".kdf_params")
}

func rootSaltPath(dataDir string) string {
	return filepath.Join(dataDir, ".root_salt")
}

func SaveVaultParams(dataDir string, p VaultParams) error {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	data, err := marshalVaultParams(p)
	if err != nil {
		return err
	}
	if err := os.WriteFile(vaultParamsPath(dataDir), data, 0600); err != nil {
		return fmt.Errorf("writing vault params: %w", err)
	}
	return nil
}

var SaveRootKDFParams = SaveVaultParams

func LoadVaultParams(dataDir string) (*VaultParams, error) {
	if p, err := loadVaultParamsFrom(vaultParamsPath(dataDir)); err == nil {
		return p, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if p, err := loadVaultParamsFrom(legacyKdfParamsPath(dataDir)); err == nil {
		if err := SaveVaultParams(dataDir, *p); err != nil {
			return nil, fmt.Errorf("migrating .kdf_params: %w", err)
		}
		_ = os.Remove(legacyKdfParamsPath(dataDir))
		return p, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	saltData, err := os.ReadFile(rootSaltPath(dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // nothing found
		}
		return nil, fmt.Errorf("reading legacy root salt: %w", err)
	}
	if len(saltData) != 32 {
		return nil, fmt.Errorf("invalid legacy root salt: expected 32 bytes, got %d", len(saltData))
	}
	p := &VaultParams{
		Version:  1,
		Salt:     saltData,
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
		KDF:      "argon2id",
		Cipher:   "aes-256-gcm",
	}
	if err := SaveVaultParams(dataDir, *p); err != nil {
		return nil, fmt.Errorf("migrating legacy root salt: %w", err)
	}
	_ = os.Remove(rootSaltPath(dataDir))
	return p, nil
}

func loadVaultParamsFrom(path string) (*VaultParams, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p VaultParams
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing vault params: %w", err)
	}
	if len(p.Salt) != 32 {
		return nil, fmt.Errorf("invalid vault params: expected 32-byte salt, got %d", len(p.Salt))
	}
	if p.Version == 0 {
		p.Version = 1
	}
	if p.Time == 0 {
		p.Time = RecommendedTime
	}
	if p.MemoryKB == 0 {
		p.MemoryKB = RecommendedMemory
	}
	if p.Threads == 0 {
		p.Threads = RecommendedThreads
	}
	if p.KDF == "" {
		p.KDF = "argon2id"
	}
	if p.Cipher == "" {
		p.Cipher = "aes-256-gcm"
	}
	return &p, nil
}

var LoadRootKDFParams = LoadVaultParams

func SaveRootSalt(dataDir string, salt []byte) error {
	p, _ := DefaultVaultParams()
	p.Salt = salt
	return SaveVaultParams(dataDir, p)
}

func LoadRootSalt(dataDir string) ([]byte, error) {
	p, err := LoadVaultParams(dataDir)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return p.Salt, nil
}

func DeriveRootKey(password string, p VaultParams) []byte {
	return argon2.IDKey(
		[]byte(password),
		p.Salt,
		p.Time,
		p.MemoryKB,
		p.Threads,
		32,
	)
}

func DeriveHKDFKey(rootKey []byte, purpose string) ([]byte, error) {
	r := hkdf.New(sha256.New, rootKey, nil, []byte(purpose))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("HKDF expand for %q: %w", purpose, err)
	}
	return key, nil
}

func DeriveKeys(password string, p VaultParams) (masterKey, vaultKey []byte, err error) {
	rootKey := DeriveRootKey(password, p)
	defer WipeBytes(rootKey)

	// Use the purpose strings stored in VaultParams.
	// Existing vaults without these fields fall back to the legacy
	// fixed-purpose strings ("master" / "vault").
	mk := p.MasterKeyPurpose
	if mk == "" {
		mk = masterKeyLegacy
	}
	vk := p.VaultKeyPurpose
	if vk == "" {
		vk = vaultKeyLegacy
	}

	masterKey, err = DeriveHKDFKey(rootKey, mk)
	if err != nil {
		return nil, nil, err
	}
	vaultKey, err = DeriveHKDFKey(rootKey, vk)
	if err != nil {
		return nil, nil, err
	}
	return masterKey, vaultKey, nil
}

func RehashVault(dataDir string, password string, oldParams VaultParams) (*VaultParams, error) {
	// Derive old keys.
	oldMasterKey, oldVaultKey, err := DeriveKeys(password, oldParams)
	if err != nil {
		return nil, fmt.Errorf("deriving old keys: %w", err)
	}
	defer WipeBytes(oldMasterKey)
	defer WipeBytes(oldVaultKey)

	// Verify old keys work.
	if _, err := loadKDFSecret(dataDir, oldMasterKey); err != nil {
		return nil, fmt.Errorf("old keys invalid (wrong password?): %w", err)
	}

	// Build new params: keep the same salt and purpose strings, bump the Argon2id cost.
	newParams := VaultParams{
		Version:          1,
		Salt:             oldParams.Salt,
		Time:             RecommendedTime,
		MemoryKB:         RecommendedMemory,
		Threads:          RecommendedThreads,
		KDF:              "argon2id",
		Cipher:           "aes-256-gcm",
		MasterKeyPurpose: oldParams.MasterKeyPurpose,
		VaultKeyPurpose:  oldParams.VaultKeyPurpose,
	}

	// Derive new keys with stronger params.
	newMasterKey, newVaultKey, err := DeriveKeys(password, newParams)
	if err != nil {
		return nil, fmt.Errorf("deriving new keys: %w", err)
	}
	defer WipeBytes(newMasterKey)
	defer WipeBytes(newVaultKey)

	// Re-encrypt .secret (includes vault params hash).
	if err := writeSecretWithParams(dataDir, newParams, newMasterKey); err != nil {
		return nil, fmt.Errorf("re-keying vault secret: %w", err)
	}

	// Re-encrypt all entries and attachments.
	if err := ReKeyEntries(dataDir, oldVaultKey, newVaultKey); err != nil {
		return nil, fmt.Errorf("re-keying entries: %w", err)
	}

	// Persist the new parameters.
	if err := SaveVaultParams(dataDir, newParams); err != nil {
		return nil, fmt.Errorf("saving vault params: %w", err)
	}

	return &newParams, nil
}

// NeedsPurposeMigration returns true when the vault params are missing the
// new HKDF purpose strings and should be upgraded from the legacy values.
func (p VaultParams) NeedsPurposeMigration() bool {
	return p.MasterKeyPurpose == "" || p.VaultKeyPurpose == ""
}

// MigrateVaultPurpose upgrades a vault from legacy HKDF purpose strings
// ("master"/"vault") to the new versioned strings ("passbook:master:v1" /
// "passbook:vault:v1"). It re-derives keys with both old and new purposes,
// re-encrypts .secret and all entries, and persists the updated vault params.
func MigrateVaultPurpose(dataDir string, password string, oldParams VaultParams) (*VaultParams, error) {
	// Derive keys with the old (legacy) purposes.
	oldMasterKey, oldVaultKey, err := DeriveKeys(password, oldParams)
	if err != nil {
		return nil, fmt.Errorf("deriving old keys: %w", err)
	}
	defer WipeBytes(oldMasterKey)
	defer WipeBytes(oldVaultKey)

	// Verify old keys work.
	if _, err := loadKDFSecret(dataDir, oldMasterKey); err != nil {
		return nil, fmt.Errorf("old keys invalid (wrong password?): %w", err)
	}

	// Build new params: keep everything, only upgrade the purpose strings.
	newParams := oldParams
	newParams.MasterKeyPurpose = masterKeyPurpose
	newParams.VaultKeyPurpose = vaultKeyPurpose

	// Derive keys with the new purposes.
	newMasterKey, newVaultKey, err := DeriveKeys(password, newParams)
	if err != nil {
		return nil, fmt.Errorf("deriving new keys: %w", err)
	}
	defer WipeBytes(newMasterKey)
	defer WipeBytes(newVaultKey)

	// Re-encrypt .secret with the new master key.
	if err := writeSecretWithParams(dataDir, newParams, newMasterKey); err != nil {
		return nil, fmt.Errorf("re-keying vault secret: %w", err)
	}

	// Re-encrypt all entries and attachments.
	if err := ReKeyEntries(dataDir, oldVaultKey, newVaultKey); err != nil {
		return nil, fmt.Errorf("re-keying entries: %w", err)
	}

	// Persist the updated vault params with new purpose strings.
	if err := SaveVaultParams(dataDir, newParams); err != nil {
		return nil, fmt.Errorf("saving vault params: %w", err)
	}

	return &newParams, nil
}

// --- BEGIN supportLegacy ---

// MigrateVault migrates a vault from the legacy fixed-salt scheme to the new
// HKDF-based scheme.
func MigrateVault(dataDir string, password string) (*VaultParams, error) {
	// Derive old keys using legacy scheme.
	oldMasterKey := DeriveLegacyMasterKey(password)
	defer WipeBytes(oldMasterKey)
	oldKDF, err := loadKDFSecret(dataDir, oldMasterKey)
	if err != nil {
		return nil, fmt.Errorf("loading legacy secret (wrong password?): %w", err)
	}
	oldVaultKey := DeriveKey(password, oldKDF)
	defer WipeBytes(oldVaultKey)

	// Generate new params with fresh salt and recommended Argon2id cost.
	newParams, err := DefaultVaultParams()
	if err != nil {
		return nil, err
	}

	// Derive new keys using HKDF scheme.
	newMasterKey, newVaultKey, err := DeriveKeys(password, newParams)
	if err != nil {
		return nil, fmt.Errorf("deriving new keys: %w", err)
	}
	defer WipeBytes(newMasterKey)
	defer WipeBytes(newVaultKey)

	// Re-encrypt .secret with the new master key (includes vault params hash).
	if err := writeSecretWithParams(dataDir, newParams, newMasterKey); err != nil {
		return nil, fmt.Errorf("re-keying vault secret: %w", err)
	}

	// Re-encrypt all entries and attachments.
	if err := ReKeyEntries(dataDir, oldVaultKey, newVaultKey); err != nil {
		return nil, fmt.Errorf("re-keying entries: %w", err)
	}

	// Persist the new vault params.
	if err := SaveVaultParams(dataDir, newParams); err != nil {
		return nil, fmt.Errorf("saving vault params: %w", err)
	}

	return &newParams, nil
}

// --- END supportLegacy ---

func VaultHasEntries(dataDir string) bool {
	categories := []string{"logins", "cards", "notes", "files"}
	for _, cat := range categories {
		dir := filepath.Join(dataDir, cat)
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".pb" {
				return true
			}
		}
	}
	return false
}

func generateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	allZero := true
	for _, b := range nonce {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return nil, errors.New("generating nonce: CSPRNG returned all-zero bytes")
	}
	return nonce, nil
}

func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := generateNonce(gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, rest := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, rest, nil)
}

func secretPath(dataDir string) string {
	return filepath.Join(dataDir, ".secret")
}

func vaultParamsAAD(dataDir string) []byte {
	data, err := os.ReadFile(vaultParamsPath(dataDir))
	if err != nil {
		return nil
	}
	h := HashVaultParamsBytes(data)
	return []byte(h)
}

func EnsureKDFSecret(dataDir string, masterKey []byte) (KDFParams, error) {
	if p, err := loadKDFSecret(dataDir, masterKey); err == nil {
		// Older vaults may lack the commit tag. Rewrite .secret to add one.
		if !secretHasCommitTag(dataDir, masterKey) {
			if writeErr := writeKDFSecretAtomic(dataDir, p, masterKey, ""); writeErr != nil {
				// Non-fatal: vault still works, just without commit tag.
				return p, nil
			}
		}
		return p, nil
	}
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, masterKey, ""); err != nil {
		return KDFParams{}, err
	}
	return loadKDFSecret(dataDir, masterKey)
}

func EnsureSecret(dataDir string, masterKey []byte, vp VaultParams) (KDFParams, error) {
	if p, err := loadKDFSecret(dataDir, masterKey); err == nil {
		// Older vaults may lack the commit tag. Rewrite .secret to add one.
		if !secretHasCommitTag(dataDir, masterKey) {
			hash, hashErr := HashVaultParams(vp)
			if hashErr != nil {
				return KDFParams{}, hashErr
			}
			if writeErr := writeKDFSecretAtomic(dataDir, p, masterKey, hash); writeErr != nil {
				// Non-fatal: vault still works, just without commit tag.
				return p, nil
			}
		}
		return p, nil
	}
	hash, err := HashVaultParams(vp)
	if err != nil {
		return KDFParams{}, err
	}
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, masterKey, hash); err != nil {
		return KDFParams{}, err
	}
	return loadKDFSecret(dataDir, masterKey)
}

func VerifyVaultParamsHash(dataDir string, masterKey []byte) error {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return fmt.Errorf("reading secret: %w", err)
	}

	// Build AAD from .vault_params on disk.
	aad := vaultParamsAAD(dataDir)

	plaintext, err := DecryptAES256GCM(b, masterKey, aad)
	if err != nil && aad != nil {
		// Fall back to nil AAD for legacy vaults.
		plaintext, err = DecryptAES256GCM(b, masterKey, nil)
	}
	if err != nil {
		return fmt.Errorf("decrypting secret: %w", err)
	}
	var sf secretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		return fmt.Errorf("parsing secret: %w", err)
	}
	// Old vaults without the hash field — skip verification.
	if sf.VaultParamsHash == "" {
		return nil
	}

	// Hash the exact bytes sitting on disk, not a re-serialisation.
	fileBytes, err := os.ReadFile(vaultParamsPath(dataDir))
	if err != nil {
		return fmt.Errorf("reading vault params file: %w", err)
	}
	actual := HashVaultParamsBytes(fileBytes)
	if sf.VaultParamsHash != actual {
		return fmt.Errorf("vault params hash mismatch: .vault_params may have been tampered with")
	}
	return nil
}

// VerifyCommitTag reads and decrypts .secret, then checks the stored
// HMAC commit tag against the provided master key and the nonce stored
// alongside it. Returns ErrWrongPassword if the tag does not match,
// nil if verification succeeds or the tag is absent (legacy vaults).
func VerifyCommitTag(dataDir string, masterKey []byte) error {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return fmt.Errorf("reading secret: %w", err)
	}

	aad := vaultParamsAAD(dataDir)

	plaintext, err := DecryptAES256GCM(b, masterKey, aad)
	if err != nil && aad != nil {
		plaintext, err = DecryptAES256GCM(b, masterKey, nil)
	}
	if err != nil {
		return fmt.Errorf("decrypting secret: %w", err)
	}

	var sf secretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		return fmt.Errorf("parsing secret: %w", err)
	}

	// Old vaults without the commit tag — skip verification.
	if sf.CommitTag == "" || len(sf.CommitNonce) == 0 {
		return nil
	}

	expected := ComputeCommitTag(masterKey, sf.CommitNonce)
	if !hmac.Equal([]byte(sf.CommitTag), []byte(expected)) {
		return ErrWrongPassword
	}
	return nil
}

func loadKDFSecret(dataDir string, masterKey []byte) (KDFParams, error) {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return KDFParams{}, err
	}

	// Build AAD from .vault_params on disk (if present).
	aad := vaultParamsAAD(dataDir)

	// Try decryption with AAD first (new vaults).
	plaintext, err := DecryptAES256GCM(b, masterKey, aad)
	if err != nil && aad != nil {
		// Fall back to nil AAD for legacy vaults encrypted without AAD.
		plaintext, err = DecryptAES256GCM(b, masterKey, nil)
	}
	if err != nil {
		return KDFParams{}, err
	}

	var sf secretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		return KDFParams{}, err
	}
	if sf.Version <= 0 {
		return KDFParams{}, errors.New("invalid secret version")
	}
	if len(sf.Salt) != 16 {
		return KDFParams{}, errors.New("invalid salt")
	}

	// Verify the commit tag if present (new vaults).
	// Old vaults without the tag are accepted for backward compatibility.
	if sf.CommitTag != "" && len(sf.CommitNonce) > 0 {
		expected := ComputeCommitTag(masterKey, sf.CommitNonce)
		if !hmac.Equal([]byte(sf.CommitTag), []byte(expected)) {
			return KDFParams{}, ErrWrongPassword
		}
	}

	p := KDFParams{Salt: sf.Salt, Time: sf.Time, MemoryKB: sf.MemoryKB, Threads: sf.Threads}
	ensureKDFParams(&p)
	return p, nil
}

// secretHasCommitTag decrypts .secret and returns true if the file
// already contains a non-empty commit nonce and tag.
func secretHasCommitTag(dataDir string, masterKey []byte) bool {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return false
	}
	aad := vaultParamsAAD(dataDir)
	plaintext, err := DecryptAES256GCM(b, masterKey, aad)
	if err != nil && aad != nil {
		plaintext, err = DecryptAES256GCM(b, masterKey, nil)
	}
	if err != nil {
		return false
	}
	var sf secretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		return false
	}
	return sf.CommitTag != "" && len(sf.CommitNonce) > 0
}

func writeKDFSecretAtomic(dataDir string, p KDFParams, masterKey []byte, vaultParamsHash string) error {
	ensureKDFParams(&p)
	if len(p.Salt) != 16 {
		s := make([]byte, 16)
		_, _ = rand.Read(s)
		p.Salt = s
	}
	commitNonce, err := GenerateCommitNonce()
	if err != nil {
		return err
	}
	sf := secretFile{
		Version:         1,
		Salt:            p.Salt,
		Time:            p.Time,
		MemoryKB:        p.MemoryKB,
		Threads:         p.Threads,
		KeyLen:          32,
		KDF:             "argon2id",
		VaultParamsHash: vaultParamsHash,
		CommitNonce:     commitNonce,
		CommitTag:       ComputeCommitTag(masterKey, commitNonce),
	}
	b, _ := json.MarshalIndent(sf, "", "  ")

	// Use the vault params hash as GCM AAD so that any tampering with
	// .vault_params causes authenticated decryption to fail.
	var aad []byte
	if vaultParamsHash != "" {
		aad = []byte(vaultParamsHash)
	}
	ciphertext, err := EncryptAES256GCM(b, masterKey, aad)
	if err != nil {
		return err
	}

	vaultDir := filepath.Dir(secretPath(dataDir))
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		return err
	}

	tmp := secretPath(dataDir) + ".tmp"
	if err := os.WriteFile(tmp, ciphertext, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, secretPath(dataDir)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func WriteSecretWithParams(dataDir string, vp VaultParams, masterKey []byte) error {
	hash, err := HashVaultParams(vp)
	if err != nil {
		return err
	}
	return writeKDFSecretAtomic(dataDir, KDFParams{}, masterKey, hash)
}

var writeSecretWithParams = WriteSecretWithParams

func ReKeyVault(dataDir string, newMasterKey []byte) error {
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, newMasterKey, ""); err != nil {
		return fmt.Errorf("writing new secret: %w", err)
	}
	return nil
}

func ReKeyEntries(dataDir string, oldKey, newKey []byte) error {
	// 1. Re-encrypt all .pb entry files in category sub-directories.
	categories := []string{"logins", "cards", "notes", "files"}
	for _, cat := range categories {
		dir := filepath.Join(dataDir, cat)
		files, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading %s: %w", dir, err)
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".pb" {
				continue
			}
			path := filepath.Join(dir, f.Name())
			if err := reEncryptFile(path, oldKey, newKey); err != nil {
				return fmt.Errorf("re-encrypting %s: %w", path, err)
			}
		}
	}

	// 2. Re-encrypt all attachment blobs.
	attDir := filepath.Join(dataDir, "_attachments")
	attFiles, err := os.ReadDir(attDir)
	if err == nil {
		for _, f := range attFiles {
			if f.IsDir() {
				continue
			}
			path := filepath.Join(attDir, f.Name())
			if err := reEncryptFile(path, oldKey, newKey); err != nil {
				return fmt.Errorf("re-encrypting attachment %s: %w", path, err)
			}
		}
	}
	return nil
}

// reEncryptFile decrypts a file with oldKey and re-encrypts it with newKey in place.
func reEncryptFile(path string, oldKey, newKey []byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	plain, err := Decrypt(oldKey, data)
	if err != nil {
		return err
	}
	defer WipeBytes(plain)
	enc, err := Encrypt(newKey, plain)
	if err != nil {
		return err
	}
	return os.WriteFile(path, enc, 0600)
}

func ensureKDFParams(p *KDFParams) {
	if p.Time == 0 {
		p.Time = 6
	}
	if p.MemoryKB == 0 {
		p.MemoryKB = 256 * 1024
	}
	if p.Threads == 0 {
		p.Threads = 4
	}
}

func EncryptAES256GCM(plaintext, key, aad []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := generateNonce(gcm.NonceSize())
	if err != nil {
		return nil, err
	}

	// Prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, aad)
	return ciphertext, nil
}

func DecryptAES256GCM(ciphertext, key, aad []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
