package crypto

import (
	"crypto/aes"
	"crypto/cipher"
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

// WipeBytes overwrites a byte slice with zeroes.  Call this to remove
// sensitive key material from memory as soon as it is no longer needed.
// This is a best-effort defence — the Go GC may have already copied the
// data, but zeroing the authoritative slice limits the exposure window.
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
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	VaultID         string `json:"vault_id,omitempty"`
	Reserved01      string `json:"reserved01,omitempty"`
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

// GenerateRootSalt creates a cryptographically random 32-byte salt.
func GenerateRootSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating root salt: %w", err)
	}
	return salt, nil
}

// ---------------------------------------------------------------------------
// Recommended Argon2id parameters for root key derivation.
// Bump these to strengthen new vaults / trigger automatic rehash on login.
// ---------------------------------------------------------------------------

const (
	RecommendedTime    uint32 = 6
	RecommendedMemory  uint32 = 256 * 1024 // 256 MB in KiB
	RecommendedThreads uint8  = 4
)

// VaultParams holds all public vault parameters stored alongside the
// encrypted data.  The struct is versioned so new fields can be added
// without breaking existing vaults.
//
// Stored as JSON in <dataDir>/.vault_params.
type VaultParams struct {
	Version  int    `json:"version"`          // schema version (currently 1)
	Salt     []byte `json:"salt"`             // 32-byte random Argon2id salt
	Time     uint32 `json:"time"`             // Argon2id time/iterations
	MemoryKB uint32 `json:"memory_kb"`        // Argon2id memory in KiB
	Threads  uint8  `json:"threads"`          // Argon2id parallelism
	KDF      string `json:"kdf,omitempty"`    // identifier, e.g. "argon2id"
	Cipher   string `json:"cipher,omitempty"` // identifier, e.g. "aes-256-gcm"
}

// RootKDFParams is a backward-compatible alias for VaultParams.
// Deprecated: use VaultParams directly.
type RootKDFParams = VaultParams

