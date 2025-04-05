package utils

import (
	"github.com/pquerna/otp/totp"
	"time"
)

func GenerateTOTP(secret string) string {
	totpCode, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "Error"
	}
	return totpCode
}
