package ui

import (
	"strings"
	"testing"

	"github.com/pquerna/otp/totp"
)

func TestRenderQRCodeBraille(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "PassBook",
		AccountName: "vault",
	})
	if err != nil {
		t.Fatal(err)
	}

	text, lines, cols := renderQRCode(totpQRURL(key.Secret()))
	if lines == 0 || cols == 0 {
		t.Fatal("expected qr output")
	}
	if !strings.Contains(text, "[black") {
		t.Fatal("expected dark modules in qr")
	}
	if lines > 12 {
		t.Fatalf("expected compact braille height, got %d lines", lines)
	}
	if cols > 20 {
		t.Fatalf("expected compact braille width, got %d cols", cols)
	}
}
