package ui

import (
	"passbook/internal/config"
	"passbook/internal/crypto"
	"passbook/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiLoginForm     *tview.Form
	uiLoginModal    tview.Primitive
	uiLoginStrength *strengthMeter
)

func goToMain(pwd string) {
	if pwd == "" {
		return
	}

	dataDir := config.ExpandPath(uiDataDir)

	vaultParams, err := crypto.LoadVaultParams(dataDir)
	if err != nil {
		return
	}

	existingVault := vaultParams != nil && crypto.SecretExists(dataDir)

	if !existingVault {
		_, level, _ := utils.PasswordStrength(pwd)
		if level < utils.StrengthGood {
			showLoginError("Password is too weak.")
			return
		}

		newParams, err := crypto.DefaultVaultParams()
		if err != nil {
			return
		}

		masterKey, vaultKey, err := crypto.DeriveKeys(pwd, newParams)
		if err != nil {
			return
		}

		if err := crypto.SaveVaultParams(dataDir, newParams); err != nil {
			crypto.WipeBytes(masterKey)
			crypto.WipeBytes(vaultKey)
			return
		}

		if _, err := crypto.EnsureSecret(dataDir, masterKey, newParams); err != nil {
			crypto.WipeBytes(masterKey)
			crypto.WipeBytes(vaultKey)
			return
		}

		uiMasterKey = vaultKey
		uiTempMasterKey = masterKey
		showPinSetup()
		return
	}

	masterKey, vaultKey, err := crypto.DeriveKeys(pwd, vaultParams)
	if err != nil {
		return
	}

	if err := crypto.VerifyMasterKey(dataDir, masterKey); err != nil {
		crypto.WipeBytes(masterKey)
		crypto.WipeBytes(vaultKey)
		showLoginError("Wrong password.")
		return
	}

	if err := crypto.VerifyCommitTag(dataDir, masterKey); err != nil {
		crypto.WipeBytes(masterKey)
		crypto.WipeBytes(vaultKey)
		showLoginError("Wrong password.")
		return
	}

	if crypto.NeedsRehash(vaultParams) {
		if newParams, err := crypto.RehashVault(dataDir, pwd, vaultParams); err == nil && newParams != nil {
			vaultParams = newParams
			crypto.WipeBytes(masterKey)
			crypto.WipeBytes(vaultKey)
			masterKey, vaultKey, err = crypto.DeriveKeys(pwd, newParams)
			if err != nil {
				return
			}
		}
	}

	uiMasterKey = vaultKey
	uiTempMasterKey = masterKey

	pinCfg, _ := crypto.ReadPinConfig(dataDir, masterKey)
	if pinCfg != nil && pinCfg.Mode != "" {
		showPinVerify(pinCfg)
	} else {
		showPinSetup()
	}
}

func setupLogin() {
	uiLoginStrength = newStrengthMeter()

	uiLoginForm = tview.NewForm()
	uiLoginForm.AddPasswordField("Master Password", "", 0, '*', func(text string) {
		uiLoginStrength.Update(text)
	})
	uiLoginStrength.AddTo(uiLoginForm)
	uiLoginForm.AddButton("Login", func() {
		uiLoginHasError = false
		pwd := uiLoginForm.GetFormItem(0).(*tview.InputField).GetText()
		goToMain(pwd)
	})

	uiLoginForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			uiLoginHasError = false
			pwd := uiLoginForm.GetFormItem(0).(*tview.InputField).GetText()
			goToMain(pwd)
			if uiLoginHasError {
				return nil
			}
		}
		return event
	})
	uiLoginForm.AddButton("Quit", func() { uiApp.Stop() })
	uiLoginForm.SetBorder(true).SetTitle(" PassBook Login ").SetTitleAlign(tview.AlignCenter)
	styleForm(uiLoginForm)

	uiLoginModal = newResponsiveModal(uiLoginForm, 55, 10, 80, 15, 0.5, 0.4)
	uiPages.AddPage("login", uiLoginModal, true, true)
}

var uiLoginHasError bool

func showLoginError(msg string) {
	if uiLoginStrength != nil {
		for _, tv := range uiLoginStrength.views {
			tv.SetText("[red]" + msg)
		}
	}
	uiLoginHasError = true
	uiApp.SetFocus(uiLoginForm.GetFormItem(0).(*tview.InputField))
}
