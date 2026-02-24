package utils

import (
	"strings"
	"testing"
)

func TestGeneratePasswordLength(t *testing.T) {
	got := GeneratePassword(24, true, true, true)
	if len(got) != 24 {
		t.Fatalf("expected length 24, got %d", len(got))
	}
}

func TestGeneratePasswordUsesSelectedCharsets(t *testing.T) {
	upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lower := "abcdefghijklmnopqrstuvwxyz"
	special := "!@#$%^&*()-_=+[]{}|;:,.<>?"

	cases := []struct {
		name       string
		length     int
		useUpper   bool
		useLower   bool
		useSpecial bool
		allowed    string
	}{
		{
			name:       "upper+lower",
			length:     32,
			useUpper:   true,
			useLower:   true,
			useSpecial: false,
			allowed:    upper + lower,
		},
		{
			name:       "lower+special",
			length:     32,
			useUpper:   false,
			useLower:   true,
			useSpecial: true,
			allowed:    lower + special,
		},
		{
			name:       "upper+special",
			length:     32,
			useUpper:   true,
			useLower:   false,
			useSpecial: true,
			allowed:    upper + special,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GeneratePassword(tc.length, tc.useUpper, tc.useLower, tc.useSpecial)
			if len(got) != tc.length {
				t.Fatalf("expected length %d, got %d", tc.length, len(got))
			}
			for _, r := range got {
				if !strings.ContainsRune(tc.allowed, r) {
					t.Fatalf("unexpected character %q", r)
				}
			}
		})
	}
}

func TestGeneratePasswordDefaultCharset(t *testing.T) {
	allowed := "abcdefghijklmnopqrstuvwxyz0123456789"
	got := GeneratePassword(32, false, false, false)
	for _, r := range got {
		if !strings.ContainsRune(allowed, r) {
			t.Fatalf("unexpected character %q", r)
		}
	}
}

func TestGeneratePasswordZeroLength(t *testing.T) {
	got := GeneratePassword(0, true, true, true)
	if got != "" {
		t.Fatalf("expected empty password for zero length")
	}
}

func TestPasswordStrengthEmpty(t *testing.T) {
	score, level, label := PasswordStrength("")
	if score != 0 || level != StrengthEmpty || label != "" {
		t.Fatalf("expected empty result, got score=%d level=%d label=%q", score, level, label)
	}
}

func TestPasswordStrengthShortAlwaysWeak(t *testing.T) {
	// <= 8 characters must always be Weak, regardless of variety.
	shortPasswords := []string{"aaa", "Ab1!", "Xy9!@z7Q"}
	for _, pw := range shortPasswords {
		_, level, _ := PasswordStrength(pw)
		if level != StrengthWeak {
			t.Errorf("expected Weak for %q (len=%d), got level=%d", pw, len(pw), level)
		}
	}
}

func TestPasswordStrength9CharsNotAutoWeak(t *testing.T) {
	// 9 chars with good variety can escape Weak.
	_, level, _ := PasswordStrength("Abcde1!xy")
	if level == StrengthWeak {
		t.Fatalf("expected 9-char mixed password to not be Weak")
	}
}

func TestPasswordStrength12MixedIsFairOrBetter(t *testing.T) {
	// >= 12 chars with mixed characters should be Fair or better.
	_, level, _ := PasswordStrength("Abcdef12!xyz")
	if level < StrengthFair {
		t.Fatalf("expected >= 12 mixed-char password to be at least Fair, got level=%d", level)
	}
}

func TestPasswordStrength12LowercaseOnlyIsFairOrWeak(t *testing.T) {
	// 12 lowercase-only chars: limited variety, may still reach Fair via length.
	_, level, _ := PasswordStrength("abcxyzpqrstu")
	if level > StrengthFair {
		t.Fatalf("expected 12-char lowercase-only to be at most Fair, got level=%d", level)
	}
}

func TestPasswordStrengthGood(t *testing.T) {
	// 16+ chars with 3+ character classes.
	_, level, _ := PasswordStrength("MyStr0ng!Passwrd")
	if level < StrengthGood {
		t.Fatalf("expected at least Good for 16-char mixed password, got level=%d", level)
	}
}

func TestPasswordStrengthStrong(t *testing.T) {
	_, level, _ := PasswordStrength("C0mpl3x!P@ss#2025xyzABC")
	if level != StrengthStrong {
		t.Fatalf("expected Strong, got level=%d", level)
	}
}

func TestPasswordStrengthCommonWordPenalty(t *testing.T) {
	s1, _, _ := PasswordStrength("password123!abc")
	s2, _, _ := PasswordStrength("xkfj8m3q12!!abc")
	if s1 >= s2 {
		t.Fatalf("expected common word 'password' to score lower: %d >= %d", s1, s2)
	}
}

func TestPasswordStrengthRepeatedCharPenalty(t *testing.T) {
	s1, _, _ := PasswordStrength("aaaaabbbbbccccc")
	s2, _, _ := PasswordStrength("axbyczdwevfugth")
	if s1 >= s2 {
		t.Fatalf("expected repeated chars to score lower: %d >= %d", s1, s2)
	}
}

func TestPasswordStrengthScoreBounds(t *testing.T) {
	for _, pw := range []string{"a", "aB3!", "VeryL0ng&Str0ng!Pass#2025xyzABC"} {
		score, _, _ := PasswordStrength(pw)
		if score < 0 || score > 100 {
			t.Fatalf("score out of bounds for %q: %d", pw, score)
		}
	}
}

func TestPasswordStrengthLevels(t *testing.T) {
	tests := []struct {
		password string
		minLevel StrengthLevel
		maxLevel StrengthLevel
	}{
		{"", StrengthEmpty, StrengthEmpty},
		{"ab", StrengthWeak, StrengthWeak},             // <= 8 → always Weak
		{"Ab1!xyzQ", StrengthWeak, StrengthWeak},       // exactly 8 → always Weak
		{"Abcdef1!x", StrengthFair, StrengthStrong},    // 9 chars, mixed
		{"Abcdef12!xyz", StrengthFair, StrengthStrong}, // 12 chars, mixed
		{"MyP@ss2025abcdef", StrengthGood, StrengthStrong},
		{"X#9kLp!mZ@4wQr7&vN2yBc", StrengthStrong, StrengthStrong},
	}
	for _, tt := range tests {
		_, level, _ := PasswordStrength(tt.password)
		if level < tt.minLevel || level > tt.maxLevel {
			t.Errorf("PasswordStrength(%q): level=%d, want [%d..%d]", tt.password, level, tt.minLevel, tt.maxLevel)
		}
	}
}

func TestPasswordStrengthScoreCappedAt29ForShort(t *testing.T) {
	// Even with all 4 character classes, <= 8 chars caps at score 29.
	score, _, _ := PasswordStrength("Ab1!Xy9@")
	if score > 29 {
		t.Fatalf("expected score <= 29 for 8-char password, got %d", score)
	}
}
