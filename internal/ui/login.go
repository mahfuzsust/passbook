package ui

import (
	"errors"
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

	// Load vault params — nil means first-time vault.
	vaultParams, err := crypto.LoadVaultParams(dataDir)
	if err != nil {
		return
	}

	if vaultParams == nil {
		// First-time vault — enforce minimum password strength.
		_, level, _ := utils.PasswordStrength(pwd)
		if level < utils.StrengthGood {
			showLoginError("Password is too weak.")
			return
		}

		// Create params and set up.
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
		crypto.WipeBytes(masterKey)

		uiMasterKey = vaultKey
	} else {
		// Existing vault — verify password.
		masterKey, vaultKey, err := crypto.DeriveKeys(pwd, *vaultParams)
		if err != nil {
			return
		}
		if _, err := crypto.EnsureSecret(dataDir, masterKey, *vaultParams); err != nil {
			crypto.WipeBytes(masterKey)
			crypto.WipeBytes(vaultKey)
			return
		}
		if err := crypto.VerifyVaultParamsHash(dataDir, masterKey); err != nil {
			crypto.WipeBytes(masterKey)
			crypto.WipeBytes(vaultKey)
			return
		}
		if err := crypto.VerifyCommitTag(dataDir, masterKey); err != nil {
			crypto.WipeBytes(masterKey)
			crypto.WipeBytes(vaultKey)
			if errors.Is(err, crypto.ErrWrongPassword) {
				return
			}
			return
		}
		crypto.WipeBytes(masterKey)

		// Auto-rehash if Argon2id parameters are weaker than recommended.
		if vaultParams.NeedsRehash() {
			if newParams, err := crypto.RehashVault(dataDir, pwd, *vaultParams); err == nil && newParams != nil {
				vaultParams = newParams
				if _, vaultKey, err = crypto.DeriveKeys(pwd, *newParams); err != nil {
					return
				}
			}
		}

		uiMasterKey = vaultKey
	}

	refreshTree("")

	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func setupLogin() {
	uiLoginStrength = newStrengthMeter()

	uiLoginForm = tview.NewForm()
	uiLoginForm.AddPasswordField("Master Password", "", 0, '*', func(text string) {
		uiLoginStrength.Update(text)
	})
	uiLoginStrength.AddTo(uiLoginForm)
	uiLoginForm.AddButton("Login", func() {
		pwd := uiLoginForm.GetFormItem(0).(*tview.InputField).GetText()
		goToMain(pwd)
	})

	uiLoginForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			pwd := uiLoginForm.GetFormItem(0).(*tview.InputField).GetText()
			goToMain(pwd)
		}
		return event
	})
	uiLoginForm.AddButton("Quit", func() { uiApp.Stop() })
	uiLoginForm.SetBorder(true).SetTitle(" PassBook Login ").SetTitleAlign(tview.AlignCenter)
	styleForm(uiLoginForm)

	uiLoginModal = newResponsiveModal(uiLoginForm, 55, 10, 80, 15, 0.5, 0.4)
	uiPages.AddPage("login", uiLoginModal, true, true)
}

func showLoginError(msg string) {
	if uiLoginStrength != nil {
		for _, tv := range uiLoginStrength.views {
			tv.SetText("[red]" + msg)
		}
	}
}

func clearLoginError() {
	// No-op: strength meter auto-updates on next keystroke.
}
