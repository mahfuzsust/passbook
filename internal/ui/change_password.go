package ui

import (
	"passbook/internal/config"
	"passbook/internal/crypto"
	"passbook/internal/utils"

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
	confirmPwd := uiChangePwdForm.GetFormItem(3).(*tview.InputField).GetText()

	if currentPwd == "" || newPwd == "" || confirmPwd == "" {
		showChangePwdError("All fields are required.")
		return
	}

	if newPwd != confirmPwd {
		showChangePwdError("New passwords do not match.")
		return
	}

	dataDir := config.ExpandPath(uiDataDir)

	// Verify the current password first.
	vaultParams, err := crypto.LoadVaultParams(dataDir)
	if err != nil || vaultParams == nil {
		showChangePwdError("Failed to load vault params.")
		return
	}
	oldMasterKey, vk, err := crypto.DeriveKeys(currentPwd, vaultParams)
	if err != nil {
		showChangePwdError("Key derivation error: " + err.Error())
		return
	}
	if err := crypto.VerifyMasterKey(dataDir, oldMasterKey); err != nil {
		crypto.WipeBytes(oldMasterKey)
		crypto.WipeBytes(vk)
		showChangePwdError("Current password is incorrect.")
		return
	}

	if currentPwd == newPwd {
		crypto.WipeBytes(oldMasterKey)
		crypto.WipeBytes(vk)
		showChangePwdError("New password must be different from current.")
		return
	}

	_, level, _ := utils.PasswordStrength(newPwd)
	if level < utils.StrengthGood {
		crypto.WipeBytes(oldMasterKey)
		crypto.WipeBytes(vk)
		showChangePwdError("New password is too weak.")
		return
	}

	pinCfg, _ := crypto.ReadPinConfig(dataDir, oldMasterKey)
	crypto.WipeBytes(oldMasterKey)
	oldVaultKey := vk

	// Always use new HKDF scheme for the new password.
	newParams, err := crypto.DefaultVaultParams()
	if err != nil {
		showChangePwdError("Failed to generate params: " + err.Error())
		return
	}

	newMasterKey, newVaultKey, err := crypto.DeriveKeys(newPwd, newParams)
	if err != nil {
		showChangePwdError("Key derivation error: " + err.Error())
		return
	}

	// Save new vault params first so the AAD used for .secret encryption
	// matches what future logins will compute from the file on disk.
	if err := crypto.SaveVaultParams(dataDir, newParams); err != nil {
		crypto.WipeBytes(newMasterKey)
		crypto.WipeBytes(newVaultKey)
		crypto.WipeBytes(oldVaultKey)
		showChangePwdError("Failed to save vault params: " + err.Error())
		return
	}

	if err := crypto.WriteSecretWithParams(dataDir, newParams, newMasterKey); err != nil {
		crypto.WipeBytes(newMasterKey)
		crypto.WipeBytes(newVaultKey)
		crypto.WipeBytes(oldVaultKey)
		showChangePwdError("Failed to write new secret: " + err.Error())
		return
	}

	if pinCfg != nil && pinCfg.Mode != "" {
		if err := crypto.WritePinConfig(dataDir, newMasterKey, *pinCfg); err != nil {
			crypto.WipeBytes(newMasterKey)
			crypto.WipeBytes(newVaultKey)
			crypto.WipeBytes(oldVaultKey)
			showChangePwdError("Failed to preserve PIN config: " + err.Error())
			return
		}
	}
	crypto.WipeBytes(newMasterKey)

	if err := crypto.ReKeyEntries(dataDir, oldVaultKey, newVaultKey); err != nil {
		crypto.WipeBytes(newVaultKey)
		crypto.WipeBytes(oldVaultKey)
		showChangePwdError("Re-encrypt failed: " + err.Error())
		return
	}
	crypto.WipeBytes(oldVaultKey)

	// Update in-memory state.
	crypto.WipeBytes(uiMasterKey)
	uiMasterKey = newVaultKey

	clearChangePwdForm()
	refreshTree("")
	uiCurrentPath = ""
	uiRightPages.SwitchToPage("empty")
	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func showChangePwdError(msg string) {
	clearChangePwdForm()
	uiErrorModal.SetText(msg)
	uiErrorModal.SetDoneFunc(func(int, string) {
		uiPages.SwitchToPage("main")
		uiApp.SetFocus(uiTreeView)
		uiErrorModal.SetDoneFunc(func(int, string) { uiPages.SwitchToPage("editor") })
	})
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
