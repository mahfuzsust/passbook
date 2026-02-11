package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"
)

// --- Data Models ---

type PasswordHistory struct {
	Password string `json:"password"`
	Date     string `json:"date"`
}

type Entry struct {
	Title      string            `json:"title"`
	Username   string            `json:"username"`
	Password   string            `json:"password"`
	Link       string            `json:"link"`
	TOTPSecret string            `json:"totp_secret"`
	CustomText string            `json:"custom_text"`
	History    []PasswordHistory `json:"history,omitempty"`
}

type AppConfig struct {
	DataDir string `json:"data_dir"`
}

// --- Globals & Palette ---

const (
	colorUnfocusedBg = tcell.Color236
	colorFocusedBg   = tcell.Color24
)

var (
	app               = tview.NewApplication()
	pages             = tview.NewPages()
	masterKey         []byte
	dataDir           = "~/.passbook/data" // New default path
	currentFile       string
	editingFilename   string
	currentEnt        Entry
	editingEnt        Entry
	lastActivity      time.Time
	lastGeneratedPass string

	// State for the collision modal
	pendingSaveData []byte
	pendingFilename string

	// UI Components
	loginForm      *tview.Form
	searchField    *tview.InputField
	fileList       *tview.List
	rightPages     *tview.Pages
	viewFlex       *tview.Flex
	settingsForm   *tview.Form
	deleteModal    *tview.Modal
	collisionModal *tview.Modal
	errorModal     *tview.Modal
	historyList    *tview.List

	// View Mode Components
	viewTitle    *tview.TextView
	viewUsername *tview.TextView
	viewPassword *tview.TextView
	viewLink     *tview.TextView
	viewTOTP     *tview.TextView
	viewTOTPBar  *tview.TextView
	viewCustom   *tview.TextView
	viewStatus   *tview.TextView
	showPassword bool

	// Custom Editor Components
	editorLayout *tview.Flex
	editTitle    *tview.InputField
	editUser     *tview.InputField
	editPass     *tview.InputField
	editLink     *tview.InputField
	editTOTP     *tview.InputField
	editCustom   *tview.TextArea
	btnGenPass   *tview.Button
	btnSave      *tview.Button
	btnDelete    *tview.Button
	btnCancel    *tview.Button

	// Custom PassGen Components
	passGenLayout  *tview.Flex
	passGenPreview *tview.TextView
	passGenForm    *tview.Form
)

// --- Styling Helpers ---

func styleInput(field *tview.InputField) *tview.InputField {
	field.SetFieldBackgroundColor(colorUnfocusedBg)
	field.SetFocusFunc(func() { field.SetFieldBackgroundColor(colorFocusedBg) })
	field.SetBlurFunc(func() { field.SetFieldBackgroundColor(colorUnfocusedBg) })
	return field
}

func styleButton(b *tview.Button) *tview.Button {
	b.SetBackgroundColor(colorUnfocusedBg)
	b.SetLabelColor(tcell.ColorWhite)
	b.SetFocusFunc(func() {
		b.SetLabelColor(colorFocusedBg)
		b.SetBackgroundColor(tcell.ColorWhite)
	})
	b.SetBlurFunc(func() {
		b.SetBackgroundColor(colorUnfocusedBg)
		b.SetLabelColor(tcell.ColorWhite)
	})
	return b
}

func styleFormInputs(f *tview.Form) {
	for i := 0; i < f.GetFormItemCount(); i++ {
		if input, ok := f.GetFormItem(i).(*tview.InputField); ok {
			styleInput(input)
		}
	}
}

func styleFormButtons(f *tview.Form) {
	for i := 0; i < f.GetButtonCount(); i++ {
		styleButton(f.GetButton(i))
	}
}

// --- Path Expansion Helper ---

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// --- Config Management ---

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json" // Fallback if home directory cannot be determined
	}
	return filepath.Join(home, ".passbook", "config.json")
}

