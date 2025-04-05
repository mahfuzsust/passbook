package windows

import (
	"encoding/json"
	"fmt"
	"os"
	"passbook/crypto"
	"passbook/utils"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var nameEntry, urlEntry, passwordEntry, notesEntry, usernameEntry, totpEntry *widget.Entry
var totpLabel *widget.Label
var progressBar *widget.ProgressBar
var editMode bool = false
var stopTOTP chan struct{}
var runningTOTPUpdater bool
var listItems []string
var storeDir = filepath.Join(os.Getenv("HOME"), ".my_store")

type FileDetails struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Password   string `json:"password"`
	Notes      string `json:"notes"`
	Username   string `json:"username"`
	TotpSecret string `json:"totp_secret"`
}

func showMainWindow(app fyne.App) {
	w := app.NewWindow("Main App")
	w.Resize(fyne.NewSize(800, 600))

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search...")

	updateList()

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

	searchEntry.OnChanged = func(s string) {
		searchList(s, list)
	}

	addButton := widget.NewButton("Add", func() {
		clearFields()
		editMode = false
	})

	leftContent := container.NewBorder(searchEntry, addButton, nil, nil, list)

	nameEntry = widget.NewEntry()
	urlEntry = widget.NewEntry()
	passwordEntry = widget.NewPasswordEntry()
	notesEntry = widget.NewMultiLineEntry()
	usernameEntry = widget.NewEntry()

	copyUsernameBtn := widget.NewButton("Copy", func() {
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(usernameEntry.Text)
	})
	copyPasswordBtn := widget.NewButton("Copy", func() {
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(passwordEntry.Text)
	})

	saveButton := widget.NewButton("Save", func() {
		fileName := nameEntry.Text
		saveFile(fileName, w, list)
	})

	totpEntry = widget.NewEntry()
	totpEntry.SetPlaceHolder("Enter TOTP Secret Key")

	totpLabel = widget.NewLabel("")
	progressBar = widget.NewProgressBar()
	copyTOTPButton := widget.NewButton("Copy", func() {
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(totpLabel.Text)
	})

	totpLabel.Show()
	progressBar.Show()
	copyTOTPButton.Show()

	rightContent := container.NewVBox(
		widget.NewLabel("Name"), nameEntry,
		widget.NewLabel("Username"), usernameEntry, copyUsernameBtn,
		widget.NewLabel("Password"), passwordEntry, copyPasswordBtn,
		widget.NewLabel("TOTP Secret"), totpEntry,
		widget.NewLabel("TOTP Secret Key"), totpEntry,
		widget.NewLabel("Generated TOTP"), container.NewHBox(totpLabel, copyTOTPButton),
		progressBar,
		copyTOTPButton,
		widget.NewLabel("URL"), urlEntry,
		widget.NewLabel("Notes"), notesEntry,
		saveButton,
	)

	split := container.NewHSplit(leftContent, rightContent)
	split.SetOffset(0.3)

	w.SetContent(split)
	w.Show()

	settingsMenuItem := fyne.NewMenuItem("Settings", func() {
		showSettingsWindow(app)
	})

	menu := fyne.NewMainMenu(fyne.NewMenu("Options", settingsMenuItem))
	w.SetMainMenu(menu)

	list.Refresh()

}

func searchList(s string, list *widget.List) {
	if len(s) > 2 {
		var filtered []string
		for _, item := range listItems {
			if strings.Contains(strings.ToLower(item), s) {
				filtered = append(filtered, item)
			}
		}
		listItems = filtered
	}
	if len(s) == 0 {
		updateList()
	}
	list.Refresh()
}

func updateList() {
	files, err := os.ReadDir(storeDir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	listItems = []string{}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue

		}
		listItems = append(listItems, file.Name())
	}
}

func clearFields() {
	nameEntry.SetText("")
	urlEntry.SetText("")
	passwordEntry.SetText("")
	notesEntry.SetText("")
	usernameEntry.SetText("")
}

func loadFile(fileName string, w fyne.Window) {
	editMode = true
	filePath := filepath.Join(storeDir, fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to load file: %v", err), w)
		return
	}
	decryptedData, err := crypto.Decrypt(data, passwordHash)
	if err != nil {
		dialog.ShowError(fmt.Errorf("decryption failed: %v", err), w)
		return
	}
	var details FileDetails
	if err := json.Unmarshal(decryptedData, &details); err != nil {
		dialog.ShowError(fmt.Errorf("failed to parse JSON: %v", err), w)
		return
	}

	nameEntry.SetText(details.Name)
	urlEntry.SetText(details.URL)
	passwordEntry.SetText(details.Password)
	notesEntry.SetText(details.Notes)
	usernameEntry.SetText(details.Username)
	totpEntry.SetText(details.TotpSecret)

	if len(details.TotpSecret) > 0 {
		progressBar.Show()
		totpLabel.Show()
		startTOTPUpdater(totpEntry, totpLabel, progressBar)
	} else {
		if runningTOTPUpdater {
			close(stopTOTP) // Stop only if running
			runningTOTPUpdater = false
		}
		totpLabel.SetText("")
		progressBar.SetValue(0)
		progressBar.Hide()
		totpLabel.Hide()
	}
}

func saveFile(fileName string, w fyne.Window, list *widget.List) {
	fileDetails := FileDetails{
		Name:       nameEntry.Text,
		URL:        urlEntry.Text,
		Password:   passwordEntry.Text,
		Notes:      notesEntry.Text,
		Username:   usernameEntry.Text,
		TotpSecret: totpEntry.Text,
	}

	data, err := json.MarshalIndent(fileDetails, "", "  ")
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to create JSON: %v", err), w)
		return
	}

	encryptedData, err := crypto.Encrypt(data, passwordHash)
	if err != nil {
		dialog.ShowError(fmt.Errorf("encryption failed: %v", err), w)
		return
	}

	filePath := filepath.Join(storeDir, fileName)
	err = os.WriteFile(filePath, encryptedData, 0644)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w)
		return
	}
	if !editMode {
		listItems = append(listItems, fileName)
		list.Refresh()
	}
}

func startTOTPUpdater(totpEntry *widget.Entry, totpLabel *widget.Label, progressBar *widget.ProgressBar) {
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
					progressBar.SetValue(0)
					return
				}

				timeLeft := 30 - (time.Now().Unix() % 30)
				progressBar.SetValue(float64(timeLeft) / 30.0)
				totpLabel.SetText(utils.GenerateTOTP(secret))

				time.Sleep(1 * time.Second)
			}
		}
	}()
}
