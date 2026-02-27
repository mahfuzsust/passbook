package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiCreateList *tview.List
)

// setupCreateMenu configures the entry type selection modal.
func setupCreateMenu() {
	uiCreateList = tview.NewList().ShowSecondaryText(false)
	uiCreateList.AddItem("Login", "Password & 2FA", 'l', func() { newEntry(TypeLogin) })
	uiCreateList.AddItem("Card", "Credit/Debit Details", 'c', func() { newEntry(TypeCard) })
	uiCreateList.AddItem("Note", "Secure Text", 'n', func() { newEntry(TypeNote) })
	uiCreateList.AddItem("File", "Encrypted Attachments", 'f', func() { newEntry(TypeFile) })
	uiCreateList.SetBorder(true).SetTitle(" Create New ")
	uiCreateList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
			return nil
		}
		return event
	})
	uiPages.AddPage("create_menu", newResponsiveModal(uiCreateList, 30, 14, 50, 20, 0.4, 0.5), true, false)
}

// showCreateMenu shows the entry type selection modal.
func showCreateMenu() {
	uiPages.SwitchToPage("create_menu")
}
