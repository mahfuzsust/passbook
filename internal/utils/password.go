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
	StrengthWeak                        // red   — reject for vault creation
	StrengthFair                        // yellow
	StrengthGood                        // blue
	StrengthStrong                      // green
)

// commonPasswords is a small blocklist of the most frequently breached
// passwords and app-specific words.  Matching any of these (case-insensitive)
// applies a heavy penalty.
var commonPasswords = []string{
	"password", "123456", "12345678", "qwerty", "abc123",
	"monkey", "master", "dragon", "letmein", "login",
	"admin", "welcome", "iloveyou", "shadow", "sunshine",
	"trustno1", "passbook", "passw0rd", "football", "baseball",
}

// PasswordStrength evaluates a password and returns a score (0–100),
// a StrengthLevel, and a human-readable label.
//
// Scoring philosophy (aligned with NIST SP 800-63B):
//   - Length is the dominant factor (0–45 points).
//   - Character-class variety adds a secondary bonus (0–30 points).
//   - Uniqueness ratio rewards non-repetitive passwords (0–15 points).
//   - Penalties apply for common/breached words and long sequential runs.
//   - Hard rule: <= 8 characters can never exceed Weak.
func PasswordStrength(password string) (score int, level StrengthLevel, label string) {
	if password == "" {
		return 0, StrengthEmpty, ""
	}

	n := len([]rune(password)) // count runes, not bytes

	// --- length score (0–45) ---
	// Length is the single most important factor (NIST SP 800-63B).
	lengthScore := 0
	switch {
	case n >= 24:
		lengthScore = 45
	case n >= 20:
		lengthScore = 40
	case n >= 16:
		lengthScore = 35
	case n >= 12:
		lengthScore = 25
	case n >= 10:
		lengthScore = 15
	case n >= 8:
		lengthScore = 8
	default:
		lengthScore = 2
	}

	// --- character-class variety (0–30) ---
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
	classes := 0
	if hasLower {
		classes++
	}
	if hasUpper {
		classes++
	}
	if hasDigit {
		classes++
	}
	if hasSpecial {
		classes++
	}
	// 1 class = 5, 2 = 12, 3 = 21, 4 = 30
	varietyScore := 0
	switch classes {
	case 1:
		varietyScore = 5
	case 2:
		varietyScore = 12
	case 3:
		varietyScore = 21
	case 4:
		varietyScore = 30
	}

	// --- uniqueness bonus (0–15) ---
	unique := make(map[rune]bool)
	for _, r := range password {
		unique[r] = true
	}
	ratio := float64(len(unique)) / float64(n)
	uniqueScore := 0
	switch {
	case ratio >= 0.9:
		uniqueScore = 15
	case ratio >= 0.7:
		uniqueScore = 12
	case ratio >= 0.5:
		uniqueScore = 8
	case ratio >= 0.3:
		uniqueScore = 4
	}

	// --- penalties ---
	penalty := 0

	// Common / breached password check.
	lower := strings.ToLower(password)
	for _, w := range commonPasswords {
		if strings.Contains(lower, w) {
			penalty += 25
			break
		}
	}

	// Sequential characters penalty (abc, 123, cba, 321, …).
	seq := 0
	runes := []rune(password)
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1]+1 || runes[i] == runes[i-1]-1 {
			seq++
		}
	}
	if seq > 4 {
		penalty += 15
	} else if seq > 2 {
		penalty += 8
	}

	// Repeated-character penalty (e.g. "aaaa").
	repeats := 0
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			repeats++
		}
	}
	if repeats > 4 {
		penalty += 12
	} else if repeats > 2 {
		penalty += 5
	}

	// --- final score ---
	score = lengthScore + varietyScore + uniqueScore - penalty
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// --- hard length gate ---
	// Passwords of 8 characters or fewer are always Weak regardless of
	// character variety — length is the primary defence against brute force.
	if n <= 8 {
		if score > 29 {
			score = 29
		}
		return score, StrengthWeak, "Weak"
	}

	switch {
	case score >= 70:
		level = StrengthStrong
		label = "Strong"
	case score >= 45:
		level = StrengthGood
		label = "Good"
	case score >= 25:
		level = StrengthFair
		label = "Fair"
	default:
		level = StrengthWeak
		label = "Weak"
	}

	return score, level, label
}
