package ui

import "github.com/gdamore/tcell/v2"

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
