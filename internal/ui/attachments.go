package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func downloadAttachment(att Attachment) {
	data, err := uiStore.ReadAttachment(att.ID)
	if err != nil {
		uiViewStatus.SetText("[red]Failed to read attachment[-]")
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
	err = os.WriteFile(dest, data, 0644)
	if err != nil {
		uiViewStatus.SetText("[red]Failed to save to Downloads[-]")
		return
	}
	uiViewStatus.SetText(fmt.Sprintf("[green]✓ Saved to Downloads: %s[-]", att.FileName))

	uiAttachmentList.SetCurrentItem(-1)
}
