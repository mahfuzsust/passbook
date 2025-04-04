package main

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"os"
	"passbook/crypto"
	"passbook/utils"
	"path/filepath"
	"strings"
)

var passwordHash = "$2a$14$zWwEnTtOPXXo4/3KryB.s.2ggEJeeulAm5hVXMq3kZKD7p6RieBfW" // In-memory password
var storeDir = filepath.Join(os.Getenv("HOME"), ".my_store")
var listItems []string

// Global variables for form entries
var nameEntry, urlEntry, passwordEntry, notesEntry, usernameEntry *widget.Entry

// FileDetails Struct to hold the details of a file
type FileDetails struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Password string `json:"password"`
	Notes    string `json:"notes"`
	Username string `json:"username"`
}

func main() {
	if utils.MissingDirectory(storeDir) {
		return
	}

	a := app.New()
	crateLoginWindow(a)
}

func crateLoginWindow(a fyne.App) {
	w := a.NewWindow("Login")
	w.Resize(fyne.NewSize(400, 300))

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter Password")
	loginButton := widget.NewButton("Login", func() {
		handleLogin(passwordEntry.Text, w, a)
	})

	passwordEntry.OnSubmitted = func(s string) {
		handleLogin(s, w, a)
	}

	w.SetContent(container.NewVBox(
		widget.NewLabel("Login"),
		passwordEntry,
		loginButton,
	))
	w.ShowAndRun()
}

func handleLogin(passwordInput string, w fyne.Window, a fyne.App) {
	if crypto.VerifyPassword(passwordInput, passwordHash) {
		w.Close()
		showMainWindow(a)
	} else {
		dialog.ShowError(nil, w)
	}
}

func showMainWindow(a fyne.App) {
	w := a.NewWindow("Main App")
	w.Resize(fyne.NewSize(800, 600))

	// Left Side (List + Search)
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search...")

	// Load the list of files from the store directory
	updateList()

	// List view
	list := widget.NewList(
		func() int { return len(listItems) },
		func() fyne.CanvasObject { return widget.NewLabel("Item") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(listItems[i])
		},
	)

	// Click a file to load it into the right panel
	list.OnSelected = func(id widget.ListItemID) {
		loadFile(listItems[id], w)
	}

	// Add Button
	addButton := widget.NewButton("Add", func() {
		clearFields()
	})

	leftContent := container.NewBorder(searchEntry, addButton, nil, nil, list)

	// Right Side (Detail View) - Create entry widgets globally
	nameEntry = widget.NewEntry()
	urlEntry = widget.NewEntry()
	passwordEntry = widget.NewPasswordEntry()
	notesEntry = widget.NewMultiLineEntry()
	usernameEntry = widget.NewEntry()

	// Clipboard copy buttons
	copyUsernameBtn := widget.NewButton("Copy", func() {
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(usernameEntry.Text)
	})
	copyPasswordBtn := widget.NewButton("Copy", func() {
		fyne.CurrentApp().Driver().AllWindows()[0].Clipboard().SetContent(passwordEntry.Text)
	})

	// Save Button
	saveButton := widget.NewButton("Save", func() {
		fileName := nameEntry.Text + ".json"
		saveFile(fileName, w, list)
	})

	rightContent := container.NewVBox(
		widget.NewLabel("Name"), nameEntry,
		widget.NewLabel("Username"), usernameEntry, copyUsernameBtn,
		widget.NewLabel("Password"), passwordEntry, copyPasswordBtn,
		widget.NewLabel("URL"), urlEntry,
		widget.NewLabel("Notes"), notesEntry,
		saveButton,
	)

	// Split Layout (30%-70%)
	split := container.NewHSplit(leftContent, rightContent)
	split.SetOffset(0.3) // 30% left, 70% right

	w.SetContent(split)
	w.Show()
	list.Refresh()
}

func updateList() {
	// Read files from the store directory
	files, err := os.ReadDir(storeDir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	// Filter out non-JSON files and create the list
	listItems = []string{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			listItems = append(listItems, file.Name())
		}
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
	filePath := filepath.Join(storeDir, fileName)

	// Read the JSON file
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
	// Parse the JSON
	var details FileDetails
	if err := json.Unmarshal(decryptedData, &details); err != nil {
		dialog.ShowError(fmt.Errorf("failed to parse JSON: %v", err), w)
		return
	}

	// Populate the fields
	nameEntry.SetText(details.Name)
	urlEntry.SetText(details.URL)
	passwordEntry.SetText(details.Password)
	notesEntry.SetText(details.Notes)
	usernameEntry.SetText(details.Username)
}

func saveFile(fileName string, w fyne.Window, list *widget.List) {
	// Collect the data from the fields
	fileDetails := FileDetails{
		Name:     nameEntry.Text,
		URL:      urlEntry.Text,
		Password: passwordEntry.Text,
		Notes:    notesEntry.Text,
		Username: usernameEntry.Text,
	}

	// Create JSON data
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

	// Save to file
	filePath := filepath.Join(storeDir, fileName)
	err = os.WriteFile(filePath, encryptedData, 0644)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to save file: %v", err), w)
		return
	}

	listItems = append(listItems, fileName)
	list.Refresh()
}