func loadConfig() {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg AppConfig
		if err := json.Unmarshal(data, &cfg); err == nil && cfg.DataDir != "" {
			dataDir = cfg.DataDir
			return
		}
	}
	// If it doesn't exist or is corrupted, create the default setup
	saveConfig()
}

func saveConfig() {
	configPath := getConfigPath()
	// Ensure the ~/.passbook directory exists
	err := os.MkdirAll(filepath.Dir(configPath), 0700)
	if err != nil {
		return
	}

	cfg := AppConfig{DataDir: dataDir}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	// Save the config file (0600 secures it to only your user account)
	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		return
	}
}

// --- Crypto Functions ---

func deriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

func encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// --- Password Generator ---

func generatePassword(length int, useUpper, useLower, useSpecial bool) string {
	charset := ""
	if useUpper {
		charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	if useLower {
		charset += "abcdefghijklmnopqrstuvwxyz"
	}
	if useSpecial {
		charset += "!@#$%^&*()-_=+[]{}|;:,.<>?"
	}
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	}

	pass := make([]byte, length)
	for i := range pass {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		pass[i] = charset[num.Int64()]
	}
	return string(pass)
}

// --- Main Application ---

func main() {
	loadConfig()
	err := os.MkdirAll(expandPath(dataDir), 0700)
	if err != nil {
		return
	}
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
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
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

func lockApp() {
	masterKey = nil
	currentFile = ""
	editingFilename = ""
	currentEnt = Entry{}
	fileList.Clear()
	clearViewPane()
	loginForm.GetFormItemByLabel("Master Password").(*tview.InputField).SetText("")
	pages.SwitchToPage("login")
	app.SetFocus(loginForm)
	lastActivity = time.Now()
}

func drawTOTP() {
	if currentFile == "" {
		return
	}

	if currentEnt.TOTPSecret != "" {
		now := time.Now()
		code, err := totp.GenerateCode(strings.ReplaceAll(currentEnt.TOTPSecret, " ", ""), now)
		if err == nil {
			viewTOTP.SetText(code)
			sec := now.Unix() % 30
			remain := 30 - sec
			bars := int((float64(remain) / 30.0) * 20.0)
			barStr := strings.Repeat("█", bars) + strings.Repeat("▒", 20-bars)

			color := "green"
			if remain <= 5 {
				color = "red"
			} else if remain <= 10 {
				color = "yellow"
			}

			viewTOTPBar.SetText(fmt.Sprintf("[%s]%02ds [%s][-]", color, remain, barStr))
		} else {
			viewTOTP.SetText("Invalid Secret")
			viewTOTPBar.SetText("")
		}
	} else {
		viewTOTP.SetText("None")
		viewTOTPBar.SetText("")
	}
}

func notifyCopied(item string) {
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied to clipboard![-]", item))
	go func() {
		time.Sleep(2 * time.Second)
		app.QueueUpdateDraw(func() { viewStatus.SetText("") })
	}()
}

func copySensitive(text, itemLabel string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		return
	}
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied to clipboard (will clear in 30s)[-]", itemLabel))

	go func() {
		time.Sleep(2 * time.Second)
		app.QueueUpdateDraw(func() { viewStatus.SetText("") })
		time.Sleep(28 * time.Second)

		currentClip, err := clipboard.ReadAll()
		if err == nil && currentClip == text {
			err := clipboard.WriteAll("")
			if err != nil {
				return
			}
			app.QueueUpdateDraw(func() {
				viewStatus.SetText(fmt.Sprintf("[yellow]✗ %s wiped from clipboard for security[-]", itemLabel))
				go func() {
					time.Sleep(3 * time.Second)
					app.QueueUpdateDraw(func() { viewStatus.SetText("") })
				}()
			})
		}
	}()
}

