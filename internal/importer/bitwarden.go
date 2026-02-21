package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"passbook/internal/config"
	"passbook/internal/crypto"
	"passbook/internal/pb"

	"google.golang.org/protobuf/proto"
)

// bitwardenExport represents the top-level Bitwarden JSON export.
type bitwardenExport struct {
	Items []bitwardenItem `json:"items"`
}

type bitwardenItem struct {
	Type            int                        `json:"type"`
	Name            string                     `json:"name"`
	Notes           string                     `json:"notes"`
	Login           *bitwardenLogin            `json:"login"`
	Card            *bitwardenCard             `json:"card"`
	Fields          []bitwardenField           `json:"fields"`
	PasswordHistory []bitwardenPasswordHistory `json:"passwordHistory"`
}

type bitwardenPasswordHistory struct {
	LastUsedDate string `json:"lastUsedDate"`
	Password     string `json:"password"`
}

type bitwardenLogin struct {
	Username string         `json:"username"`
	Password string         `json:"password"`
	Totp     string         `json:"totp"`
	URIs     []bitwardenURI `json:"uris"`
}

type bitwardenURI struct {
	URI string `json:"uri"`
}

type bitwardenCard struct {
	CardholderName string `json:"cardholderName"`
	Number         string `json:"number"`
	ExpMonth       string `json:"expMonth"`
	ExpYear        string `json:"expYear"`
	Code           string `json:"code"`
}

type bitwardenField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  int    `json:"type"`
}

// ImportBitwarden reads a Bitwarden JSON export and creates encrypted entries.
func ImportBitwarden(jsonPath, masterPassword string, cfg config.AppConfig) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var export bitwardenExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	dataDir := config.ExpandPath(cfg.DataDir)

	// Ensure directories exist.
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	// Derive encryption keys.
	masterKey := crypto.DeriveMasterKey(masterPassword)
	kdfParams, err := crypto.EnsureKDFSecret(dataDir, masterKey)
	if err != nil {
		return fmt.Errorf("wrong master password or vault error: %w", err)
	}
	encKey := crypto.DeriveKey(masterPassword, kdfParams)

	var imported, skipped int
	for _, item := range export.Items {
		entry, subDir := convertItem(item)
		if entry == nil {
			skipped++
			continue
		}

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
			fmt.Printf("  ⚠ skipping %q: marshal error: %v\n", item.Name, err)
			skipped++
			continue
		}

		enc, err := crypto.Encrypt(encKey, bytes)
		if err != nil {
			fmt.Printf("  ⚠ skipping %q: encrypt error: %v\n", item.Name, err)
			skipped++
			continue
		}

		if err := os.WriteFile(path, enc, 0600); err != nil {
			fmt.Printf("  ⚠ skipping %q: write error: %v\n", item.Name, err)
			skipped++
			continue
		}

		imported++
	}

	fmt.Printf("Import complete: %d imported, %d skipped (total %d items)\n",
		imported, skipped, len(export.Items))
	return nil
}

func convertItem(item bitwardenItem) (*pb.Entry, string) {
	switch item.Type {
	case 1: // Login
		entry := &pb.Entry{
			Type:       "Login",
			Title:      item.Name,
			CustomText: item.Notes,
		}
		if item.Login != nil {
			entry.Username = item.Login.Username
			entry.Password = item.Login.Password
			entry.TotpSecret = item.Login.Totp
			if len(item.Login.URIs) > 0 {
				entry.Link = item.Login.URIs[0].URI
			}
		}
		// Map Bitwarden password history.
		for _, h := range item.PasswordHistory {
			entry.History = append(entry.History, &pb.PasswordHistory{
				Password: h.Password,
				Date:     h.LastUsedDate,
			})
		}
		// Append custom fields to notes.
		if extra := formatFields(item.Fields); extra != "" {
			if entry.CustomText != "" {
				entry.CustomText += "\n\n"
			}
			entry.CustomText += extra
		}
		return entry, "logins"

	case 2: // Secure Note
		entry := &pb.Entry{
			Type:       "Note",
			Title:      item.Name,
			CustomText: item.Notes,
		}
		if extra := formatFields(item.Fields); extra != "" {
			if entry.CustomText != "" {
				entry.CustomText += "\n\n"
			}
			entry.CustomText += extra
		}
		return entry, "notes"

	case 3: // Card
		entry := &pb.Entry{
			Type:       "Card",
			Title:      item.Name,
			CustomText: item.Notes,
		}
		if item.Card != nil {
			entry.CardNumber = item.Card.Number
			entry.Cvv = item.Card.Code
			if item.Card.ExpMonth != "" && item.Card.ExpYear != "" {
				entry.Expiry = fmt.Sprintf("%s/%s", item.Card.ExpMonth, item.Card.ExpYear)
			} else if item.Card.ExpMonth != "" {
				entry.Expiry = item.Card.ExpMonth
			} else if item.Card.ExpYear != "" {
				entry.Expiry = item.Card.ExpYear
			}
			if item.Card.CardholderName != "" {
				if entry.CustomText != "" {
					entry.CustomText += "\n\n"
				}
				entry.CustomText += "Cardholder: " + item.Card.CardholderName
			}
		}
		if extra := formatFields(item.Fields); extra != "" {
			if entry.CustomText != "" {
				entry.CustomText += "\n\n"
			}
			entry.CustomText += extra
		}
		return entry, "cards"

	default:
		return nil, ""
	}
}

func formatFields(fields []bitwardenField) string {
	if len(fields) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fields {
		if f.Name != "" || f.Value != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", f.Name, f.Value))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Custom Fields:\n" + strings.Join(parts, "\n")
}

func sanitizeTitle(title string) string {
	title = strings.TrimSpace(title)
	// Remove characters that are invalid in filenames.
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
