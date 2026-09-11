package ui

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func totpQRURL(secret string) string {
	return fmt.Sprintf("otpauth://totp/PassBook:v?secret=%s&issuer=PassBook", secret)
}

func renderQRCode(url string) (string, int, int) {
	qr, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return "", 0, 0
	}
	qr.DisableBorder = true

	bmp := qr.Bitmap()
	rows := len(bmp)
	if rows == 0 {
		return "", 0, 0
	}
	cols := len(bmp[0])
	displayCols := (cols + 1) / 2

	var buf strings.Builder
	lines := 0

	for y := 0; y < rows; y += 4 {
		for x := 0; x < cols; x += 2 {
			ch := brailleAt(bmp, x, y)
			if ch == '\u2800' {
				buf.WriteRune(' ')
			} else {
				buf.WriteString(qrModuleStyle.Render(string(ch)))
			}
		}
		buf.WriteString("\n")
		lines++
	}

	return buf.String(), lines, displayCols
}

func brailleAt(bmp [][]bool, x, y int) rune {
	set := func(bit, bx, by int, bits *rune) {
		if qrBitmapAt(bmp, bx, by) {
			*bits |= 1 << bit
		}
	}

	var bits rune
	set(0, x, y, &bits)
	set(1, x, y+1, &bits)
	set(2, x, y+2, &bits)
	set(3, x+1, y, &bits)
	set(4, x+1, y+1, &bits)
	set(5, x+1, y+2, &bits)
	set(6, x, y+3, &bits)
	set(7, x+1, y+3, &bits)
	return '\u2800' + bits
}

func qrBitmapAt(bmp [][]bool, x, y int) bool {
	if y < 0 || y >= len(bmp) {
		return false
	}
	row := bmp[y]
	if x < 0 || x >= len(row) {
		return false
	}
	return row[x]
}

func formatTotpSecret(secret string) string {
	return strings.ToUpper(secret)
}
