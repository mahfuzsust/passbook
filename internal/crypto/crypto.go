package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

type KDFParams struct {
	Salt     []byte
	Time     uint32
	MemoryKB uint32
	Threads  uint8
}

type secretFile struct {
	Version    int    `json:"version"`
	Salt       []byte `json:"salt"`
	Time       uint32 `json:"time"`
	MemoryKB   uint32 `json:"memory_kb"`
	Threads    uint8  `json:"threads"`
	KeyLen     uint32 `json:"key_len"`
	KDF        string `json:"kdf"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	VaultID    string `json:"vault_id,omitempty"`
	Reserved01 string `json:"reserved01,omitempty"`
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

// DeriveRootKey derives a root key from the password and a random salt using Argon2id.
func DeriveRootKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		6,
		256*1024,
		4,
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
// vault key (for entry encryption) from a password and random salt using
// the new HKDF-based scheme:
//
//	root_key   = Argon2id(password, random_salt)
//	master_key = HKDF(root_key, "master")
//	vault_key  = HKDF(root_key, "vault")
func DeriveKeys(password string, salt []byte) (masterKey, vaultKey []byte, err error) {
	rootKey := DeriveRootKey(password, salt)
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

// --- BEGIN supportLegacy ---

// MigrateVault migrates a vault from the legacy fixed-salt scheme to the new
// HKDF-based scheme. It:
//  1. Derives the old keys using the legacy scheme
//  2. Generates a new random salt
//  3. Derives new keys using the HKDF scheme
//  4. Re-encrypts the .secret file with the new master key
//  5. Re-encrypts all entries with the new vault key
//
// Returns the new salt so the caller can persist it in config.
func MigrateVault(dataDir string, password string) (newSalt []byte, err error) {
	// Derive old keys using legacy scheme.
	oldMasterKey := DeriveLegacyMasterKey(password)
	oldKDF, err := loadKDFSecret(dataDir, oldMasterKey)
	if err != nil {
		return nil, fmt.Errorf("loading legacy secret (wrong password?): %w", err)
	}
	oldVaultKey := DeriveKey(password, oldKDF)

	// Generate new random salt.
	newSalt, err = GenerateRootSalt()
	if err != nil {
		return nil, err
	}

	// Derive new keys using HKDF scheme.
	newMasterKey, newVaultKey, err := DeriveKeys(password, newSalt)
	if err != nil {
		return nil, fmt.Errorf("deriving new keys: %w", err)
	}

	// Re-encrypt .secret with the new master key.
	if err := ReKeyVault(dataDir, newMasterKey); err != nil {
		return nil, fmt.Errorf("re-keying vault secret: %w", err)
	}

	// Re-encrypt all entries and attachments.
	if err := ReKeyEntries(dataDir, oldVaultKey, newVaultKey); err != nil {
		return nil, fmt.Errorf("re-keying entries: %w", err)
	}

	return newSalt, nil
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

func EnsureKDFSecret(dataDir string, masterKey []byte) (KDFParams, error) {
	if p, err := loadKDFSecret(dataDir, masterKey); err == nil {
		return p, nil
	}
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, masterKey); err != nil {
		return KDFParams{}, err
	}
	return loadKDFSecret(dataDir, masterKey)
}

func loadKDFSecret(dataDir string, masterKey []byte) (KDFParams, error) {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return KDFParams{}, err
	}
	// Decrypt the file content
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

func writeKDFSecretAtomic(dataDir string, p KDFParams, masterKey []byte) error {
	ensureKDFParams(&p)
	if len(p.Salt) != 16 {
		s := make([]byte, 16)
		_, _ = rand.Read(s)
		p.Salt = s
	}
	sf := secretFile{
		Version:  1,
		Salt:     p.Salt,
		Time:     p.Time,
		MemoryKB: p.MemoryKB,
		Threads:  p.Threads,
		KeyLen:   32,
		KDF:      "argon2id",
	}
	b, _ := json.MarshalIndent(sf, "", "  ")

	// Encrypt the JSON data
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

// ReKeyVault writes a new .secret file with a fresh salt, encrypted with the
// new master key. Call ReKeyEntries separately to re-encrypt vault data.
func ReKeyVault(dataDir string, newMasterKey []byte) error {
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}, newMasterKey); err != nil {
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
