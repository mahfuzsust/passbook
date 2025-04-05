package windows

import (
	"passbook/crypto"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var passwordHash = "$2a$14$zWwEnTtOPXXo4/3KryB.s.2ggEJeeulAm5hVXMq3kZKD7p6RieBfW"

func CrateLoginWindow(app fyne.App) {
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
	w.ShowAndRun()
}

func handleLogin(passwordInput string, w fyne.Window, app fyne.App) {
	if crypto.VerifyPassword(passwordInput, passwordHash) {
		w.Close()
		showMainWindow(app)
	} else {
		dialog.ShowError(nil, w)
	}
}