func makeRow(label string, content *tview.TextView, buttons ...*tview.Button) *tview.Flex {
	f := tview.NewFlex().SetDirection(tview.FlexColumn)
	f.AddItem(tview.NewTextView().SetText(label).SetTextColor(tcell.ColorYellow), 12, 0, false)
	f.AddItem(content, 0, 1, false)
	for _, b := range buttons {
		f.AddItem(tview.NewTextView().SetText(" "), 1, 0, false)
		f.AddItem(b, 9, 0, false)
	}
	return f
}

func makeEditRow(label string, input tview.Primitive) *tview.Flex {
	f := tview.NewFlex().SetDirection(tview.FlexColumn)
	f.AddItem(tview.NewTextView().SetText(label).SetTextColor(tcell.ColorYellow), 15, 0, false)
	f.AddItem(input, 0, 1, true)
	return f
}

func setupUI() {
	tview.Styles.ContrastBackgroundColor = colorUnfocusedBg
	tview.Styles.TitleColor = tcell.ColorLightSkyBlue

	// 1. Login Page
	loginForm = tview.NewForm()
	loginForm.AddPasswordField("Master Password", "", 0, '*', nil).
		AddButton("Login", func() {
			pwd := loginForm.GetFormItemByLabel("Master Password").(*tview.InputField).GetText()
			if pwd != "" {
				masterKey = deriveKey(pwd)
				loadFiles("")
				pages.SwitchToPage("main")
				app.SetFocus(fileList)
			}
		}).
		AddButton("Quit", func() { app.Stop() })
	loginForm.SetBorder(true).SetTitle(" Login ").SetTitleAlign(tview.AlignCenter)
	styleFormInputs(loginForm)
	styleFormButtons(loginForm)

	loginFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(nil, 0, 1, false).AddItem(loginForm, 9, 1, true).AddItem(nil, 0, 1, false), 60, 1, true).
		AddItem(nil, 0, 1, false)

	// 2. Main Layout - Left Side
	searchField = styleInput(tview.NewInputField().SetLabel("Search: "))
	searchField.SetChangedFunc(func(text string) { loadFiles(text) })

	fileList = tview.NewList().ShowSecondaryText(false)
	fileList.SetBorder(true).SetTitle(" Files ")
	fileList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		currentFile = mainText
		showContent(mainText)
	})

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchField, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(fileList, 0, 1, true)
	leftFlex.SetBorder(true).SetTitle(" Vault (Ctrl+A Add) ")

	// 3. Main Layout - Right Side
	emptyView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("\n\n\n[yellow]No entry selected.[-]\n\nPress [green]Ctrl+A[-] to create a new entry.")

	viewFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	viewTitle = tview.NewTextView().SetDynamicColors(true)
	viewUsername = tview.NewTextView().SetDynamicColors(true)
	viewPassword = tview.NewTextView().SetDynamicColors(true)
	viewLink = tview.NewTextView().SetDynamicColors(true)
	viewTOTP = tview.NewTextView().SetDynamicColors(true)
	viewTOTPBar = tview.NewTextView().SetDynamicColors(true)
	viewCustom = tview.NewTextView().SetDynamicColors(true)
	viewStatus = tview.NewTextView().SetDynamicColors(true)

	btnCopyUser := styleButton(tview.NewButton("Copy").SetSelectedFunc(func() {
		err := clipboard.WriteAll(currentEnt.Username)
		if err != nil {
			return
		}
		notifyCopied("Username")
	}))
	btnCopyPass := styleButton(tview.NewButton("Copy").SetSelectedFunc(func() {
		copySensitive(currentEnt.Password, "Password")
	}))
	btnViewPass := styleButton(tview.NewButton("View").SetSelectedFunc(func() {
		showPassword = !showPassword
		updateViewPane()
	}))
	btnHistory := styleButton(tview.NewButton("History").SetSelectedFunc(func() {
		showHistory()
	}))
	btnCopyTOTP := styleButton(tview.NewButton("Copy").SetSelectedFunc(func() {
		if currentEnt.TOTPSecret != "" {
			code, _ := totp.GenerateCode(strings.ReplaceAll(currentEnt.TOTPSecret, " ", ""), time.Now())
			copySensitive(code, "TOTP")
		}
	}))

	viewFlex.AddItem(makeRow("Title:", viewTitle), 1, 0, false)
	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
	viewFlex.AddItem(makeRow("Username:", viewUsername, btnCopyUser), 1, 0, false)
	viewFlex.AddItem(makeRow("Password:", viewPassword, btnViewPass, btnCopyPass, btnHistory), 1, 0, false)
	viewFlex.AddItem(makeRow("Link:", viewLink), 1, 0, false)
	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
	viewFlex.AddItem(makeRow("TOTP:", viewTOTP, btnCopyTOTP), 1, 0, false)
	viewFlex.AddItem(makeRow("", viewTOTPBar), 1, 0, false)
	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
	viewFlex.AddItem(tview.NewTextView().SetText("[yellow]Custom Text:[-]").SetDynamicColors(true), 1, 0, false)
	viewFlex.AddItem(viewCustom, 0, 1, false)
	viewFlex.AddItem(viewStatus, 1, 0, false)

	rightPages = tview.NewPages()
	rightPages.SetBorder(true).SetTitle(" Contents (Ctrl+E Edit | Ctrl+D Delete | Ctrl+O Settings) ")
	rightPages.AddPage("empty", emptyView, true, true)
	rightPages.AddPage("content", viewFlex, true, false)

	mainFlex := tview.NewFlex().AddItem(leftFlex, 35, 1, true).AddItem(rightPages, 0, 2, false)

	mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlA {
			openEditor(Entry{}, "")
			return nil
		}
		if event.Key() == tcell.KeyCtrlE {
			if currentFile != "" {
				openEditor(currentEnt, currentFile)
			}
			return nil
		}
		if event.Key() == tcell.KeyCtrlD {
			if currentFile != "" {
				showDeleteModal()
			}
			return nil
		}
		if event.Key() == tcell.KeyCtrlF {
			app.SetFocus(searchField)
			return nil
		}
		if event.Key() == tcell.KeyCtrlQ {
			app.Stop()
			return nil
		}
		if event.Key() == tcell.KeyCtrlO {
			openSettings()
			return nil
		}
		if event.Key() == tcell.KeyEsc {
			app.SetFocus(fileList)
			return nil
		}
		return event
	})

	// 4. Custom Editor Layout
	editTitle = styleInput(tview.NewInputField())
	editUser = styleInput(tview.NewInputField())
	editPass = styleInput(tview.NewInputField())
	editLink = styleInput(tview.NewInputField())
	editTOTP = styleInput(tview.NewInputField())

	editCustom = tview.NewTextArea()
	unfocusedText := tcell.StyleDefault.Background(colorUnfocusedBg).Foreground(tview.Styles.PrimaryTextColor)
	focusedText := tcell.StyleDefault.Background(colorFocusedBg).Foreground(tview.Styles.PrimaryTextColor)
	editCustom.SetTextStyle(unfocusedText)
	editCustom.SetFocusFunc(func() { editCustom.SetTextStyle(focusedText) })
	editCustom.SetBlurFunc(func() { editCustom.SetTextStyle(unfocusedText) })

	btnGenPass = styleButton(tview.NewButton("Gen Pass").SetSelectedFunc(func() { openPassGen() }))
	btnSave = styleButton(tview.NewButton("Save").SetSelectedFunc(func() { saveEntry() }))
	btnDelete = styleButton(tview.NewButton("Delete").SetSelectedFunc(func() { showDeleteModal() }))
	btnCancel = styleButton(tview.NewButton("Cancel").SetSelectedFunc(func() { pages.SwitchToPage("main") }))

	passEditRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewTextView().SetText("Password:").SetTextColor(tcell.ColorYellow), 15, 0, false).
		AddItem(editPass, 0, 1, true).
		AddItem(tview.NewTextView().SetText(" "), 1, 0, false).
		AddItem(btnGenPass, 10, 0, false)

	actionButtons := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(btnSave, 10, 0, false).
		AddItem(tview.NewTextView().SetText(" "), 1, 0, false).
		AddItem(btnDelete, 10, 0, false).
		AddItem(tview.NewTextView().SetText(" "), 1, 0, false).
		AddItem(btnCancel, 10, 0, false).
		AddItem(nil, 0, 1, false)

	editorLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(makeEditRow("File/Title:", editTitle), 1, 0, true).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(makeEditRow("Username:", editUser), 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(passEditRow, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(makeEditRow("Link:", editLink), 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(makeEditRow("TOTP Secret:", editTOTP), 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(tview.NewTextView().SetText("Custom Text:").SetTextColor(tcell.ColorYellow), 1, 0, false).
		AddItem(editCustom, 0, 1, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(actionButtons, 1, 0, false)
	editorLayout.SetBorder(true).SetTitle(" Add / Edit Entry ")

	focusableElements := []tview.Primitive{
		editTitle, editUser, editPass, btnGenPass, editLink, editTOTP, editCustom, btnSave, btnDelete, btnCancel,
	}
	editorLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			for i, p := range focusableElements {
				if p.HasFocus() {
					app.SetFocus(focusableElements[(i+1)%len(focusableElements)])
					return nil
				}
			}
		} else if event.Key() == tcell.KeyBacktab {
			for i, p := range focusableElements {
				if p.HasFocus() {
					app.SetFocus(focusableElements[(i-1+len(focusableElements))%len(focusableElements)])
					return nil
				}
			}
		}
		return event
	})

	// 5. PassGen Modal
	passGenPreview = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)

	passGenForm = tview.NewForm().
		AddInputField("Length", "16", 10, tview.InputFieldInteger, nil).
		AddCheckbox("A-Z", true, nil).
		AddCheckbox("a-z", true, nil).
		AddCheckbox("Special Chars", true, nil).
		AddButton("Refresh", func() { updatePassPreview() }).
		AddButton("Use", func() {
			editPass.SetText(lastGeneratedPass)
			pages.SwitchToPage("editor")
			app.SetFocus(editLink)
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("editor")
			app.SetFocus(editLink)
		})
	styleFormInputs(passGenForm)
	styleFormButtons(passGenForm)

	passGenLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Generated Password:").SetTextColor(tcell.ColorYellow), 1, 0, false).
		AddItem(passGenPreview, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(passGenForm, 0, 1, true)
	passGenLayout.SetBorder(true).SetTitle(" Generate Password ")

	// 6. Settings Modal
	settingsForm = tview.NewForm()
	settingsForm.SetBorder(true).SetTitle(" Settings ")

	// 7. Modals
	deleteModal = tview.NewModal().
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" && currentFile != "" {
				err := os.Remove(filepath.Join(expandPath(dataDir), currentFile))
				if err != nil {
					return
				}
				currentFile = ""
				loadFiles(searchField.GetText())
			}
			pages.SwitchToPage("main")
			app.SetFocus(fileList)
		})

	collisionModal = tview.NewModal().
		AddButtons([]string{"Replace", "Add Suffix", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Cancel" {
				pages.SwitchToPage("editor")
				app.SetFocus(editTitle)
			} else if buttonLabel == "Replace" {
				commitSave(pendingFilename, pendingSaveData)
			} else if buttonLabel == "Add Suffix" {
				base := strings.TrimSuffix(pendingFilename, ".md")
				counter := 1
				var newFilename string
				for {
					newFilename = fmt.Sprintf("%s_%d.md", base, counter)
					if _, err := os.Stat(filepath.Join(expandPath(dataDir), newFilename)); os.IsNotExist(err) {
						break
					}
					counter++
				}
				commitSave(newFilename, pendingSaveData)
			}
		})

	errorModal = tview.NewModal().
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.SwitchToPage("editor")
			app.SetFocus(editTitle)
		})

	// 8. History Modal
	historyList = tview.NewList().ShowSecondaryText(true)
	historyList.SetBorder(true).SetTitle(" Password History (Esc to Close) ")
	historyList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEnter {
			pages.SwitchToPage("main")
			app.SetFocus(rightPages)
			return nil
		}
		return event
	})

	// Assemble Pages
	pages.AddPage("login", loginFlex, true, true)
	pages.AddPage("main", mainFlex, true, false)
	pages.AddPage("editor", centeredModal(editorLayout, 70, 24), true, false)
	pages.AddPage("passgen", centeredModal(passGenLayout, 45, 17), true, false)
	pages.AddPage("settings", centeredModal(settingsForm, 50, 11), true, false)
	pages.AddPage("delete", deleteModal, true, false)
	pages.AddPage("collision", collisionModal, true, false)
	pages.AddPage("error", errorModal, true, false)
	pages.AddPage("history", centeredModal(historyList, 50, 15), true, false)
}

func centeredModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

// --- Logic ---

func loadFiles(filter string) {
	fileList.Clear()
	entries, _ := os.ReadDir(expandPath(dataDir))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			if filter == "" || strings.Contains(strings.ToLower(e.Name()), strings.ToLower(filter)) {
				fileList.AddItem(e.Name(), "", 0, nil)
			}
		}
	}
	if fileList.GetItemCount() > 0 {
		name, _ := fileList.GetItemText(0)
		currentFile = name
		showContent(name)
	} else {
		currentFile = ""
		currentEnt = Entry{}
		clearViewPane()
	}
}

func showContent(filename string) {
	path := filepath.Join(expandPath(dataDir), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		clearViewPane()
		return
	}

	decrypted, err := decrypt(data)
	if err != nil {
		clearViewPane()
		return
	}

	var entry Entry
	if err := json.Unmarshal(decrypted, &entry); err != nil {
		clearViewPane()
		return
	}

	currentEnt = entry
	showPassword = false

	rightPages.SwitchToPage("content")
	updateViewPane()
}

func updateViewPane() {
	viewTitle.SetText(currentEnt.Title)
	viewUsername.SetText(currentEnt.Username)
	viewLink.SetText(currentEnt.Link)
	viewCustom.SetText(currentEnt.CustomText)

	if showPassword {
		viewPassword.SetText(currentEnt.Password)
	} else {
		viewPassword.SetText(strings.Repeat("*", len(currentEnt.Password)))
	}
	drawTOTP()
}

