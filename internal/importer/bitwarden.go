package importer

import (
	"encoding/json"
	"fmt"
	"os"

	"passbook/internal/config"
	"passbook/internal/pb"
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

	var entries []*pb.Entry
	var names []string

	for _, item := range export.Items {
		entry := convertBitwardenItem(item)
		entries = append(entries, entry)
		names = append(names, item.Name)
	}

	return saveEntries(entries, names, masterPassword, cfg)
}

func convertBitwardenItem(item bitwardenItem) *pb.Entry {
	fields := convertBitwardenFields(item.Fields)

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
		for _, h := range item.PasswordHistory {
			entry.History = append(entry.History, &pb.PasswordHistory{
				Password: h.Password,
				Date:     h.LastUsedDate,
			})
		}
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(fields))
		return entry

	case 2: // Secure Note
		entry := &pb.Entry{
			Type:       "Note",
			Title:      item.Name,
			CustomText: item.Notes,
		}
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(fields))
		return entry

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
				entry.CustomText = appendNotes(entry.CustomText, "Cardholder: "+item.Card.CardholderName)
			}
		}
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(fields))
		return entry

	default:
		return nil
	}
}

func convertBitwardenFields(fields []bitwardenField) [][2]string {
	var result [][2]string
	for _, f := range fields {
		result = append(result, [2]string{f.Name, f.Value})
	}
	return result
}
