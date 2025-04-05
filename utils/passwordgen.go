package utils

import (
	"crypto/rand"
	"log"
	"math/big"
	"strings"
)

const (
	lowercase    = "abcdefghijklmnopqrstuvwxyz"
	uppercase    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers      = "0123456789"
	specialChars = "!@#$%^&*()-_=+[]{}<>?/|:;.,~"
)

func GeneratePassword(length int, useUpper, useNumbers, useSpecial bool) string {
	charSet := lowercase
	if useUpper {
		charSet += uppercase
	}
	if useNumbers {
		charSet += numbers
	}
	if useSpecial {
		charSet += specialChars
	}

	var password strings.Builder
	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
		if err != nil {
			log.Fatal("Error generating random number:", err)
		}
		password.WriteByte(charSet[randomIndex.Int64()])
	}
	return password.String()
}
