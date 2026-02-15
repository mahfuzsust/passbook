package ui

import (
	"fmt"
	"os"
	"passbook/internal/crypto"
	"path/filepath"
	"runtime"
)

func downloadAttachment(att *Attachment) {
	data, err := os.ReadFile(filepath.Join(getAttachmentDir(), att.Id))
	if err != nil {
		uiViewStatus.SetText("[red]Failed to read attachment[-]")
		return
	}
	dec, err := crypto.Decrypt(uiMasterKey, data)
	if err != nil {
		uiViewStatus.SetText("[red]Failed to decrypt attachment (wrong key?)[-]")
		return
	}

	var downDir string
	if runtime.GOOS == "windows" {
		downDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	} else {
		home, _ := os.UserHomeDir()
		downDir = filepath.Join(home, "Downloads")
	}

	dest := filepath.Join(downDir, att.FileName)
	err = os.WriteFile(dest, dec, 0644)
	if err != nil {
		uiViewStatus.SetText("[red]Failed to save to Downloads[-]")
		return
	}
	uiViewStatus.SetText(fmt.Sprintf("[green]✓ Saved to Downloads: %s[-]", att.FileName))

	// Clear selection to prevent persistent highlighting
	uiAttachmentList.SetCurrentItem(-1)
}