func clearViewPane() {
	viewTitle.SetText("")
	viewUsername.SetText("")
	viewPassword.SetText("")
	viewLink.SetText("")
	viewTOTP.SetText("")
	viewTOTPBar.SetText("")
	viewCustom.SetText("")
	viewStatus.SetText("")
	rightPages.SwitchToPage("empty")
}

func showDeleteModal() {
	if currentFile == "" {
		return
	}
	deleteModal.SetText(fmt.Sprintf("Are you sure you want to delete '%s'?", currentFile))
	pages.SwitchToPage("delete")
}

func showHistory() {
	historyList.Clear()
	if len(currentEnt.History) == 0 {
		historyList.AddItem("No previous passwords found.", "", 0, nil)
	} else {
		for i := len(currentEnt.History) - 1; i >= 0; i-- {
			h := currentEnt.History[i]
			historyList.AddItem(h.Password, "Changed: "+h.Date, 0, nil)
		}
	}
	historyList.AddItem("[ Close ]", "Return to vault", 'c', func() {
		pages.SwitchToPage("main")
		app.SetFocus(rightPages)
	})

	pages.SwitchToPage("history")
	app.SetFocus(historyList)
}

func openEditor(entry Entry, filename string) {
	editingFilename = filename
	editingEnt = entry

	editTitle.SetText(entry.Title)
	editUser.SetText(entry.Username)
	editPass.SetText(entry.Password)
	editLink.SetText(entry.Link)
	editTOTP.SetText(entry.TOTPSecret)
	editCustom.SetText(entry.CustomText, false)

	pages.SwitchToPage("editor")
	app.SetFocus(editTitle)
}

