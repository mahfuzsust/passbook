package windows

import (
	"fmt"
	"passbook/crypto"
	"passbook/models"
	"passbook/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowSettingsWindow(app fyne.App, showLogin, showRun bool) {
	w := app.NewWindow("Settings")
	w.Resize(fyne.NewSize(400, 300))

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter Password")

	directoryEntry := widget.NewEntry()
	directoryEntry.SetPlaceHolder("Enter Directory")
	directoryEntry.SetText(utils.GetDefaultDirectory())

	passwordLengthLabel := widget.NewLabel("Password Length: 18")
	passwordLength := widget.NewSlider(8, 128)
	passwordLength.SetValue(18)
	passwordLength.Step = 1
	passwordLength.OnChanged = func(value float64) {
		st := fmt.Sprintf("%.0f", value)
		passwordLengthLabel.SetText("Password Length: " + st)
		passwordLength.SetValue(value)
	}

	useUpper := widget.NewCheck("Use Uppercase", nil)
	useUpper.SetChecked(true)

	useNumbers := widget.NewCheck("Use Numbers", nil)
	useNumbers.SetChecked(true)

	useSpecial := widget.NewCheck("Use Special Characters", nil)
	useSpecial.SetChecked(true)

	if !showRun {
		directoryEntry.SetText(settings.StorageDirectory)
		passwordLength.SetValue(float64(settings.PasswordLength))
		useUpper.SetChecked(settings.UseUpper)
		useNumbers.SetChecked(settings.UseNumbers)
		useSpecial.SetChecked(settings.UseSpecial)
	}

	saveButton := widget.NewButton("Save", func() {
		if showRun && passwordEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("password cannot be empty"), w)
			return
		}

		passwordHash, err := crypto.HashPassword(passwordEntry.Text)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		if !showRun {
			passwordHash = settings.PasswordHash
		}

		settings = models.Settings{
			PasswordHash:     passwordHash,
			StorageDirectory: directoryEntry.Text,
			BackupEnabled:    false,
			BackupInterval:   0,
			PasswordLength:   int(passwordLength.Value),
			UseUpper:         useUpper.Checked,
			UseNumbers:       useNumbers.Checked,
			UseSpecial:       useSpecial.Checked,
		}
		_, err = utils.SaveSettings(settings)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Settings Saved", "Settings have been saved successfully.", w)
		w.Close()
		if showLogin {
			CrateLoginWindow(app, settings, false)
		}
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Password"),
		passwordEntry,
		widget.NewLabel("Storage Directory"),
		directoryEntry,
		passwordLengthLabel, passwordLength,
		useUpper,
		useNumbers,
		useSpecial,

		saveButton,
	))
	if showRun {
		w.ShowAndRun()
	} else {
		w.Show()
	}
}
