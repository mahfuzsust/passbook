package ui

import (
	"testing"

	"github.com/rivo/tview"
)

func initViewTestState() {
	uiViewFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	uiViewTitle = tview.NewTextView()
	uiViewSubtitle = tview.NewTextView()
	uiViewPassword = tview.NewTextView()
	uiViewDetails = tview.NewTextView()
	uiViewTOTP = tview.NewTextView()
	uiViewTOTPBar = tview.NewTextView()
	uiViewCustom = tview.NewTextView()
	uiViewStatus = tview.NewTextView()
	uiAttachmentList = tview.NewList()
	uiShowSensitive = false
}

func TestUpdateViewPaneSetsTitle(t *testing.T) {
	resetEditorTestState()
	initViewTestState()
	uiCurrentEnt = &Entry{Type: string(TypeNote), Title: "My Note"}

	updateViewPane()
	got := uiViewTitle.GetText(false)
	if got != "My Note" {
		t.Fatalf("expected title text to be set, got %q", got)
	}
}

func TestUpdateViewPaneMasksCardNumber(t *testing.T) {
	resetEditorTestState()
	initViewTestState()
	uiCurrentEnt = &Entry{Type: string(TypeCard), Title: "Card", CardNumber: "1234567812345678", Expiry: "12/34", Cvv: "123"}

	updateViewPane()
	got := uiViewSubtitle.GetText(false)
	if got != "**** **** **** 5678" {
		t.Fatalf("expected masked card number, got %q", got)
	}
}

func TestUpdateViewPaneShowsNotes(t *testing.T) {
	resetEditorTestState()
	initViewTestState()
	uiCurrentEnt = &Entry{Type: string(TypeNote), Title: "Note", CustomText: "hello"}

	updateViewPane()
	got := uiViewCustom.GetText(false)
	if got != "hello" {
		t.Fatalf("expected notes text to be set, got %q", got)
	}
}

func TestUpdateViewPaneAddsAttachments(t *testing.T) {
	resetEditorTestState()
	initViewTestState()
	uiCurrentEnt = &Entry{
		Type:        string(TypeFile),
		Title:       "Files",
		Attachments: []*Attachment{{Id: "1", FileName: "a.txt", Size: 10}},
	}

	updateViewPane()
	if uiAttachmentList.GetItemCount() == 0 {
		t.Fatalf("expected attachment list to be populated")
	}
}
