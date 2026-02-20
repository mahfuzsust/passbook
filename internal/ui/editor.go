package ui

import (
	"fmt"
	"os"
	"passbook/internal/config"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"passbook/internal/crypto"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/protobuf/proto"
)

func setupEditor() {
	uiCreateList = tview.NewList().ShowSecondaryText(false)
	uiCreateList.AddItem("Login", "Password & 2FA", 'l', func() { newEntry(TypeLogin) })
	uiCreateList.AddItem("Card", "Credit/Debit Details", 'c', func() { newEntry(TypeCard) })
	uiCreateList.AddItem("Note", "Secure Text", 'n', func() { newEntry(TypeNote) })
	uiCreateList.AddItem("File", "Encrypted Attachments", 'f', func() { newEntry(TypeFile) })
	uiCreateList.SetBorder(true).SetTitle(" Create New ")
	uiPages.AddPage("create_menu", newResponsiveModal(uiCreateList, 30, 14, 50, 20, 0.4, 0.5), true, false)

	uiEditorForm = tview.NewForm()
	uiAttachList = tview.NewList().ShowSecondaryText(false).SetMainTextColor(tcell.ColorGreen)

	uiAttachFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	uiAttachFlex.AddItem(tview.NewTextView().SetText(" Attachments:").SetTextColor(tcell.ColorYellow), 1, 0, false)
	uiAttachFlex.AddItem(uiAttachList, 0, 1, true)

	uiEditorLayout = tview.NewFlex().SetDirection(tview.FlexRow)
	uiEditorLayout.AddItem(uiEditorForm, 0, 1, true)
	uiEditorLayout.SetBorder(true).SetTitle(" Edit Entry ")
	uiPages.AddPage("editor", newResponsiveModal(uiEditorLayout, 60, 25, 120, 50, 0.8, 0.85), true, false)

	setupFileBrowser()
	setupPassGen()
	setupCollisionModals()
}

func setupFileBrowser() {
	uiFileBrowser = tview.NewTreeView()
	uiFileBrowser.SetBorder(true).SetTitle(" Select File (Enter to Pick, Esc Cancel) ")
	uiFileBrowser.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("editor")
		}
		return event
	})

	uiFileBrowserModal = newResponsiveModal(uiFileBrowser, 50, 20, 100, 40, 0.7, 0.75)
	uiPages.AddPage("filebrowser", uiFileBrowserModal, true, false)
}

func setupPassGen() {
	uiPassGenPreview = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	uiPassGenForm = tview.NewForm()
	uiPassGenForm.AddInputField("Length", "28", 10, tview.InputFieldInteger, nil)
	uiPassGenForm.AddCheckbox("A-Z", true, nil)
	uiPassGenForm.AddCheckbox("a-z", true, nil)
	uiPassGenForm.AddCheckbox("Special", true, nil)
	uiPassGenForm.AddButton("Refresh", func() { updatePassPreview() })
	uiPassGenForm.AddButton("Use", func() {
		if uiEditorPasswordField != nil {
			uiEditorPasswordField.SetText(uiLastGeneratedPass)
			uiPages.SwitchToPage("editor")
			uiApp.SetFocus(uiEditorPasswordField)
		} else {
			uiPages.SwitchToPage("editor")
		}
	})
	uiPassGenForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("editor")
			if uiEditorPasswordField != nil {
				uiApp.SetFocus(uiEditorPasswordField)
			}
		}
		return event
	})
	uiPassGenForm.AddButton("Cancel", func() {
		uiPages.SwitchToPage("editor")
		if uiEditorPasswordField != nil {
			uiApp.SetFocus(uiEditorPasswordField)
		}
	})
	styleForm(uiPassGenForm)

	uiPassGenLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Generated:").SetTextColor(tcell.ColorYellow), 1, 0, false).
		AddItem(uiPassGenPreview, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(uiPassGenForm, 0, 1, true)
	uiPassGenLayout.SetBorder(true).SetTitle(" Generator ")
	uiPages.AddPage("passgen", newResponsiveModal(uiPassGenLayout, 45, 20, 70, 30, 0.6, 0.6), true, false)
}

