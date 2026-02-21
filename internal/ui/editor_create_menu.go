package ui

import "github.com/rivo/tview"

// setupCreateMenu configures the entry type selection modal.
func setupCreateMenu() {
	uiCreateList = tview.NewList().ShowSecondaryText(false)
	uiCreateList.AddItem("Login", "Password & 2FA", 'l', func() { newEntry(TypeLogin) })
	uiCreateList.AddItem("Card", "Credit/Debit Details", 'c', func() { newEntry(TypeCard) })
	uiCreateList.AddItem("Note", "Secure Text", 'n', func() { newEntry(TypeNote) })
	uiCreateList.AddItem("File", "Encrypted Attachments", 'f', func() { newEntry(TypeFile) })
	uiCreateList.SetBorder(true).SetTitle(" Create New ")
	uiPages.AddPage("create_menu", newResponsiveModal(uiCreateList, 30, 14, 50, 20, 0.4, 0.5), true, false)
}

// showCreateMenu shows the entry type selection modal.
func showCreateMenu() {
	uiPages.SwitchToPage("create_menu")
}
