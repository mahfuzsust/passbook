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

func TestPasswordStrengthWeak(t *testing.T) {
	score, level, _ := PasswordStrength("aaa")
	if level != StrengthWeak {
		t.Fatalf("expected Weak for 'aaa', got level=%d score=%d", level, score)
	}
}

func TestPasswordStrengthFair(t *testing.T) {
	score, level, _ := PasswordStrength("abcd12")
	if level != StrengthFair {
		t.Fatalf("expected Fair for 'abcd12', got level=%d score=%d", level, score)
	}
}

func TestPasswordStrengthGood(t *testing.T) {
	score, level, _ := PasswordStrength("Hello123!world")
	if level < StrengthGood {
		t.Fatalf("expected at least Good for 'Hello123!world', got level=%d score=%d", level, score)
	}
}

func TestPasswordStrengthStrong(t *testing.T) {
	score, level, _ := PasswordStrength("C0mpl3x!P@ssw0rd#2025xyz")
	if level != StrengthStrong {
		t.Fatalf("expected Strong, got level=%d score=%d", level, score)
	}
}

func TestPasswordStrengthCommonWordPenalty(t *testing.T) {
	s1, _, _ := PasswordStrength("password123!")
	s2, _, _ := PasswordStrength("xkfj8m3q12!!")
	if s1 >= s2 {
		t.Fatalf("expected common word 'password' to score lower: %d >= %d", s1, s2)
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
	}{
		{"", StrengthEmpty},
		{"ab", StrengthWeak},
		{"Abcdef1!", StrengthFair},
		{"MyP@ss2025abcdef", StrengthGood},
		{"X#9kLp!mZ@4wQr7&vN2yB", StrengthStrong},
	}
	for _, tt := range tests {
		_, level, _ := PasswordStrength(tt.password)
		if level < tt.minLevel {
			t.Errorf("PasswordStrength(%q): level=%d, want >= %d", tt.password, level, tt.minLevel)
		}
	}
}
