package ui

import "passbook/internal/store"

type Entry = store.EntryFull
type Attachment = store.AttachmentMeta
type PasswordHistory = store.PasswordHistory
type EntryType string

const (
	TypeLogin EntryType = "Login"
	TypeCard  EntryType = "Card"
	TypeNote  EntryType = "Note"
	TypeFile  EntryType = "File"
)
