package ui

import (
	"fmt"
	"strings"

	"passbook/internal/utils"
)

const strengthBarWidth = 15

func formatStrengthBar(password string) string {
	if password == "" {
		return ""
	}

	score, level, label := utils.PasswordStrength(password)

	var color lipglossColor
	switch level {
	case utils.StrengthWeak:
		color = "9"
	case utils.StrengthFair:
		color = "11"
	case utils.StrengthGood:
		color = "12"
	case utils.StrengthStrong:
		color = "10"
	default:
		return ""
	}

	filled := score * strengthBarWidth / 100
	if filled < 1 && score > 0 {
		filled = 1
	}
	empty := strengthBarWidth - filled

	return fmt.Sprintf("\x1b[%sm%s\x1b[90m%s\x1b[0m  \x1b[%sm%s\x1b[0m",
		ansiFg(color),
		strings.Repeat("━", filled),
		strings.Repeat("━", empty),
		ansiFg(color),
		label,
	)
}

type lipglossColor = string

func ansiFg(c lipglossColor) string {
	switch c {
	case "9":
		return "31"
	case "11":
		return "33"
	case "12":
		return "34"
	case "10":
		return "32"
	default:
		return "37"
	}
}
