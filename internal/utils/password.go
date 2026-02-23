package utils

import (
	"crypto/rand"
	"math/big"
	"strings"
	"unicode"
)

func GeneratePassword(length int, useUpper, useLower, useSpecial bool) string {
	charset := ""
	if useUpper {
		charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	if useLower {
		charset += "abcdefghijklmnopqrstuvwxyz"
	}
	if useSpecial {
		charset += "!@#$%^&*()-_=+[]{}|;:,.<>?"
	}
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	}

	pass := make([]byte, length)
	for i := range pass {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		pass[i] = charset[num.Int64()]
	}
	return string(pass)
}

// StrengthLevel represents the overall password strength.
type StrengthLevel int

const (
	StrengthEmpty  StrengthLevel = iota // ""
	StrengthWeak                        // red
	StrengthFair                        // orange/yellow
	StrengthGood                        // blue
	StrengthStrong                      // green
)

// PasswordStrength evaluates a password and returns a score (0–100),
// a StrengthLevel, and a human-readable label.
func PasswordStrength(password string) (score int, level StrengthLevel, label string) {
	if password == "" {
		return 0, StrengthEmpty, ""
	}

	n := len(password)

	// --- length score (0–40) ---
	lengthScore := 0
	switch {
	case n >= 20:
		lengthScore = 40
	case n >= 16:
		lengthScore = 35
	case n >= 12:
		lengthScore = 25
	case n >= 8:
		lengthScore = 15
	case n >= 6:
		lengthScore = 5
	}

	// --- character variety (0–40) ---
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	variety := 0
	if hasLower {
		variety++
	}
	if hasUpper {
		variety++
	}
	if hasDigit {
		variety++
	}
	if hasSpecial {
		variety++
	}
	varietyScore := variety * 10

	// --- uniqueness bonus (0–20) ---
	unique := make(map[rune]bool)
	for _, r := range password {
		unique[r] = true
	}
	ratio := float64(len(unique)) / float64(n)
	uniqueScore := 0
	switch {
	case ratio >= 0.9:
		uniqueScore = 20
	case ratio >= 0.7:
		uniqueScore = 15
	case ratio >= 0.5:
		uniqueScore = 10
	case ratio >= 0.3:
		uniqueScore = 5
	}

	// --- penalty for common patterns ---
	penalty := 0
	lower := strings.ToLower(password)
	commonWords := []string{"password", "123456", "qwerty", "admin", "letmein", "welcome", "monkey", "master", "passbook"}
	for _, w := range commonWords {
		if strings.Contains(lower, w) {
			penalty += 20
			break
		}
	}
	// Sequential characters penalty (abc, 123, etc.)
	seq := 0
	runes := []rune(password)
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1]+1 || runes[i] == runes[i-1]-1 {
			seq++
		}
	}
	if seq > 3 {
		penalty += 10
	}

	score = lengthScore + varietyScore + uniqueScore - penalty
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	switch {
	case score >= 70:
		level = StrengthStrong
		label = "Strong"
	case score >= 50:
		level = StrengthGood
		label = "Good"
	case score >= 30:
		level = StrengthFair
		label = "Fair"
	default:
		level = StrengthWeak
		label = "Weak"
	}

	return score, level, label
}