func updatePassPreview() {
	lengthStr := passGenForm.GetFormItemByLabel("Length").(*tview.InputField).GetText()
	length, err := strconv.Atoi(lengthStr)
	if err != nil || length <= 0 {
		length = 16
	}
	upper := passGenForm.GetFormItemByLabel("A-Z").(*tview.Checkbox).IsChecked()
	lower := passGenForm.GetFormItemByLabel("a-z").(*tview.Checkbox).IsChecked()
	special := passGenForm.GetFormItemByLabel("Special Chars").(*tview.Checkbox).IsChecked()

	lastGeneratedPass = generatePassword(length, upper, lower, special)
	passGenPreview.SetText("[green]" + lastGeneratedPass + "[-]")
}

func openPassGen() {
	updatePassPreview()
	pages.SwitchToPage("passgen")
	app.SetFocus(passGenForm)
}

func openSettings() {
	settingsForm.Clear(true)
	settingsForm.
		AddInputField("Data Directory", dataDir, 50, nil, nil).
		AddButton("Save", func() {
			newDir := settingsForm.GetFormItemByLabel("Data Directory").(*tview.InputField).GetText()
			if newDir != "" {
				dataDir = newDir
				err := os.MkdirAll(expandPath(dataDir), 0700)
				if err != nil {
					return
				}
				saveConfig()
				loadFiles(searchField.GetText())
			}
			pages.SwitchToPage("main")
			app.SetFocus(fileList)
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("main")
			app.SetFocus(fileList)
		})
	styleFormInputs(settingsForm)
	styleFormButtons(settingsForm)
	pages.SwitchToPage("settings")
}

