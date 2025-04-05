package windows

import (
	"fmt"
	"passbook/lib"
	"passbook/models"
	"passbook/utils"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var nameEntry, urlEntry, passwordEntry, notesEntry, usernameEntry, totpEntry *widget.Entry
var totpLabel *widget.Label
var totpRow *fyne.Container
var editMode bool = false
var stopTOTP chan struct{}
var runningTOTPUpdater bool
var listItems []string
var progress *lib.CircularProgressBar

func ShowMainWindow(app fyne.App) {
	w := app.NewWindow("Passbook")
	w.Resize(fyne.NewSize(800, 600))

	listItems = utils.UpdateList(settings.StorageDirectory)

	list := widget.NewList(
		func() int { return len(listItems) },
		func() fyne.CanvasObject { return widget.NewLabel("Item") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(listItems[i])
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		loadFile(listItems[id], w)
	}

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search...")
	searchEntry.OnChanged = func(s string) {
		searchList(s, list)
	}

	addButton := widget.NewButton("Add", func() {
		clearFields()
		editMode = false
		totpRow.Hide()
		if runningTOTPUpdater {
			close(stopTOTP)
			runningTOTPUpdater = false
		}
	})

	leftContent := container.NewBorder(searchEntry, addButton, nil, nil, list)

	nameEntry = widget.NewEntry()
	urlEntry = widget.NewEntry()
	passwordEntry = widget.NewPasswordEntry()
	notesEntry = widget.NewMultiLineEntry()
	usernameEntry = widget.NewEntry()

	copyUsernameBtn := newDynamicCopyButton(usernameEntry)
	copyPasswordBtn := newDynamicCopyButton(passwordEntry)

	userNameRow := container.NewBorder(
		nil, nil, nil, copyUsernameBtn,
		container.NewStack(usernameEntry),
	)
	passwordRow := container.NewBorder(
		nil, nil, nil, copyPasswordBtn,
		container.NewStack(passwordEntry),
	)

	saveButton := widget.NewButton("Save", func() {
		fileName := nameEntry.Text
		saveFile(fileName, w, list)
	})

	totpEntry = widget.NewEntry()
	totpEntry.SetPlaceHolder("Enter TOTP Secret Key")

	totpLabel = widget.NewLabel("")
	copyTOTPButton := newLabelCopyButton(totpLabel)
	progress = lib.NewCircularProgressBar()

	totpRow = container.NewBorder(
		nil, nil, nil, copyTOTPButton, progress,
		container.NewStack(totpLabel),
	)
	totpRow.Hide()

	rightContent := container.NewVBox(
		widget.NewLabel("Name"), nameEntry,
		widget.NewLabel("Username"), userNameRow,
		widget.NewLabel("Password"), passwordRow,
		widget.NewLabel("TOTP Secret Key"), totpEntry,
		widget.NewLabel("Generated TOTP"), totpRow,
		widget.NewLabel("URL"), urlEntry,
		widget.NewLabel("Notes"), notesEntry,
		saveButton,
	)

	split := container.NewHSplit(leftContent, rightContent)
	split.SetOffset(0.3)

	w.SetContent(split)
	w.Show()

	settingsMenuItem := fyne.NewMenuItem("Settings", func() {
		ShowSettingsWindow(app, false)
	})

	menu := fyne.NewMainMenu(fyne.NewMenu("Options", settingsMenuItem))
	w.SetMainMenu(menu)

	list.Refresh()

}

func newDynamicCopyButton(entry *widget.Entry) *widget.Button {
	return widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		text := entry.Text
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(text)
	})
}
func newLabelCopyButton(entry *widget.Label) *widget.Button {
	return widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		text := entry.Text
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(text)
	})
}

func searchList(s string, list *widget.List) {
	if len(s) == 0 {
		listItems = utils.UpdateList(settings.StorageDirectory)
	} else {
		listItems = utils.GetFilteredList(s, listItems)
	}
	list.Refresh()
}

func clearFields() {
	nameEntry.SetText("")
	urlEntry.SetText("")
	passwordEntry.SetText("")
	notesEntry.SetText("")
	usernameEntry.SetText("")
	totpEntry.SetText("")
}

func loadFile(fileName string, w fyne.Window) {
	editMode = true
	filePath := filepath.Join(settings.StorageDirectory, fileName)

	details, err := utils.LoadFileContent(filePath, settings.PasswordHash)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to load file: %v", err), w)
		return
	}

	nameEntry.SetText(details.Name)
	urlEntry.SetText(details.URL)
	passwordEntry.SetText(details.Password)
	notesEntry.SetText(details.Notes)
	usernameEntry.SetText(details.Username)
	totpEntry.SetText(details.TotpSecret)

	if len(details.TotpSecret) > 0 {
		totpRow.Show()
		startTOTPUpdater(totpEntry, totpLabel)
	} else {
		if runningTOTPUpdater {
			close(stopTOTP) // Stop only if running
			runningTOTPUpdater = false
		}
		totpRow.Hide()
	}
}

func saveFile(fileName string, w fyne.Window, list *widget.List) {
	fileDetails := models.FileDetails{
		Name:       nameEntry.Text,
		URL:        urlEntry.Text,
		Password:   passwordEntry.Text,
		Notes:      notesEntry.Text,
		Username:   usernameEntry.Text,
		TotpSecret: totpEntry.Text,
	}

	filePath := filepath.Join(settings.StorageDirectory, fileName)
	_, err := utils.SaveFileContent(filePath, settings.PasswordHash, fileDetails)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w)
		return
	}

	if !editMode {
		listItems = append(listItems, fileName)
		list.Refresh()
	}
}

func startTOTPUpdater(totpEntry *widget.Entry, totpLabel *widget.Label) {
	if runningTOTPUpdater {
		close(stopTOTP)
		runningTOTPUpdater = false
	}
	stopTOTP = make(chan struct{})
	runningTOTPUpdater = true

	go func() {
		for {
			select {
			case <-stopTOTP:
				runningTOTPUpdater = false
				return
			default:
				secret := totpEntry.Text
				if secret == "" {
					totpLabel.SetText("No TOTP Secret")
					return
				}

				timeLeft := 30 - (time.Now().Unix() % 30)
				totpLabel.SetText(utils.GenerateTOTP(secret))
				progress.SetProgress(float64(timeLeft) / 30.0)

				time.Sleep(1 * time.Second)
			}
		}
	}()
}
