package ui

import (
	"passbook/internal/config"
	"passbook/internal/crypto"

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

	if uiCfg.IsMigrated && len(uiCfg.RootSalt) > 0 {
		// Already migrated — use new HKDF scheme.
		masterKey, vaultKey, err := crypto.DeriveKeys(pwd, uiCfg.RootSalt)
		if err != nil {
			return
		}
		if _, err := crypto.EnsureKDFSecret(uiDataDir, masterKey); err != nil {
			return
		}
		uiMasterKey = vaultKey

		// --- BEGIN supportLegacy ---
	} else if crypto.SupportLegacy() {
		// Legacy scheme — verify password first.
		legacyKey := crypto.DeriveLegacyMasterKey(pwd)
		secret, err := crypto.EnsureKDFSecret(uiDataDir, legacyKey)
		if err != nil {
			return
		}
		uiKDF = secret

		// Check if vault has data (non-empty .secret means existing vault).
		dataDir := config.ExpandPath(uiDataDir)
		hasEntries := crypto.VaultHasEntries(dataDir)

		if hasEntries {
			// Existing vault — run migration.
			newSalt, err := crypto.MigrateVault(dataDir, pwd)
			if err != nil {
				// Migration failed — fall back to legacy for this session.
				uiMasterKey = crypto.DeriveKey(pwd, uiKDF)
				refreshTree("")
				uiPages.SwitchToPage("main")
				uiApp.SetFocus(uiTreeView)
				return
			}

			// Persist migration state.
			uiCfg.IsMigrated = true
			uiCfg.RootSalt = newSalt
			_ = config.Save(uiCfg)

			// Derive keys using new scheme.
			_, vaultKey, err := crypto.DeriveKeys(pwd, newSalt)
			if err != nil {
				return
			}
			uiMasterKey = vaultKey
		} else {
			// Brand new vault — set up with new scheme directly.
			newSalt, err := crypto.GenerateRootSalt()
			if err != nil {
				return
			}

			masterKey, vaultKey, err := crypto.DeriveKeys(pwd, newSalt)
			if err != nil {
				return
			}

			// Re-write the .secret with the new master key.
			if err := crypto.ReKeyVault(dataDir, masterKey); err != nil {
				return
			}

			uiCfg.IsMigrated = true
			uiCfg.RootSalt = newSalt
			_ = config.Save(uiCfg)

			uiMasterKey = vaultKey
		}
		// --- END supportLegacy ---

	} else {
		// New vault (legacy removed) — generate salt and set up HKDF scheme.
		newSalt, err := crypto.GenerateRootSalt()
		if err != nil {
			return
		}

		dataDir := config.ExpandPath(uiDataDir)
		masterKey, vaultKey, err := crypto.DeriveKeys(pwd, newSalt)
		if err != nil {
			return
		}

		if _, err := crypto.EnsureKDFSecret(dataDir, masterKey); err != nil {
			return
		}

		uiCfg.IsMigrated = true
		uiCfg.RootSalt = newSalt
		_ = config.Save(uiCfg)

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

	uiLoginModal = newResponsiveModal(uiLoginForm, 40, 10, 80, 15, 0.5, 0.4)
	uiPages.AddPage("login", uiLoginModal, true, true)
}
