package main

import (
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
	lastActivity = time.Now()

	setupUI()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		lastActivity = time.Now()
		return event
	})
	app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		lastActivity = time.Now()
		return event, action
	})

	go func() {
		for range time.Tick(1 * time.Second) {
			app.QueueUpdateDraw(func() {
				if len(masterKey) > 0 && time.Since(lastActivity) > 5*time.Minute {
					lockApp()
				} else {
					drawTOTP()
				}
			})
		}
	}()

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
