package main

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

// --- Constants ---

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

// --- Data Models ---

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
}

type AppConfig struct {
	DataDir string `json:"data_dir"`
}

// --- Global State ---

var (
	masterKey         []byte
	dataDir           = "~/.passbook/data"
	currentPath       string
	currentEnt        Entry
	editingEnt        Entry
	lastActivity      time.Time
	lastGeneratedPass string

	// State for Attachments & Collisions
	pendingAttachments []Attachment
	pendingFilePaths   map[string]string
	pendingSaveData    []byte
	pendingPath        string
)