// DefaultVaultParams returns a new VaultParams with a fresh random salt
// and the current recommended Argon2id parameters.
func DefaultVaultParams() (VaultParams, error) {
	salt, err := GenerateRootSalt()
	if err != nil {
		return VaultParams{}, err
	}
	return VaultParams{
		Version:  1,
		Salt:     salt,
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
		KDF:      "argon2id",
		Cipher:   "aes-256-gcm",
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

// HashVaultParams computes a SHA-256 hex digest of the canonical JSON
// encoding of a VaultParams value.  This hash is stored inside the
// encrypted .secret file so tampering with .vault_params can be detected.
func HashVaultParams(p VaultParams) (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshalling vault params for hash: %w", err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// vaultParamsPath returns the path to the vault params file.
func vaultParamsPath(dataDir string) string {
	return filepath.Join(dataDir, ".vault_params")
}

// legacyKdfParamsPath returns the old .kdf_params path (kept for migration).
func legacyKdfParamsPath(dataDir string) string {
	return filepath.Join(dataDir, ".kdf_params")
}

// rootSaltPath returns the legacy .root_salt path (kept for migration).
func rootSaltPath(dataDir string) string {
	return filepath.Join(dataDir, ".root_salt")
}

// SaveVaultParams writes the vault parameters to <dataDir>/.vault_params.
func SaveVaultParams(dataDir string, p VaultParams) error {
	if p.Version == 0 {
		p.Version = 1
	}
	if p.KDF == "" {
		p.KDF = "argon2id"
	}
	if p.Cipher == "" {
		p.Cipher = "aes-256-gcm"
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling vault params: %w", err)
	}
	if err := os.WriteFile(vaultParamsPath(dataDir), data, 0600); err != nil {
		return fmt.Errorf("writing vault params: %w", err)
	}
	return nil
}

// SaveRootKDFParams is a backward-compatible alias.
// Deprecated: use SaveVaultParams.
var SaveRootKDFParams = SaveVaultParams

// LoadVaultParams reads the vault parameters from <dataDir>/.vault_params.
// Falls back to legacy .kdf_params and .root_salt if the new file is absent.
// Returns nil, nil if no params file exists at all.
func LoadVaultParams(dataDir string) (*VaultParams, error) {
	// 1. Try .vault_params (current).
	if p, err := loadVaultParamsFrom(vaultParamsPath(dataDir)); err == nil {
		return p, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// 2. Try legacy .kdf_params → migrate to .vault_params.
	if p, err := loadVaultParamsFrom(legacyKdfParamsPath(dataDir)); err == nil {
		if err := SaveVaultParams(dataDir, *p); err != nil {
			return nil, fmt.Errorf("migrating .kdf_params: %w", err)
		}
		_ = os.Remove(legacyKdfParamsPath(dataDir))
		return p, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// 3. Try legacy .root_salt → migrate to .vault_params.
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

// loadVaultParamsFrom reads and validates a VaultParams JSON file at the
// given path.  Returns os.ErrNotExist-wrapping error when file is absent.
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
	// Back-fill defaults.
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

// LoadRootKDFParams is a backward-compatible alias.
// Deprecated: use LoadVaultParams.
var LoadRootKDFParams = LoadVaultParams

// SaveRootSalt is a convenience wrapper that writes a VaultParams with
// recommended parameters.
func SaveRootSalt(dataDir string, salt []byte) error {
	p, _ := DefaultVaultParams()
	p.Salt = salt
	return SaveVaultParams(dataDir, p)
}

// LoadRootSalt is a convenience wrapper that returns only the salt.
// Returns nil, nil if no params file exists.
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

// DeriveRootKey derives a root key from the password using the given
// VaultParams (salt + Argon2id parameters).
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

// DeriveHKDFKey derives a purpose-specific 32-byte sub-key from a root key
// using HKDF-SHA256. Purpose should be "master" or "vault".
func DeriveHKDFKey(rootKey []byte, purpose string) ([]byte, error) {
	r := hkdf.New(sha256.New, rootKey, nil, []byte(purpose))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("HKDF expand for %q: %w", purpose, err)
	}
	return key, nil
}

// DeriveKeys derives both the master key (for .secret encryption) and the
// vault key (for entry encryption) from a password and VaultParams using
// the HKDF-based scheme:
//
//	root_key   = Argon2id(password, salt, time, memory, threads)
//	master_key = HKDF(root_key, "master")
//	vault_key  = HKDF(root_key, "vault")
func DeriveKeys(password string, p VaultParams) (masterKey, vaultKey []byte, err error) {
	rootKey := DeriveRootKey(password, p)
	defer WipeBytes(rootKey)
	masterKey, err = DeriveHKDFKey(rootKey, "master")
	if err != nil {
		return nil, nil, err
	}
	vaultKey, err = DeriveHKDFKey(rootKey, "vault")
	if err != nil {
		return nil, nil, err
	}
	return masterKey, vaultKey, nil
}

// RehashVault upgrades a vault's Argon2id parameters to the current
// recommended values.  It re-derives keys with the stronger parameters and
// re-encrypts the .secret file and all entries/attachments in place.
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

	// Build new params: keep the same salt, bump the Argon2id cost.
	newParams := VaultParams{
		Version:  1,
		Salt:     oldParams.Salt,
		Time:     RecommendedTime,
		MemoryKB: RecommendedMemory,
		Threads:  RecommendedThreads,
		KDF:      "argon2id",
		Cipher:   "aes-256-gcm",
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

// VaultHasEntries checks if the vault directory contains any .pb entry files.
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

func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
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

// EnsureKDFSecret loads or creates the .secret file encrypted with masterKey.
// When vaultParams is available, callers should prefer EnsureSecret which
// also stores the vault params hash.
func EnsureKDFSecret(dataDir string, masterKey []byte) (KDFParams, error) {
	if p, err := loadKDFSecret(dataDir, masterKey); err == nil {
		return p, nil
	}
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, masterKey, ""); err != nil {
		return KDFParams{}, err
	}
	return loadKDFSecret(dataDir, masterKey)
}

// EnsureSecret loads the .secret file or creates it with the vault params
// hash embedded.  Returns the KDF params and an error.
func EnsureSecret(dataDir string, masterKey []byte, vp VaultParams) (KDFParams, error) {
	if p, err := loadKDFSecret(dataDir, masterKey); err == nil {
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

// VerifyVaultParamsHash checks whether the vault params hash stored inside
// the encrypted .secret matches the given VaultParams.  Returns nil if the
// hash matches (or if the .secret pre-dates the hash field).
func VerifyVaultParamsHash(dataDir string, masterKey []byte, vp VaultParams) error {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return fmt.Errorf("reading secret: %w", err)
	}
	plaintext, err := DecryptAES256GCM(b, masterKey)
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
	expected, err := HashVaultParams(vp)
	if err != nil {
		return err
	}
	if sf.VaultParamsHash != expected {
		return fmt.Errorf("vault params hash mismatch: .vault_params may have been tampered with")
	}
	return nil
}

func loadKDFSecret(dataDir string, masterKey []byte) (KDFParams, error) {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return KDFParams{}, err
	}
	plaintext, err := DecryptAES256GCM(b, masterKey)
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

	p := KDFParams{Salt: sf.Salt, Time: sf.Time, MemoryKB: sf.MemoryKB, Threads: sf.Threads}
	ensureKDFParams(&p)
	return p, nil
}

func writeKDFSecretAtomic(dataDir string, p KDFParams, masterKey []byte, vaultParamsHash string) error {
	ensureKDFParams(&p)
	if len(p.Salt) != 16 {
		s := make([]byte, 16)
		_, _ = rand.Read(s)
		p.Salt = s
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
	}
	b, _ := json.MarshalIndent(sf, "", "  ")

	ciphertext, err := EncryptAES256GCM(b, masterKey)
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

// WriteSecretWithParams writes a new .secret file that includes the
// SHA-256 hash of the given VaultParams.  Use this when changing the
// master password or re-keying the vault.
func WriteSecretWithParams(dataDir string, vp VaultParams, masterKey []byte) error {
	hash, err := HashVaultParams(vp)
	if err != nil {
		return err
	}
	return writeKDFSecretAtomic(dataDir, KDFParams{}, masterKey, hash)
}

// writeSecretWithParams is the internal alias.
var writeSecretWithParams = WriteSecretWithParams

// ReKeyVault writes a new .secret file with a fresh salt, encrypted with the
// new master key. Call ReKeyEntries separately to re-encrypt vault data.
// Deprecated: prefer writeSecretWithParams which also embeds the hash.
func ReKeyVault(dataDir string, newMasterKey []byte) error {
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, newMasterKey, ""); err != nil {
		return fmt.Errorf("writing new secret: %w", err)
	}
	return nil
}

// ReKeyEntries decrypts all .pb entries and attachments with oldKey, then
// re-encrypts them in place with newKey.
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

func EncryptAES256GCM(plaintext, key []byte) ([]byte, error) {
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

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func DecryptAES256GCM(ciphertext, key []byte) ([]byte, error) {
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
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
