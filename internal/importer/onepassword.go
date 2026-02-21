package importer

import (
	"encoding/json"
	"fmt"
	"os"

	"passbook/internal/config"
	"passbook/internal/pb"
)

// onePasswordExport represents the top-level 1Password JSON export.
type onePasswordExport struct {
	Accounts []onePasswordAccount `json:"accounts"`
}

type onePasswordAccount struct {
	Vaults []onePasswordVault `json:"vaults"`
}

type onePasswordVault struct {
	Items []onePasswordItem `json:"items"`
}

type onePasswordItem struct {
	Title    string               `json:"title"`
	Category string               `json:"categoryUuid"`
	URLs     []onePasswordURL     `json:"urls"`
	Fields   []onePasswordField   `json:"fields"`
	Sections []onePasswordSection `json:"sections"`
	Notes    string               `json:"notesPlain"`
}

type onePasswordURL struct {
	URL string `json:"url"`
}

type onePasswordField struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Designation string `json:"designation"`
	Type        string `json:"type"`
	ID          string `json:"id"`
}

type onePasswordSection struct {
	Title  string             `json:"title"`
	Fields []onePasswordField `json:"fields"`
}

// Import1Password reads a 1Password JSON export (.1pux) and creates encrypted entries.
func Import1Password(jsonPath, masterPassword string, cfg config.AppConfig) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var export onePasswordExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	var entries []*pb.Entry
	var subDirs, names []string

	for _, acct := range export.Accounts {
		for _, vault := range acct.Vaults {
			for _, item := range vault.Items {
				entry, subDir := convert1PasswordItem(item)
				entries = append(entries, entry)
				subDirs = append(subDirs, subDir)
				names = append(names, item.Title)
			}
		}
	}

	return saveEntries(entries, subDirs, names, masterPassword, cfg)
}

func convert1PasswordItem(item onePasswordItem) (*pb.Entry, string) {
	switch item.Category {
	case "001": // Login
		entry := &pb.Entry{
			Type:       "Login",
			Title:      item.Title,
			CustomText: item.Notes,
		}

		for _, f := range item.Fields {
			switch f.Designation {
			case "username":
				entry.Username = f.Value
			case "password":
				entry.Password = f.Value
			}
		}

		// Look for TOTP in sections.
		for _, sec := range item.Sections {
			for _, f := range sec.Fields {
				if f.Type == "OTP" || f.ID == "TOTP" || f.Name == "one-time password" {
					entry.TotpSecret = f.Value
				}
			}
		}

		if len(item.URLs) > 0 {
			entry.Link = item.URLs[0].URL
		}

		extra := collect1PasswordExtraFields(item)
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(extra))
		return entry, "logins"

	case "002": // Credit Card
		entry := &pb.Entry{
			Type:       "Card",
			Title:      item.Title,
			CustomText: item.Notes,
		}

		var cardholder string
		for _, f := range item.Fields {
			switch f.Name {
			case "ccnum":
				entry.CardNumber = f.Value
			case "cvv":
				entry.Cvv = f.Value
			case "expiry":
				entry.Expiry = f.Value
			case "cardholder":
				cardholder = f.Value
			}
		}
		// Also check sections for card fields.
		for _, sec := range item.Sections {
			for _, f := range sec.Fields {
				switch f.Name {
				case "ccnum":
					if entry.CardNumber == "" {
						entry.CardNumber = f.Value
					}
				case "cvv":
					if entry.Cvv == "" {
						entry.Cvv = f.Value
					}
				case "expiry":
					if entry.Expiry == "" {
						entry.Expiry = f.Value
					}
				case "cardholder":
					if cardholder == "" {
						cardholder = f.Value
					}
				}
			}
		}
		if cardholder != "" {
			entry.CustomText = appendNotes(entry.CustomText, "Cardholder: "+cardholder)
		}

		extra := collect1PasswordExtraFields(item)
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(extra))
		return entry, "cards"

	case "003": // Secure Note
		entry := &pb.Entry{
			Type:       "Note",
			Title:      item.Title,
			CustomText: item.Notes,
		}

		extra := collect1PasswordExtraFields(item)
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(extra))
		return entry, "notes"

	case "006": // Document
		entry := &pb.Entry{
			Type:       "Note",
			Title:      item.Title,
			CustomText: item.Notes,
		}

		extra := collect1PasswordExtraFields(item)
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(extra))
		return entry, "notes"

	default:
		// Map any other category as a Note to avoid losing data.
		if item.Title == "" {
			return nil, ""
		}
		entry := &pb.Entry{
			Type:       "Note",
			Title:      item.Title,
			CustomText: item.Notes,
		}
		extra := collect1PasswordExtraFields(item)
		entry.CustomText = appendNotes(entry.CustomText, formatCustomFields(extra))
		return entry, "notes"
	}
}

// collect1PasswordExtraFields gathers non-designation fields from sections
// that aren't already mapped to standard entry fields.
func collect1PasswordExtraFields(item onePasswordItem) [][2]string {
	skip := map[string]bool{
		"username": true, "password": true,
		"ccnum": true, "cvv": true, "expiry": true, "cardholder": true,
	}

	var fields [][2]string
	for _, sec := range item.Sections {
		for _, f := range sec.Fields {
			if skip[f.Name] || skip[f.Designation] {
				continue
			}
			if f.Type == "OTP" || f.ID == "TOTP" || f.Name == "one-time password" {
				continue
			}
			if f.Value == "" {
				continue
			}
			label := f.Name
			if label == "" {
				label = f.ID
			}
			fields = append(fields, [2]string{label, f.Value})
		}
	}
	return fields
}
