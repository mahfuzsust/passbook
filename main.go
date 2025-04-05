package main

import (
	"os"
	"passbook/utils"
	"passbook/windows"
	"path/filepath"

	"fyne.io/fyne/v2/app"
)

var storeDir = filepath.Join(os.Getenv("HOME"), ".my_store")

func main() {
	if utils.MissingDirectory(storeDir) {
		return
	}

	app := app.New()
	windows.CrateLoginWindow(app)
}
