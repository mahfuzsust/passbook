package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func WipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func GeneratePinKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating pin key: %w", err)
	}
	return key, nil
}

func ComputePinTag(pinKey []byte, pin string) string {
	mac := hmac.New(sha256.New, pinKey)
	mac.Write([]byte("passbook:pin:" + pin))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyPinTag(pinKey []byte, pin, storedTag string) bool {
	computed := ComputePinTag(pinKey, pin)
	return hmac.Equal([]byte(computed), []byte(storedTag))
}
