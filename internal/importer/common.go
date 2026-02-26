package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"passbook/internal/config"
	"passbook/internal/crypto"
	"passbook/internal/pb"

	"google.golang.org/protobuf/proto"
)

// saveEntries encrypts and writes a slice of entries to the vault root.
func saveEntries(entries []*pb.Entry, names []string, masterPassword string, cfg config.AppConfig) error {
	dataDir := config.ExpandPath(cfg.DataDir)

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	var encKey []byte
	vaultParams, err := crypto.LoadVaultParams(dataDir)
	if err != nil || vaultParams == nil {
		return fmt.Errorf("failed to load vault params: vault may not be initialized")
	}
	masterKey, vaultKey, err := crypto.DeriveKeys(masterPassword, vaultParams)
	if err != nil {
		return fmt.Errorf("key derivation error: %w", err)
	}
	if _, err := crypto.EnsureSecret(dataDir, masterKey, vaultParams); err != nil {
		crypto.WipeBytes(masterKey)
		crypto.WipeBytes(vaultKey)
		return fmt.Errorf("wrong master password or vault error: %w", err)
	}
	crypto.WipeBytes(masterKey)
	encKey = vaultKey
	defer crypto.WipeBytes(encKey)

	var imported, skipped int
	for i, entry := range entries {
		if entry == nil {
			skipped++
			continue
		}

		origName := names[i]

		title := sanitizeTitle(entry.Title)
		if title == "" {
			title = "Untitled"
		}
		entry.Title = title

		filename := title + ".pb"
		path := filepath.Join(dataDir, filename)

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			counter := 1
			for {
				path = filepath.Join(dataDir, fmt.Sprintf("%s_%d.pb", title, counter))
				if _, err := os.Stat(path); os.IsNotExist(err) {
					break
				}
				counter++
			}
		}

		bytes, err := proto.Marshal(entry)
		if err != nil {
			fmt.Printf("  ⚠ skipping %q: marshal error: %v\n", origName, err)
			skipped++
			continue
		}

		enc, err := crypto.Encrypt(encKey, bytes)
		if err != nil {
			fmt.Printf("  ⚠ skipping %q: encrypt error: %v\n", origName, err)
			skipped++
			continue
		}

		if err := os.WriteFile(path, enc, 0600); err != nil {
			fmt.Printf("  ⚠ skipping %q: write error: %v\n", origName, err)
			skipped++
			continue
		}

		imported++
	}

	total := len(entries)
	fmt.Printf("Import complete: %d imported, %d skipped (total %d items)\n",
		imported, skipped, total)
	return nil
}

func sanitizeTitle(title string) string {
	title = strings.TrimSpace(title)
	replacer := strings.NewReplacer(
		"<", "", ">", "", ":", "", "\"", "", "/", "-",
		"\\", "-", "|", "", "?", "", "*", "",
	)
	title = replacer.Replace(title)
	if title == "." || title == ".." {
		title = "entry"
	}
	return title
}

func formatCustomFields(fields [][2]string) string {
	if len(fields) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fields {
		if f[0] != "" || f[1] != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", f[0], f[1]))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Custom Fields:\n" + strings.Join(parts, "\n")
}

func appendNotes(existing, extra string) string {
	if extra == "" {
		return existing
	}
	if existing != "" {
		return existing + "\n\n" + extra
	}
	return extra
}
