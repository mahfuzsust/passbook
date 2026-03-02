package ui

import (
	"passbook/internal/store"
	"passbook/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiChangePwdForm     *tview.Form
	uiChangePwdFlex     *tview.Flex
	uiChangePwdModal    tview.Primitive
	uiChangePwdStrength *strengthMeter
	uiChangePwdStatus   *tview.TextView
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

	styleForm(uiChangePwdForm)
	enableButtonNav(uiChangePwdForm)

	uiChangePwdStatus = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)

	uiChangePwdFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(uiChangePwdForm, 0, 1, true).
		AddItem(uiChangePwdStatus, 1, 0, false)
	uiChangePwdFlex.SetBorder(true).SetTitle(" Change Master Password ").SetTitleAlign(tview.AlignCenter)

	uiChangePwdFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			clearChangePwdForm()
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
			return nil
		}
		return event
	})

	uiChangePwdModal = newResponsiveModal(uiChangePwdFlex, 50, 15, 80, 19, 0.5, 0.4)
	uiPages.AddPage("changepwd", uiChangePwdModal, true, false)
}

func showChangePassword() {
	clearChangePwdForm()
	uiPages.SwitchToPage("changepwd")
	uiApp.SetFocus(uiChangePwdForm.GetFormItem(0))
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

	if err := store.VerifyKey(uiDBPath, currentPwd); err != nil {
		showChangePwdError("Current password is incorrect.")
		return
	}

	if currentPwd == newPwd {
		showChangePwdError("New password must be different from current.")
		return
	}

	_, level, _ := utils.PasswordStrength(newPwd)
	if level < utils.StrengthGood {
		showChangePwdError("New password is too weak.")
		return
	}

	if err := uiStore.Rekey(newPwd); err != nil {
		showChangePwdFatal("Failed to change encryption key: " + err.Error())
		return
	}

	clearChangePwdForm()
	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func showChangePwdError(msg string) {
	if uiChangePwdStatus != nil {
		uiChangePwdStatus.SetText("[red]" + msg)
	}
}

func showChangePwdFatal(msg string) {
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
	if uiChangePwdStatus != nil {
		uiChangePwdStatus.SetText("")
	}
}