func saveEntry() {
	title := editTitle.GetText()
	if title == "" {
		title = "Untitled"
	}
	newPass := editPass.GetText()

	history := editingEnt.History
	if editingEnt.Password != "" && editingEnt.Password != newPass {
		history = append(history, PasswordHistory{
			Password: editingEnt.Password,
			Date:     time.Now().Format("2006-01-02 15:04:05"),
		})
	}

	entry := Entry{
		Title:      title,
		Username:   editUser.GetText(),
		Password:   newPass,
		Link:       editLink.GetText(),
		TOTPSecret: editTOTP.GetText(),
		CustomText: editCustom.GetText(),
		History:    history,
	}

	data, _ := json.Marshal(entry)
	encrypted, err := encrypt(data)
	if err != nil {
		return
	}

	filename := title
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	targetPath := filepath.Join(expandPath(dataDir), filename)
	_, err = os.Stat(targetPath)
	fileExists := !os.IsNotExist(err)

	if fileExists && editingFilename != filename {
		if editingFilename == "" {
			pendingSaveData = encrypted
			pendingFilename = filename
			collisionModal.SetText(fmt.Sprintf("A file named '%s' already exists.\nWhat would you like to do?", filename))
			pages.SwitchToPage("collision")
			app.SetFocus(collisionModal)
		} else {
			errorModal.SetText(fmt.Sprintf("Cannot rename to '%s': A file with this name already exists.\nPlease choose a different title.", title))
			pages.SwitchToPage("error")
			app.SetFocus(errorModal)
		}
		return
	}

	commitSave(filename, encrypted)
}

func commitSave(filename string, encryptedData []byte) {
	if editingFilename != "" && editingFilename != filename {
		err := os.Remove(filepath.Join(expandPath(dataDir), editingFilename))
		if err != nil {
			return
		}
	}

	err := os.WriteFile(filepath.Join(expandPath(dataDir), filename), encryptedData, 0600)
	if err != nil {
		return
	}

	pages.SwitchToPage("main")
	loadFiles(searchField.GetText())
}
