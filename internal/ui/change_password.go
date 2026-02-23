package ui

import (
	"passbook/internal/config"
	"passbook/internal/crypto"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiChangePwdForm     *tview.Form
	uiChangePwdModal    tview.Primitive
	uiChangePwdStrength *strengthMeter
)

func setupChangePassword() {
	uiChangePwdStrength = newStrengthMeter()

	uiChangePwdForm = tview.NewForm()
	uiChangePwdForm.AddPasswordField("Current Password", "", 0, '*', nil)
	uiChangePwdForm.AddPasswordField("New Password", "", 0, '*', func(text string) {
		uiChangePwdStrength.Update(text)
	})
	uiChangePwdStrength.AddTo(uiChangePwdForm)
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

	uiChangePwdModal = newResponsiveModal(uiChangePwdForm, 50, 14, 80, 18, 0.5, 0.4)
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

	dataDir := config.ExpandPath(uiDataDir)

	// Verify the current password and derive the old vault key.
	var oldVaultKey []byte
	if uiCfg.IsMigrated {
		kdfParams, err := crypto.LoadRootKDFParams(dataDir)
		if err != nil || kdfParams == nil {
			showChangePwdError("Failed to load KDF params.")
			return
		}
		oldMasterKey, vk, err := crypto.DeriveKeys(currentPwd, *kdfParams)
		if err != nil {
			showChangePwdError("Key derivation error: " + err.Error())
			return
		}
		if _, err := crypto.EnsureKDFSecret(dataDir, oldMasterKey); err != nil {
			showChangePwdError("Current password is incorrect.")
			return
		}
		oldVaultKey = vk

		// --- BEGIN supportLegacy ---
	} else if crypto.SupportLegacy() {
		// Legacy scheme.
		oldMasterKey := crypto.DeriveLegacyMasterKey(currentPwd)
		oldKDF, err := crypto.EnsureKDFSecret(dataDir, oldMasterKey)
		if err != nil {
			showChangePwdError("Current password is incorrect.")
			return
		}
		oldVaultKey = crypto.DeriveKey(currentPwd, oldKDF)
		// --- END supportLegacy ---

	} else {
		showChangePwdError("Current password is incorrect.")
		return
	}

	// Always use new HKDF scheme for the new password.
	newParams, err := crypto.DefaultRootKDFParams()
	if err != nil {
		showChangePwdError("Failed to generate params: " + err.Error())
		return
	}

	newMasterKey, newVaultKey, err := crypto.DeriveKeys(newPwd, newParams)
	if err != nil {
		showChangePwdError("Key derivation error: " + err.Error())
		return
	}

	// Re-encrypt .secret with the new master key.
	if err := crypto.ReKeyVault(dataDir, newMasterKey); err != nil {
		showChangePwdError("Failed to write new secret: " + err.Error())
		return
	}

	// Re-encrypt all entries and attachments.
	if err := crypto.ReKeyEntries(dataDir, oldVaultKey, newVaultKey); err != nil {
		showChangePwdError("Re-encrypt failed: " + err.Error())
		return
	}

	// Persist the new KDF params and migration state.
	if err := crypto.SaveRootKDFParams(dataDir, newParams); err != nil {
		showChangePwdError("Failed to save KDF params: " + err.Error())
		return
	}
	uiCfg.IsMigrated = true
	if err := config.Save(uiCfg); err != nil {
		showChangePwdError("Failed to save config: " + err.Error())
		return
	}

	// Update in-memory state.
	uiMasterKey = newVaultKey

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
	if uiChangePwdStrength != nil {
		uiChangePwdStrength.Update("")
	}
}
