package ui

import (
	"passbook/internal/store"
	"passbook/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiLoginForm     *tview.Form
	uiLoginModal    tview.Primitive
	uiLoginStrength *strengthMeter
	uiFreshInstall  bool
)

func goToMain(pwd string) {
	if pwd == "" {
		return
	}

	dbExisted := store.DBExists(uiDBPath)
	freshInstall := !dbExisted

	s, err := store.Open(uiDBPath, pwd)
	if err != nil {
		if freshInstall {
			store.RemoveDBFiles(uiDBPath)
		}
		showLoginError(loginErrorMessage(err, freshInstall))
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
		store.RemoveDBFiles(uiDBPath)
	}
}

func loginErrorMessage(err error, freshInstall bool) string {
	if store.IsCGODisabled(err) {
		return "Database unavailable. Reinstall a CGO-enabled build."
	}
	if freshInstall {
		return "Could not create vault."
	}
	return "Wrong password."
}

func submitLogin() {
	uiLoginHasError = false
	pwd := uiLoginForm.GetFormItem(0).(*tview.InputField).GetText()
	goToMain(pwd)
}

func loginFormEnterPressed() bool {
	focused := uiApp.GetFocus()
	for i := 0; i < uiLoginForm.GetButtonCount(); i++ {
		if focused == uiLoginForm.GetButton(i) {
			return false
		}
	}
	submitLogin()
	return uiLoginHasError
}

func setupLogin() {
	uiFreshInstall = !store.DBExists(uiDBPath)

	uiLoginStrength = newStrengthMeter()

	uiLoginForm = tview.NewForm()
	uiLoginForm.AddPasswordField("Master Password", "", 0, '*', func(text string) {
		uiLoginStrength.Update(text)
	})
	uiLoginStrength.AddTo(uiLoginForm)

	submitLabel := "Login"
	title := " PassBook Login "
	if uiFreshInstall {
		submitLabel = "Create Vault"
		title = " PassBook Setup "
	}

	uiLoginForm.AddButton(submitLabel, submitLogin)
	uiLoginForm.AddButton("Quit", func() { uiApp.Stop() })

	uiLoginForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			uiApp.Stop()
			return nil
		case tcell.KeyEnter:
			if loginFormEnterPressed() {
				return nil
			}
		}
		return event
	})
	uiLoginForm.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignCenter)
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