func setupCollisionModals() {
	uiCollisionModal = tview.NewModal().
		AddButtons([]string{"Replace", "Add Suffix", "Cancel"}).
		SetDoneFunc(func(index int, label string) {
			if label == "Cancel" {
				uiPages.SwitchToPage("editor")
			} else if label == "Replace" {
				commitSave(uiPendingPath, uiPendingSaveData)
			} else if label == "Add Suffix" {
				dir, base := filepath.Dir(uiPendingPath), strings.TrimSuffix(filepath.Base(uiPendingPath), ".pb")
				counter := 1
				var newPath string
				for {
					newPath = filepath.Join(dir, fmt.Sprintf("%s_%d.pb", base, counter))
					if _, err := os.Stat(newPath); os.IsNotExist(err) {
						break
					}
					counter++
				}
				commitSave(newPath, uiPendingSaveData)
			}
		})
	uiPages.AddPage("collision", uiCollisionModal, true, false)

	uiErrorModal = tview.NewModal().AddButtons([]string{"OK"}).
		SetDoneFunc(func(i int, l string) { uiPages.SwitchToPage("editor") })
	uiPages.AddPage("error", uiErrorModal, true, false)
}

func updatePassPreview() {
	lStr := uiPassGenForm.GetFormItemByLabel("Length").(*tview.InputField).GetText()
	l, _ := strconv.Atoi(lStr)
	upper := uiPassGenForm.GetFormItemByLabel("A-Z").(*tview.Checkbox).IsChecked()
	lower := uiPassGenForm.GetFormItemByLabel("a-z").(*tview.Checkbox).IsChecked()
	special := uiPassGenForm.GetFormItemByLabel("Special").(*tview.Checkbox).IsChecked()
	uiLastGeneratedPass = crypto.GeneratePassword(l, upper, lower, special)
	uiPassGenPreview.SetText("[green]" + uiLastGeneratedPass)
}

func showCreateMenu() {
	uiPages.SwitchToPage("create_menu")
}

func newEntry(t EntryType) {
	uiCurrentPath = ""
	ent := NewEntry(t)
	openEditor(ent)
}

