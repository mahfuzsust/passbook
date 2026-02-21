package ui

import (
	"passbook/internal/crypto"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiLoginForm  *tview.Form
	uiLoginModal tview.Primitive
)

func goToMain(pwd string) {
	if pwd == "" {
		return
	}

	mkey := crypto.DeriveMasterKey(pwd)
	secret, err := crypto.EnsureKDFSecret(uiDataDir, mkey)

	if err == nil {
		uiKDF = secret
	}

	uiMasterKey = crypto.DeriveKey(pwd, uiKDF)
	refreshTree("")

	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func setupLogin() {
	uiLoginForm = tview.NewForm()
	uiLoginForm.AddPasswordField("Master Password", "", 0, '*', nil)
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

	uiLoginModal = newResponsiveModal(uiLoginForm, 40, 9, 80, 15, 0.5, 0.4)
	uiPages.AddPage("login", uiLoginModal, true, true)
}
