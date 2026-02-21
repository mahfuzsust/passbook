package ui

import "passbook/internal/pb"

type Entry = pb.Entry
type Attachment = pb.Attachment
type PasswordHistory = pb.PasswordHistory
type EntryType string

const (
	TypeLogin EntryType = "Login"
	TypeCard  EntryType = "Card"
	TypeNote  EntryType = "Note"
	TypeFile  EntryType = "File"
)
