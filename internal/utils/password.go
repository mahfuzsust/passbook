package utils

import (
	"crypto/rand"
	"math/big"
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
