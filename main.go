package main

import (
	"os"
	"time"
)

func main() {
	loadConfig()
	err := os.MkdirAll(expandPath(dataDir), 0700)
	if err != nil {
		return
	}
	err = os.MkdirAll(getAttachmentDir(), 0700)
	if err != nil {
		return
	}
	ensureKDFSecret()

	setupUI()

	go func() {
		for range time.Tick(1 * time.Second) {
			app.QueueUpdateDraw(func() { drawTOTP() })
		}
	}()

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
