package importer

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"passbook/internal/config"
	"passbook/internal/store"
)

func ImportLastPass(csvPath, masterPassword string, cfg config.AppConfig) error {
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)

	reader := csv.NewReader(f)
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}

	if len(records) < 2 {
		return nil
	}

	header := records[0]
	colIndex := buildColumnIndex(header)

	var entries []*store.EntryFull
	var names []string

	for _, row := range records[1:] {
		entry := convertLastPassRow(row, colIndex)
		entries = append(entries, entry)
		names = append(names, colVal(row, colIndex, "name"))
	}

	return saveEntries(entries, names, masterPassword, cfg)
}

func buildColumnIndex(header []string) map[string]int {
	idx := make(map[string]int)
	for i, col := range header {
		idx[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return idx
}

func colVal(row []string, idx map[string]int, col string) string {
	if i, ok := idx[col]; ok && i < len(row) {
		return row[i]
	}
	return ""
}

func convertLastPassRow(row []string, colIndex map[string]int) *store.EntryFull {
	name := colVal(row, colIndex, "name")
	url := colVal(row, colIndex, "url")
	username := colVal(row, colIndex, "username")
	password := colVal(row, colIndex, "password")
	totpVal := colVal(row, colIndex, "totp")
	extra := colVal(row, colIndex, "extra")
	grouping := colVal(row, colIndex, "grouping")

	if name == "" && username == "" && password == "" && extra == "" {
		return nil
	}

	if isLastPassSecureNote(url, grouping) {
		return &store.EntryFull{
			Type:       "Note",
			Title:      name,
			CustomText: extra,
		}
	}

	entry := &store.EntryFull{
		Type:       "Login",
		Title:      name,
		Username:   username,
		Password:   password,
		Link:       url,
		TotpSecret: totpVal,
		CustomText: extra,
	}

	if name == "" && url != "" {
		entry.Title = url
	}

	return entry
}

func isLastPassSecureNote(url, grouping string) bool {
	if url == "http://sn" {
		return true
	}
	if strings.EqualFold(grouping, "Secure Notes") {
		return true
	}
	return false
}
