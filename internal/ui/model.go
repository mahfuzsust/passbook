package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

const (
	colorUnfocusedBg = tcell.Color236
	colorFocusedBg   = tcell.Color24
)

func NewEntry(t EntryType) *Entry {
	return &Entry{
		Type:        string(t),
		Attachments: make([]*Attachment, 0),
		History:     make([]*PasswordHistory, 0),
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
