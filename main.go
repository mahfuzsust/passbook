package main

import (
	"passbook/utils"
	"passbook/windows"

	"fyne.io/fyne/v2/app"
)

func main() {
	app := app.New()
	settings, err := utils.LoadSettings()

	if err != nil {
		windows.ShowSettingsWindow(app, true)
	} else {
		windows.CrateLoginWindow(app, settings, true)
	}
}
