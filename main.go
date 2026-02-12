package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	app   = tview.NewApplication()
	pages = tview.NewPages()
)

func main() {
	loadConfig()
	os.MkdirAll(expandPath(dataDir), 0700)
	os.MkdirAll(getAttachmentDir(), 0700)
	lastActivity = time.Now()

	setupUI()

	// Global Input Handlers
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		lastActivity = time.Now()
		return event
	})
	app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		lastActivity = time.Now()
		return event, action
	})

	// Background Ticker
	go func() {
		for range time.Tick(1 * time.Second) {
			app.QueueUpdateDraw(func() {
				if len(masterKey) > 0 && time.Since(lastActivity) > 5*time.Minute {
					lockApp()
				} else {
					drawTOTP() // Defined in ui.go
				}
			})
		}
	}()

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// --- Shared Helper Functions ---

func generatePassword(length int, useUpper, useLower, useSpecial bool) string {
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

func downloadAttachment(att Attachment) {
	data, err := os.ReadFile(filepath.Join(getAttachmentDir(), att.ID))
	if err != nil {
		return
	}

	dec, err := decrypt(data)
	if err != nil {
		return
	}

	var downDir string
	if runtime.GOOS == "windows" {
		downDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	} else {
		home, _ := os.UserHomeDir()
		downDir = filepath.Join(home, "Downloads")
	}

	os.WriteFile(filepath.Join(downDir, att.FileName), dec, 0644)
	notifyCopied(att.FileName + " downloaded")
}

func notifyCopied(item string) {
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied![-]", item))
	go func() { time.Sleep(2 * time.Second); app.QueueUpdateDraw(func() { viewStatus.SetText("") }) }()
}

func copySensitive(text, item string) {
	clipboard.WriteAll(text)
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied (clears in 30s)[-]", item))
	go func() {
		time.Sleep(30 * time.Second)
		curr, _ := clipboard.ReadAll()
		if curr == text {
			clipboard.WriteAll("")
			app.QueueUpdateDraw(func() { viewStatus.SetText("[yellow]Clipboard cleared[-]") })
		}
	}()
}

// --- Path & Config Helpers ---

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func getAttachmentDir() string {
	return filepath.Join(expandPath(dataDir), "_attachments")
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".passbook", "config.json")
}

func loadConfig() {
	data, err := os.ReadFile(getConfigPath())
	if err == nil {
		var cfg AppConfig
		if json.Unmarshal(data, &cfg) == nil && cfg.DataDir != "" {
			dataDir = cfg.DataDir
		}
	}
	saveConfig()
}

func saveConfig() {
	path := getConfigPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	cfg := AppConfig{DataDir: dataDir}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(path, data, 0600)
}

// --- Crypto ---

func deriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

func encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
