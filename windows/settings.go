package windows

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func showSettingsWindow(app fyne.App) {
	settingsWindow := app.NewWindow("Settings")
	settingsWindow.Resize(fyne.NewSize(400, 300))

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter new password")

	dirEntry := widget.NewEntry()
	dirEntry.SetPlaceHolder("Set storage directory")

	saveButton := widget.NewButton("Save", func() {
		// Save settings logic (write to config file)
		saveSettings(passwordEntry.Text, dirEntry.Text)
		settingsWindow.Close()
	})

	settingsContent := container.NewVBox(
		widget.NewLabel("Create Password"),
		passwordEntry,
		widget.NewLabel("Set Directory"),
		dirEntry,
		saveButton,
	)

	settingsWindow.SetContent(settingsContent)
	settingsWindow.Show()
}

func saveSettings(s1, s2 string) {
	panic("unimplemented")
}
