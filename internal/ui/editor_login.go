package ui

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"

	"passbook/internal/platform"
)

var uiEditorLoginStrength *strengthMeter

// addLoginFields adds login-specific form fields to the editor.
func addLoginFields(ent *Entry) {
	ent.Attachments = nil
	uiEditorForm.AddInputField("Username", ent.Username, 40, nil, nil)

	uiEditorLoginStrength = newStrengthMeter()
	uiEditorLoginStrength.Update(ent.Password)

	uiEditorPasswordField = tview.NewInputField().SetLabel("Password").SetText(ent.Password).SetFieldWidth(40)
	uiEditorPasswordField.SetChangedFunc(func(text string) {
		uiEditorLoginStrength.Update(text)
	})
	uiEditorForm.AddFormItem(uiEditorPasswordField)
	uiEditorLoginStrength.AddTo(uiEditorForm)
	uiEditorForm.AddButton("generate", func() {
		updatePassPreview()
		uiPages.SwitchToPage("passgen")
	})

	uiEditorForm.AddInputField("Link", ent.Link, 40, nil, nil)
	uiEditorForm.AddInputField("TOTP Secret", ent.TotpSecret, 40, nil, nil)
}

// collectLoginFields reads login form values into the entry.
func collectLoginFields(ent *Entry, priorPassword string) {
	ent.Username = uiEditorForm.GetFormItemByLabel("Username").(*tview.InputField).GetText()

	if uiEditorPasswordField != nil {
		ent.Password = uiEditorPasswordField.GetText()
	}

	ent.Link = uiEditorForm.GetFormItemByLabel("Link").(*tview.InputField).GetText()
	ent.TotpSecret = uiEditorForm.GetFormItemByLabel("TOTP Secret").(*tview.InputField).GetText()

	if priorPassword != "" && priorPassword != ent.Password {
		ent.History = append(ent.History, &PasswordHistory{
			Password: priorPassword,
			Date:     time.Now().Format("2006-01-02 15:04"),
		})
	}
}

// renderLoginView renders the login-type view pane content.
func renderLoginView() {
	if uiCurrentEnt.Username != "" {
		uiViewSubtitle.SetText(uiCurrentEnt.Username)
		btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
			if err := clipboard.WriteAll(uiCurrentEnt.Username); err != nil {
				return
			}
			notifyCopied("Username")
		}))
		uiViewFlex.AddItem(makeRow("Username:", uiViewSubtitle, btnCopy), 1, 0, false)
	}

	if uiCurrentEnt.Password != "" {
		pass := strings.Repeat("*", len(uiCurrentEnt.Password))
		if uiShowSensitive {
			pass = uiCurrentEnt.Password
		}
		uiViewPassword.SetText(pass)
		btnPass := styleButton(tview.NewButton("cp").SetSelectedFunc(func() { copySensitive(uiCurrentEnt.Password, "Password") }))
		btnShow := styleButton(tview.NewButton("vw").SetSelectedFunc(func() { uiShowSensitive = !uiShowSensitive; updateViewPane() }))
		btnHist := styleButton(tview.NewButton("his").SetSelectedFunc(func() { showHistory() }))
		uiViewFlex.AddItem(makeRow("Password:", uiViewPassword, btnShow, btnPass, btnHist), 1, 0, false)
	} else {
		uiShowSensitive = false
	}

	if strings.TrimSpace(uiCurrentEnt.Link) != "" {
		linkText := tview.NewTextView().SetDynamicColors(true)
		linkText.SetText("[blue::u]" + uiCurrentEnt.Link + "[-:-:-]")
		btnOpen := styleButton(tview.NewButton("open").SetSelectedFunc(func() { _ = platform.OpenURL(uiCurrentEnt.Link) }))
		btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
			if err := clipboard.WriteAll(uiCurrentEnt.Link); err != nil {
				return
			}
			notifyCopied("Link")
		}))
		uiViewFlex.AddItem(makeRow("Link:", linkText, btnOpen, btnCopy), 1, 0, false)
	}

	cleanSecret := strings.ReplaceAll(uiCurrentEnt.TotpSecret, " ", "")
	if cleanSecret != "" {
		uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
		btnTotp := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
			code, err := totp.GenerateCode(cleanSecret, time.Now())
			if err == nil {
				copySensitive(code, "TOTP")
			}
		}))
		uiViewFlex.AddItem(makeRow("TOTP:", uiViewTOTP, btnTotp), 1, 0, false)
		uiViewFlex.AddItem(makeRow("", uiViewTOTPBar), 1, 0, false)
		drawTOTP()
	} else {
		uiViewTOTP.SetText("")
		uiViewTOTPBar.SetText("")
	}
}
