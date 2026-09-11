package ui

import (
	"fmt"
	"strconv"
	"strings"
)

func collectCardFields(e editorModel) (string, string, string) {
	return strings.TrimSpace(e.cardNumber.Value()),
		strings.TrimSpace(e.expiry.Value()),
		strings.TrimSpace(e.cvv.Value())
}

func validateCardFieldValues(number, expiry, cvv string, entryType EntryType) error {
	if entryType != TypeCard {
		return nil
	}

	number = strings.TrimSpace(number)
	expiry = strings.TrimSpace(expiry)
	cvv = strings.TrimSpace(cvv)

	if number != "" {
		if len(number) < 13 || len(number) > 19 || !isDigits(number) {
			return fmt.Errorf("card number must be 13-19 digits")
		}
	}

	if expiry != "" {
		if len(expiry) != 5 || expiry[2] != '/' {
			return fmt.Errorf("expiry must be MM/YY")
		}
		mm, yy := expiry[:2], expiry[3:]
		if !isDigits(mm) || !isDigits(yy) {
			return fmt.Errorf("expiry must be MM/YY")
		}
		month, _ := strconv.Atoi(mm)
		if month < 1 || month > 12 {
			return fmt.Errorf("expiry must be MM/YY")
		}
	}

	if cvv != "" {
		if (len(cvv) != 3 && len(cvv) != 4) || !isDigits(cvv) {
			return fmt.Errorf("CVV must be 3 or 4 digits")
		}
	}

	return nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
