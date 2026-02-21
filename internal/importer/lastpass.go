package importer

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"passbook/internal/config"
	"passbook/internal/pb"
)

// ImportLastPass reads a LastPass CSV export and creates encrypted entries.
func ImportLastPass(csvPath, masterPassword string, cfg config.AppConfig) error {
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or has no data rows")
	}

	header := records[0]
	colIndex := buildColumnIndex(header)

	var entries []*pb.Entry
	var subDirs, names []string

	for _, row := range records[1:] {
		entry, subDir := convertLastPassRow(row, colIndex)
		entries = append(entries, entry)
		subDirs = append(subDirs, subDir)
		names = append(names, colVal(row, colIndex, "name"))
	}

	return saveEntries(entries, subDirs, names, masterPassword, cfg)
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

func convertLastPassRow(row []string, colIndex map[string]int) (*pb.Entry, string) {
	name := colVal(row, colIndex, "name")
	url := colVal(row, colIndex, "url")
	username := colVal(row, colIndex, "username")
	password := colVal(row, colIndex, "password")
	totp := colVal(row, colIndex, "totp")
	extra := colVal(row, colIndex, "extra")
	grouping := colVal(row, colIndex, "grouping")

	if name == "" && username == "" && password == "" && extra == "" {
		return nil, ""
	}

	// LastPass uses grouping to identify Secure Notes.
	if isLastPassSecureNote(url, grouping) {
		entry := &pb.Entry{
			Type:       "Note",
			Title:      name,
			CustomText: extra,
		}
		return entry, "notes"
	}

	// Default: treat as Login.
	entry := &pb.Entry{
		Type:       "Login",
		Title:      name,
		Username:   username,
		Password:   password,
		Link:       url,
		TotpSecret: totp,
		CustomText: extra,
	}

	if name == "" && url != "" {
		entry.Title = url
	}

	return entry, "logins"
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
