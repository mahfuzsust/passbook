package windows

import (
	"passbook/crypto"
	"passbook/models"
	"passbook/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowSettingsWindow(app fyne.App, showLogin bool) {
	w := app.NewWindow("Settings")
	w.Resize(fyne.NewSize(400, 300))

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter Password")

	directoryEntry := widget.NewEntry()
	directoryEntry.SetPlaceHolder("Enter Directory")
	directoryEntry.SetText(".passbook")

	saveButton := widget.NewButton("Save", func() {
		passwordHash, err := crypto.HashPassword(passwordEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		settings := models.Settings{
			PasswordHash:     passwordHash,
			StorageDirectory: directoryEntry.Text,
			BackupEnabled:    false,
			BackupInterval:   0,
		}
		_, err = utils.SaveSettings(settings)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Settings Saved", "Settings have been saved successfully.", w)
		if showLogin {
			w.Close()
			CrateLoginWindow(app, settings, false)
		}
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Password"),
		passwordEntry,
		widget.NewLabel("Storage Directory"),
		directoryEntry,
		saveButton,
	))
	w.ShowAndRun()
}
