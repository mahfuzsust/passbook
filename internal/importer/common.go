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

// saveEntries encrypts and writes a slice of entries to the vault.
func saveEntries(entries []*pb.Entry, subDirs []string, names []string, masterPassword string, cfg config.AppConfig) error {
	dataDir := config.ExpandPath(cfg.DataDir)

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	var encKey []byte
	if cfg.IsMigrated {
		rootSalt, err := crypto.LoadRootSalt(dataDir)
		if err != nil || len(rootSalt) == 0 {
			return fmt.Errorf("failed to load root salt: vault may not be migrated")
		}
		masterKey, vaultKey, err := crypto.DeriveKeys(masterPassword, rootSalt)
		if err != nil {
			return fmt.Errorf("key derivation error: %w", err)
		}
		if _, err := crypto.EnsureKDFSecret(dataDir, masterKey); err != nil {
			return fmt.Errorf("wrong master password or vault error: %w", err)
		}
		encKey = vaultKey

		// --- BEGIN supportLegacy ---
	} else if crypto.SupportLegacy() {
		masterKey := crypto.DeriveMasterKey(masterPassword)
		kdfParams, err := crypto.EnsureKDFSecret(dataDir, masterKey)
		if err != nil {
			return fmt.Errorf("wrong master password or vault error: %w", err)
		}
		encKey = crypto.DeriveKey(masterPassword, kdfParams)
		// --- END supportLegacy ---

	} else {
		return fmt.Errorf("vault not migrated and legacy support is disabled")
	}

	var imported, skipped int
	for i, entry := range entries {
		if entry == nil {
			skipped++
			continue
		}

		subDir := subDirs[i]
		origName := names[i]

		fullDir := filepath.Join(dataDir, subDir)
		if err := os.MkdirAll(fullDir, 0700); err != nil {
			return fmt.Errorf("creating dir %s: %w", subDir, err)
		}

		title := sanitizeTitle(entry.Title)
		if title == "" {
			title = "Untitled"
		}
		entry.Title = title

		filename := title + ".pb"
		path := filepath.Join(fullDir, filename)

		// Handle duplicate titles by appending a suffix.
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			counter := 1
			for {
				path = filepath.Join(fullDir, fmt.Sprintf("%s_%d.pb", title, counter))
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