func openEditor(ent *Entry) {
	uiEditingEnt = ent
	uiPendingAttachments = append([]*Attachment{}, ent.Attachments...)
	uiPendingFilePaths = make(map[string]string)

	uiEditorForm.Clear(true)
	uiEditorTitleField, uiEditorPasswordField, uiEditorSaveButton = nil, nil, nil
	uiEditorCardNumber, uiEditorExpiry, uiEditorCVV = nil, nil, nil

	if uiCurrentPath == "" {
		uiEditorLayout.SetTitle(" Add Entry ")
	} else {
		uiEditorLayout.SetTitle(" Edit Entry ")
	}

	titleField := tview.NewInputField().SetLabel("Title").SetText(ent.Title).SetFieldWidth(40)
	titleField.SetChangedFunc(func(text string) { updateEditorSaveState() })
	uiEditorTitleField = titleField
	uiEditorForm.AddFormItem(titleField)

	uiEditorLayout.RemoveItem(uiAttachFlex)
	switch EntryType(ent.Type) {
	case TypeLogin:
		ent.Attachments = nil
		uiEditorForm.AddInputField("Username", ent.Username, 40, nil, nil)

		// Store password field reference
		uiEditorPasswordField = tview.NewInputField().SetLabel("Password").SetText(ent.Password).SetFieldWidth(40)
		uiEditorForm.AddFormItem(uiEditorPasswordField)
		uiEditorForm.AddButton("generate", func() {
			updatePassPreview()
			uiPages.SwitchToPage("passgen")
		})

		uiEditorForm.AddInputField("Link", ent.Link, 40, nil, nil)
		uiEditorForm.AddInputField("TOTP Secret", ent.TotpSecret, 40, nil, nil)
	case TypeCard:
		ent.Attachments = nil

		cardNumberField := tview.NewInputField().SetLabel("Card Number").SetText(ent.CardNumber).SetFieldWidth(40)
		cardNumberField.SetAcceptanceFunc(func(text string, last rune) bool {
			if last == 0 {
				return len(text) <= 16
			}
			if len(text) > 16 {
				return false
			}
			for _, r := range text {
				if r < '0' || r > '9' {
					return false
				}
			}
			return true
		})
		cardNumberField.SetChangedFunc(func(string) { updateEditorSaveState() })
		uiEditorCardNumber = cardNumberField
		uiEditorForm.AddFormItem(cardNumberField)

		expiryField := tview.NewInputField().SetLabel("Expiry (MM/YY)").SetText(ent.Expiry).SetFieldWidth(10)
		expiryField.SetAcceptanceFunc(func(text string, last rune) bool {
			if last == 0 {
				return len(text) <= 5
			}
			if len(text) > 5 {
				return false
			}
			for i, r := range text {
				if i == 2 {
					if r != '/' {
						return false
					}
					continue
				}
				if r < '0' || r > '9' {
					return false
				}
			}
			return true
		})
		expiryField.SetChangedFunc(func(string) { updateEditorSaveState() })
		uiEditorExpiry = expiryField
		uiEditorForm.AddFormItem(expiryField)

		cvvField := tview.NewInputField().SetLabel("CVV").SetText(ent.Cvv).SetFieldWidth(5)
		cvvField.SetAcceptanceFunc(func(text string, last rune) bool {
			if last == 0 {
				return len(text) <= 3
			}
			if len(text) > 3 {
				return false
			}
			for _, r := range text {
				if r < '0' || r > '9' {
					return false
				}
			}
			return true
		})
		cvvField.SetChangedFunc(func(string) { updateEditorSaveState() })
		uiEditorCVV = cvvField
		uiEditorForm.AddFormItem(cvvField)
	case TypeFile:
		uiEditorLayout.AddItem(uiAttachFlex, 0, 0, false)

		uiEditorForm.AddButton("Browse Filesystem", func() {
			home, _ := os.UserHomeDir()
			openFileBrowser(home)
		})

		dropZone := tview.NewTextArea().
			SetLabel("Drag File Here").
			SetPlaceholder("Click here, then drop/paste a file path, then press Enter to attach").
			SetSize(5, 40)

		dropZone.SetBorder(true)
		dropZone.SetTitle(" Dropzone ")
		dropZone.SetTitleColor(tcell.ColorYellow)
		dropZone.SetBackgroundColor(tcell.ColorBlack)

		resetDropZone := func() {
			dropZone.SetText("", true)
			dropZone.SetLabel("Drag File Here")
			dropZone.SetPlaceholder("Click here, then drop/paste a file path, then press Enter to attach")
			dropZone.SetBorder(true)
			dropZone.SetTitle(" Dropzone ")
			dropZone.SetTitleColor(tcell.ColorYellow)
			dropZone.SetBackgroundColor(tcell.ColorBlack)
		}

		attachFromDropZone := func() {
			rawPath := dropZone.GetText()
			cleanPath := strings.Trim(rawPath, "\"' \n\r\t")
			if cleanPath == "" {
				return
			}

			if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
				cleanPath = strings.ReplaceAll(cleanPath, "\\ ", " ")
			}

			if strings.Contains(cleanPath, "\n") || strings.Contains(cleanPath, "\r") {
				return
			}

			fi, err := os.Stat(cleanPath)
			if err != nil || fi.IsDir() {
				return
			}

			id := fmt.Sprintf("%d", time.Now().UnixNano())
			att := &Attachment{Id: id, FileName: filepath.Base(cleanPath), Size: fi.Size()}
			uiPendingAttachments = append(uiPendingAttachments, att)
			uiPendingFilePaths[id] = cleanPath

			resetDropZone()
			refreshAttachmentList(TypeFile)
			uiApp.SetFocus(uiAttachList)
		}

		dropZone.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEnter {
				attachFromDropZone()
				return nil
			}
			return event
		})

		uiEditorForm.AddFormItem(dropZone)

		resetDropZone()
		refreshAttachmentList(TypeFile)
	}

	uiEditorForm.AddTextArea("Notes", ent.CustomText, 50, 5, 0, nil)
	saveButtonIndex := uiEditorForm.GetButtonCount()
	uiEditorForm.AddButton("Save", func() { saveEntry(EntryType(ent.Type)) })
	uiEditorSaveButton = uiEditorForm.GetButton(saveButtonIndex)
	updateEditorSaveState()
	uiEditorForm.AddButton("Cancel", func() { uiPages.SwitchToPage("main"); uiApp.SetFocus(uiTreeView) })
	styleForm(uiEditorForm)
	if uiEditorSaveButton != nil {
		uiEditorSaveButton.SetLabelColor(tcell.ColorIndianRed)
		uiEditorSaveButton.SetBackgroundColor(colorUnfocusedBg)
		uiEditorSaveButton.SetFocusFunc(func() {
			uiEditorSaveButton.SetLabelColor(tcell.ColorIndianRed)
			uiEditorSaveButton.SetBackgroundColor(tcell.ColorWhite)
		})
		uiEditorSaveButton.SetBlurFunc(func() {
			uiEditorSaveButton.SetBackgroundColor(colorUnfocusedBg)
			uiEditorSaveButton.SetLabelColor(tcell.ColorIndianRed)
		})
	}
	uiEditorForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("main")
		}
		return event
	})
	uiPages.SwitchToPage("editor")
}

