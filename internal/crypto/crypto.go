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
	"strings"

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

func LoadVaultParams(dataDir string) (*VaultParams, error) {
	p, err := loadVaultParamsFrom(vaultParamsPath(dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
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

	if p.MasterKeyPurpose == "" || p.VaultKeyPurpose == "" {
		return nil, nil, errors.New("vault params missing HKDF purpose strings")
	}

	masterKey, err = DeriveHKDFKey(rootKey, p.MasterKeyPurpose)
	if err != nil {
		return nil, nil, err
	}
	vaultKey, err = DeriveHKDFKey(rootKey, p.VaultKeyPurpose)
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
	if err := WriteSecretWithParams(dataDir, newParams, newMasterKey); err != nil {
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

func VaultHasEntries(dataDir string) bool {
	found := false
	_ = filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == dataDir {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) == ".pb" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
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

func SecretExists(dataDir string) bool {
	_, err := os.Stat(secretPath(dataDir))
	return err == nil
}

// VerifyMasterKey attempts to decrypt .secret with the given master key.
// Returns nil on success, or an error if decryption fails (wrong password).
func VerifyMasterKey(dataDir string, masterKey []byte) error {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return err
	}
	aad := vaultParamsAAD(dataDir)
	_, err = DecryptAES256GCM(b, masterKey, aad)
	return err
}

func vaultParamsAAD(dataDir string) []byte {
	data, err := os.ReadFile(vaultParamsPath(dataDir))
	if err != nil {
		return nil
	}
	h := HashVaultParamsBytes(data)
	return []byte(h)
}

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

func VerifyVaultParamsHash(dataDir string, masterKey []byte) error {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return fmt.Errorf("reading secret: %w", err)
	}

	aad := vaultParamsAAD(dataDir)
	plaintext, err := DecryptAES256GCM(b, masterKey, aad)
	if err != nil {
		return fmt.Errorf("decrypting secret: %w", err)
	}
	var sf secretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		return fmt.Errorf("parsing secret: %w", err)
	}
	if sf.VaultParamsHash == "" {
		return fmt.Errorf("vault params hash missing from .secret")
	}

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
// alongside it. Returns ErrWrongPassword if the tag does not match.
func VerifyCommitTag(dataDir string, masterKey []byte) error {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return fmt.Errorf("reading secret: %w", err)
	}

	aad := vaultParamsAAD(dataDir)
	plaintext, err := DecryptAES256GCM(b, masterKey, aad)
	if err != nil {
		return fmt.Errorf("decrypting secret: %w", err)
	}

	var sf secretFile
	if err := json.Unmarshal(plaintext, &sf); err != nil {
		return fmt.Errorf("parsing secret: %w", err)
	}

	if sf.CommitTag == "" || len(sf.CommitNonce) == 0 {
		return fmt.Errorf("commit tag or nonce missing from .secret")
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

	aad := vaultParamsAAD(dataDir)
	plaintext, err := DecryptAES256GCM(b, masterKey, aad)
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

	if sf.CommitTag == "" || len(sf.CommitNonce) == 0 {
		return KDFParams{}, errors.New("commit tag or nonce missing from .secret")
	}
	expected := ComputeCommitTag(masterKey, sf.CommitNonce)
	if !hmac.Equal([]byte(sf.CommitTag), []byte(expected)) {
		return KDFParams{}, ErrWrongPassword
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
	aad := []byte(vaultParamsHash)
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

func ReKeyVault(dataDir string, newMasterKey []byte) error {
	hash := vaultParamsAADHex(dataDir)
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, newMasterKey, hash); err != nil {
		return fmt.Errorf("writing new secret: %w", err)
	}
	return nil
}

// vaultParamsAADHex returns the hex SHA-256 hash of the raw .vault_params
// file on disk, suitable for passing to writeKDFSecretAtomic.
func vaultParamsAADHex(dataDir string) string {
	b, err := os.ReadFile(vaultParamsPath(dataDir))
	if err != nil {
		return ""
	}
	return HashVaultParamsBytes(b)
}

func ReKeyEntries(dataDir string, oldKey, newKey []byte) error {
	// Re-encrypt all .pb entry files (in root and any user-created folders).
	err := filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == dataDir {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) == ".pb" {
			if err := reEncryptFile(path, oldKey, newKey); err != nil {
				return fmt.Errorf("re-encrypting %s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Re-encrypt all attachment blobs.
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
