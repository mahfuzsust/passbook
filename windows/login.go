package windows

import (
	"passbook/crypto"
	"passbook/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var settings models.Settings

func CrateLoginWindow(app fyne.App, s models.Settings, showRun bool) {
	settings = s
	w := app.NewWindow("Login")
	w.Resize(fyne.NewSize(400, 300))

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter Password")
	loginButton := widget.NewButton("Login", func() {
		handleLogin(passwordEntry.Text, w, app)
	})

	passwordEntry.OnSubmitted = func(s string) {
		handleLogin(s, w, app)
	}

	w.SetContent(container.NewVBox(
		widget.NewLabel("Login"),
		passwordEntry,
		loginButton,
	))
	if showRun {
		w.ShowAndRun()
	} else {
		w.Show()
	}
}

func handleLogin(passwordInput string, w fyne.Window, app fyne.App) {
	if crypto.VerifyPassword(passwordInput, settings.PasswordHash) {
		w.Close()
		ShowMainWindow(app)
	} else {
		dialog.ShowError(nil, w)
	}
}
