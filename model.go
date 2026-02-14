package main

import (
	"github.com/gdamore/tcell/v2"
)

const (
	colorUnfocusedBg = tcell.Color236
	colorFocusedBg   = tcell.Color24
)

const (
	TypeLogin EntryType = "Login"
	TypeCard  EntryType = "Card"
	TypeNote  EntryType = "Note"
	TypeFile  EntryType = "File"
)

type EntryType string

type PasswordHistory struct {
	Password string `json:"password"`
	Date     string `json:"date"`
}

type Attachment struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
}

type Entry struct {
	Type        EntryType         `json:"type"`
	Title       string            `json:"title"`
	Username    string            `json:"username,omitempty"`
	Password    string            `json:"password,omitempty"`
	Link        string            `json:"link,omitempty"`
	TOTPSecret  string            `json:"totp_secret,omitempty"`
	CardNumber  string            `json:"card_number,omitempty"`
	Expiry      string            `json:"expiry,omitempty"`
	CVV         string            `json:"cvv,omitempty"`
	CustomText  string            `json:"custom_text,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	History     []PasswordHistory `json:"history,omitempty"`
	FileName    string            `json:"file_name,omitempty"`
	FileData    []byte            `json:"file_data,omitempty"`
}