func validateTitleField() (string, error) {
	if uiEditorTitleField == nil {
		return "", fmt.Errorf("title field unavailable")
	}

	title := strings.TrimSpace(uiEditorTitleField.GetText())
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	if strings.ContainsAny(title, `<>:"/\\|?*`) || title == "." || title == ".." {
		return "", fmt.Errorf("title contains invalid characters")
	}

	return title, nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validateCardFields() (string, string, string, error) {
	if uiEditingEnt == nil || EntryType(uiEditingEnt.Type) != TypeCard {
		return "", "", "", nil
	}
	if uiEditorCardNumber == nil || uiEditorExpiry == nil || uiEditorCVV == nil {
		return "", "", "", fmt.Errorf("card fields unavailable")
	}

	number := strings.TrimSpace(uiEditorCardNumber.GetText())
	expiry := strings.TrimSpace(uiEditorExpiry.GetText())
	cvv := strings.TrimSpace(uiEditorCVV.GetText())

	if number != "" {
		if len(number) != 16 || !isDigits(number) {
			return "", "", "", fmt.Errorf("card number must be 16 digits")
		}
	}

	if expiry != "" {
		if len(expiry) != 5 || expiry[2] != '/' {
			return "", "", "", fmt.Errorf("expiry must be MM/YY")
		}
		mm, yy := expiry[:2], expiry[3:]
		if !isDigits(mm) || !isDigits(yy) {
			return "", "", "", fmt.Errorf("expiry must be MM/YY")
		}
		month, _ := strconv.Atoi(mm)
		if month < 1 || month > 12 {
			return "", "", "", fmt.Errorf("expiry must be MM/YY")
		}
	}

	if cvv != "" {
		if len(cvv) != 3 || !isDigits(cvv) {
			return "", "", "", fmt.Errorf("CVV must be 3 digits")
		}
	}

	return number, expiry, cvv, nil
}

func updateEditorSaveState() {
	if uiEditorTitleField == nil || uiEditorSaveButton == nil {
		return
	}

	_, titleErr := validateTitleField()
	_, _, _, cardErr := validateCardFields()
	uiEditorSaveButton.SetDisabled(titleErr != nil || cardErr != nil)
}

func refreshAttachmentList(t EntryType) {
	uiAttachList.Clear()
	size := 0

	if t == TypeFile || len(uiPendingAttachments) > 0 {
		if len(uiPendingAttachments) > 0 {
			size = 6
			for i, att := range uiPendingAttachments {
				label := att.FileName
				if _, isNew := uiPendingFilePaths[att.Id]; isNew {
					label += " [green](New)[-]"
				}
				idx := i
				uiAttachList.AddItem(label, "Press Enter to Remove", 0, func() {
					uiPendingAttachments = append(uiPendingAttachments[:idx], uiPendingAttachments[idx+1:]...)
					refreshAttachmentList(t)
				})
			}
		}
	}
	uiEditorLayout.ResizeItem(uiAttachFlex, size, 0)
}
func openFileBrowser(path string) {
	rootDir, _ := filepath.Abs(path)
	rootNode := tview.NewTreeNode(rootDir).SetColor(tcell.ColorYellow).SetReference(rootDir)
	uiFileBrowser.SetRoot(rootNode).SetCurrentNode(rootNode)
	addNodes(rootNode, rootDir)

	uiFileBrowser.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			return
		}
		path := ref.(string)
		fi, err := os.Stat(path)
		if err != nil {
			return
		}

		if fi.IsDir() {
			if len(node.GetChildren()) == 0 {
				addNodes(node, path)
			}
			node.SetExpanded(!node.IsExpanded())
		} else {
			id := fmt.Sprintf("%d", time.Now().UnixNano())
			att := &Attachment{Id: id, FileName: filepath.Base(path), Size: fi.Size()}
			uiPendingAttachments = append(uiPendingAttachments, att)
			uiPendingFilePaths[id] = path
			refreshAttachmentList(TypeFile)
			uiPages.SwitchToPage("editor")
		}
	})
	uiPages.SwitchToPage("filebrowser")
}

