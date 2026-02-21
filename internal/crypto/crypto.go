package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
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

func DeriveKey(password string, p KDFParams) []byte {
	return argon2.IDKey([]byte(password), p.Salt, p.Time, p.MemoryKB, p.Threads, 32)
}

func DeriveMasterKey(masterPassword string) []byte {
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
