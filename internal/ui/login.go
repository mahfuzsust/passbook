package ui

import (
	"os"

	"passbook/internal/store"
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

	dbExisted := store.DBExists(uiDBPath)

	s, err := store.Open(uiDBPath, pwd)
	if err != nil {
		showLoginError("Wrong password.")
		return
	}
	uiStore = s

	isNewVault := !uiStore.HasEntries() && !uiStore.PinConfigExists()

	if isNewVault {
		_, level, _ := utils.PasswordStrength(pwd)
		if level < utils.StrengthGood {
			closeAndCleanupStore(!dbExisted)
			showLoginError("Password is too weak.")
			return
		}
		showPinSetup()
		return
	}

	pinCfg, _ := uiStore.ReadPinConfig()
	if pinCfg != nil && pinCfg.Mode != "" {
		showPinVerify(pinCfg)
	} else {
		showPinSetup()
	}
}

func closeAndCleanupStore(removeDB bool) {
	if uiStore != nil {
		uiStore.Close()
		uiStore = nil
	}
	if removeDB {
		os.Remove(uiDBPath)
		os.Remove(uiDBPath + "-wal")
		os.Remove(uiDBPath + "-shm")
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
	enableButtonNav(uiLoginForm)

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