func addNodes(target *tview.TreeNode, path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		node := tview.NewTreeNode(f.Name()).SetReference(filepath.Join(path, f.Name()))
		if f.IsDir() {
			node.SetColor(tcell.ColorSkyblue)
		}
		target.AddChild(node)
	}
}

func saveEntry(eType EntryType) {
	title, err := validateTitleField()
	if err != nil {
		updateEditorSaveState()
		return
	}

	cardNumber, expiry, cvv, err := validateCardFields()
	if err != nil {
		updateEditorSaveState()
		return
	}

	var priorPassword string
	var priorHistory []*PasswordHistory
	if uiEditingEnt != nil {
		priorPassword = uiEditingEnt.Password
		priorHistory = uiEditingEnt.History
	}

	ent := &Entry{
		Type:        string(eType),
		Title:       title,
		CustomText:  uiEditorForm.GetFormItemByLabel("Notes").(*tview.TextArea).GetText(),
		History:     priorHistory,
		Attachments: uiPendingAttachments,
	}

	switch eType {
	case TypeLogin:
		ent.Username = uiEditorForm.GetFormItemByLabel("Username").(*tview.InputField).GetText()

		// Get password from stored field reference
		if uiEditorPasswordField != nil {
			ent.Password = uiEditorPasswordField.GetText()
		}

		ent.Link = uiEditorForm.GetFormItemByLabel("Link").(*tview.InputField).GetText()
		ent.TotpSecret = uiEditorForm.GetFormItemByLabel("TOTP Secret").(*tview.InputField).GetText()
		if priorPassword != "" && priorPassword != ent.Password {
			ent.History = append(ent.History, &PasswordHistory{Password: priorPassword, Date: time.Now().Format("2006-01-02 15:04")})
		}
	case TypeCard:
		ent.CardNumber = cardNumber
		ent.Expiry = expiry
		ent.Cvv = cvv
	}

	bytes, _ := proto.Marshal(ent)
	enc, _ := crypto.Encrypt(uiMasterKey, bytes)

	subDir := strings.ToLower(string(eType)) + "s"
	fullDir := filepath.Join(config.ExpandPath(uiDataDir), subDir)
	err = os.MkdirAll(fullDir, 0700)
	if err != nil {
		return
	}
	filename := ent.Title + ".pb"
	newPath := filepath.Join(fullDir, filename)

	_, err = os.Stat(newPath)
	if !os.IsNotExist(err) && uiCurrentPath != newPath {
		if uiCurrentPath == "" {
			uiPendingSaveData, uiPendingPath = enc, newPath
			uiCollisionModal.SetText(fmt.Sprintf("'%s' exists. Replace or Add Suffix?", filename))
			uiPages.SwitchToPage("collision")
		} else {
			uiErrorModal.SetText(fmt.Sprintf("Cannot rename to '%s': File exists.", ent.Title))
			uiPages.SwitchToPage("error")
		}
		return
	}
	commitSave(newPath, enc)
}

func commitSave(newPath string, enc []byte) {
	for id, localPath := range uiPendingFilePaths {
		data, err := os.ReadFile(localPath)
		if err == nil {
			encData, _ := crypto.Encrypt(uiMasterKey, data)
			err := os.WriteFile(filepath.Join(getAttachmentDir(), id), encData, 0600)
			if err != nil {
				return
			}
		}
	}
	if uiCurrentPath != "" && uiCurrentPath != newPath {
		err := os.Remove(uiCurrentPath)
		if err != nil {
			return
		}
	}
	err := os.WriteFile(newPath, enc, 0600)
	if err != nil {
		return
	}
	refreshTree(uiSearchField.GetText())
	selectTreePath(newPath)
	uiPages.SwitchToPage("main")
	loadEntry(newPath)
}
