package crypto

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

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

func secretPath(dataDir string) string {
	return filepath.Join(dataDir, ".secret")
}

func EnsureKDFSecret(dataDir string) (KDFParams, error) {
	if p, err := loadKDFSecret(dataDir); err == nil {
		return p, nil
	}
	if err := writeKDFSecretAtomic(dataDir, KDFParams{}); err != nil {
		return KDFParams{}, err
	}
	return loadKDFSecret(dataDir)
}

func loadKDFSecret(dataDir string) (KDFParams, error) {
	b, err := os.ReadFile(secretPath(dataDir))
	if err != nil {
		return KDFParams{}, err
	}
	var sf secretFile
	if err := json.Unmarshal(b, &sf); err != nil {
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

func writeKDFSecretAtomic(dataDir string, p KDFParams) error {
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

	vaultDir := filepath.Dir(secretPath(dataDir))
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		return err
	}

	tmp := secretPath(dataDir) + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, secretPath(dataDir)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func ensureKDFParams(p *KDFParams) {
	if p.Time == 0 {
		p.Time = 3
	}
	if p.MemoryKB == 0 {
		p.MemoryKB = 64 * 1024
	}
	if p.Threads == 0 {
		p.Threads = 2
	}
}
