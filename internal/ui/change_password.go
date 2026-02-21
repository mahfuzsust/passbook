package ui

import (
	"passbook/internal/config"
	"passbook/internal/crypto"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiChangePwdForm  *tview.Form
	uiChangePwdModal tview.Primitive
)

func setupChangePassword() {
	uiChangePwdForm = tview.NewForm()
	uiChangePwdForm.AddPasswordField("Current Password", "", 0, '*', nil)
	uiChangePwdForm.AddPasswordField("New Password", "", 0, '*', nil)
	uiChangePwdForm.AddPasswordField("Confirm Password", "", 0, '*', nil)

	uiChangePwdForm.AddButton("Change", doChangePassword)
	uiChangePwdForm.AddButton("Cancel", func() {
		clearChangePwdForm()
		uiPages.SwitchToPage("main")
		uiApp.SetFocus(uiTreeView)
	})

	uiChangePwdForm.SetBorder(true).SetTitle(" Change Master Password ").SetTitleAlign(tview.AlignCenter)
	styleForm(uiChangePwdForm)

	uiChangePwdForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			clearChangePwdForm()
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
			return nil
		}
		return event
	})

	uiChangePwdModal = newResponsiveModal(uiChangePwdForm, 50, 13, 80, 18, 0.5, 0.4)
	uiPages.AddPage("changepwd", uiChangePwdModal, true, false)
}

func showChangePassword() {
	clearChangePwdForm()
	uiPages.SwitchToPage("changepwd")
	uiApp.SetFocus(uiChangePwdForm)
}

func doChangePassword() {
	currentPwd := uiChangePwdForm.GetFormItem(0).(*tview.InputField).GetText()
	newPwd := uiChangePwdForm.GetFormItem(1).(*tview.InputField).GetText()
	confirmPwd := uiChangePwdForm.GetFormItem(2).(*tview.InputField).GetText()

	if currentPwd == "" || newPwd == "" || confirmPwd == "" {
		showChangePwdError("All fields are required.")
		return
	}

	if newPwd != confirmPwd {
		showChangePwdError("New passwords do not match.")
		return
	}

	if currentPwd == newPwd {
		showChangePwdError("New password must be different from current.")
		return
	}

	// Verify the current password by trying to load the KDF secret.
	oldMasterKey := crypto.DeriveMasterKey(currentPwd)
	oldKDF, err := crypto.EnsureKDFSecret(uiDataDir, oldMasterKey)
	if err != nil {
		showChangePwdError("Current password is incorrect.")
		return
	}
	oldKey := crypto.DeriveKey(currentPwd, oldKDF)

	// Derive the new master key (used to encrypt the .secret file).
	newMasterKey := crypto.DeriveMasterKey(newPwd)

	dataDir := config.ExpandPath(uiDataDir)

	// Write a new .secret with a fresh salt, encrypted with the new master key.
	if err := crypto.ReKeyVault(dataDir, newMasterKey); err != nil {
		showChangePwdError("Failed to write new secret: " + err.Error())
		return
	}

	// Load the freshly written secret (new salt) and derive the new data key.
	newKDF, err := crypto.EnsureKDFSecret(dataDir, newMasterKey)
	if err != nil {
		showChangePwdError("Failed to load new secret: " + err.Error())
		return
	}
	newKey := crypto.DeriveKey(newPwd, newKDF)

	// Re-encrypt all entries and attachments: old data key → new data key.
	if err := crypto.ReKeyEntries(dataDir, oldKey, newKey); err != nil {
		showChangePwdError("Re-encrypt failed: " + err.Error())
		return
	}

	// Update the in-memory state.
	uiKDF = newKDF
	uiMasterKey = newKey

	clearChangePwdForm()
	refreshTree("")
	uiCurrentPath = ""
	uiRightPages.SwitchToPage("empty")
	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func showChangePwdError(msg string) {
	uiErrorModal.SetText(msg)
	uiPages.SwitchToPage("error")
}

func clearChangePwdForm() {
	if uiChangePwdForm == nil {
		return
	}
	for i := 0; i < uiChangePwdForm.GetFormItemCount(); i++ {
		if input, ok := uiChangePwdForm.GetFormItem(i).(*tview.InputField); ok {
			input.SetText("")
		}
	}
}
