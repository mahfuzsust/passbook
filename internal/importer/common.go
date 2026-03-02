package importer

import (
	"fmt"
	"path/filepath"
	"strings"

	"passbook/internal/config"
	"passbook/internal/store"
)

func saveEntries(entries []*store.EntryFull, names []string, masterPassword string, cfg config.AppConfig) error {
	dataDir := config.ExpandPath(cfg.DataDir)
	dbPath := filepath.Join(dataDir, "passbook.db")

	s, err := store.Open(dbPath, masterPassword)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()

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

		finalTitle := title
		if s.EntryExistsInFolder(0, finalTitle) {
			counter := 1
			for {
				finalTitle = fmt.Sprintf("%s_%d", title, counter)
				if !s.EntryExistsInFolder(0, finalTitle) {
					break
				}
				counter++
			}
			entry.Title = finalTitle
		}

		if _, err := s.SaveEntry(0, entry); err != nil {
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
