package main

import (
	"github.com/gdamore/tcell/v2"
	"google.golang.org/protobuf/proto"
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

// Helper functions to convert between Entry and the internal format
func (e *Entry) GetTypeEnum() EntryType {
	return EntryType(e.Type)
}

func (e *Entry) SetTypeEnum(t EntryType) {
	e.Type = string(t)
}

func (e *Entry) GetTOTPSecret() string {
	return e.TotpSecret
}

func (e *Entry) SetTOTPSecret(s string) {
	e.TotpSecret = s
}

func (e *Entry) GetCVV() string {
	return e.Cvv
}

func (e *Entry) SetCVV(s string) {
	e.Cvv = s
}

// Helper to create a new Entry
func NewEntry(t EntryType) *Entry {
	return &Entry{
		Type:        string(t),
		Attachments: make([]*Attachment, 0),
		History:     make([]*PasswordHistory, 0),
	}
}

// Helper functions for serialization
func marshalEntry(e *Entry) ([]byte, error) {
	return proto.Marshal(e)
}

func unmarshalEntry(data []byte) (*Entry, error) {
	e := &Entry{}
	err := proto.Unmarshal(data, e)
	return e, err
}
