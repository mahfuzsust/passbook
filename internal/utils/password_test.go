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
