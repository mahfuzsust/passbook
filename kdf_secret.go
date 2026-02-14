package main

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

func secretPath() string {
	return filepath.Join(expandPath(dataDir), ".secret")
}

func loadKDFSecret() error {
	b, err := os.ReadFile(secretPath())
	if err != nil {
		return err
	}
	var sf secretFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return err
	}
	if sf.Version <= 0 {
		return errors.New("invalid secret version")
	}
	if len(sf.Salt) != 16 {
		return errors.New("invalid salt")
	}
	kdfSalt = sf.Salt
	kdfTime = sf.Time
	kdfMemoryKB = sf.MemoryKB
	kdfThreads = sf.Threads
	if kdfTime == 0 || kdfMemoryKB == 0 || kdfThreads == 0 {
		ensureKDFParams()
	}
	return nil
}

func writeKDFSecretAtomic() error {
	ensureKDFParams()
	if len(kdfSalt) != 16 {
		s := make([]byte, 16)
		_, _ = rand.Read(s)
		kdfSalt = s
	}
	sf := secretFile{
		Version:  1,
		Salt:     kdfSalt,
		Time:     kdfTime,
		MemoryKB: kdfMemoryKB,
		Threads:  kdfThreads,
		KeyLen:   32,
		KDF:      "argon2id",
	}
	b, _ := json.MarshalIndent(sf, "", "  ")

	vaultDir := filepath.Dir(secretPath())
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		return err
	}

	tmp := secretPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, secretPath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func ensureKDFSecret() {
	if err := loadKDFSecret(); err == nil {
		return
	}
	_ = writeKDFSecretAtomic()
	_ = loadKDFSecret()
}
