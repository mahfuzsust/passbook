package ui

import (
	"fmt"
	"os"
	"passbook/internal/config"
	"passbook/internal/crypto"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/protobuf/proto"
)

func setupEditor() {
	setupCreateMenu()

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
		addLoginFields(ent)
	case TypeCard:
		addCardFields(ent)
	case TypeNote:
		addNoteFields(ent)
	case TypeFile:
		addFileFields(ent)
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
	uiEditorForm.SetFocusFunc(func() { highlightFocusedEditorItem() })
	uiEditorForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("main")
		}
		go func() {
			uiApp.QueueUpdateDraw(func() { highlightFocusedEditorItem() })
		}()
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

func updateEditorSaveState() {
	if uiEditorTitleField == nil || uiEditorSaveButton == nil {
		return
	}

	_, titleErr := validateTitleField()
	cardErr := validateCardFields()
	uiEditorSaveButton.SetDisabled(titleErr != nil || cardErr != nil)
}

func saveEntry(eType EntryType) {
	title, err := validateTitleField()
	if err != nil {
		updateEditorSaveState()
		return
	}

	if err := validateCardFields(); err != nil {
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
		collectLoginFields(ent, priorPassword)
	case TypeCard:
		number, expiry, cvv := collectCardFields()
		ent.CardNumber = number
		ent.Expiry = expiry
		ent.Cvv = cvv
	case TypeNote:
		collectNoteFields(ent)
	case TypeFile:
		collectFileFields(ent)
	}

	bytes, _ := proto.Marshal(ent)
	enc, _ := crypto.Encrypt(uiMasterKey, bytes)

	subDir := strings.ToLower(string(eType)) + "s"
	fullDir := filepath.Join(config.ExpandPath(uiDataDir), subDir)
	if err := os.MkdirAll(fullDir, 0700); err != nil {
		return
	}
	filename := ent.Title + ".pb"
	newPath := filepath.Join(fullDir, filename)

	if _, err := os.Stat(newPath); !os.IsNotExist(err) && uiCurrentPath != newPath {
		uiErrorModal.SetText("Title already exists. Please change the title.")
		uiPages.SwitchToPage("error")
		return
	}
	commitSave(newPath, enc)
}

func commitSave(newPath string, enc []byte) {
	for id, localPath := range uiPendingFilePaths {
		data, err := os.ReadFile(localPath)
		if err == nil {
			encData, _ := crypto.Encrypt(uiMasterKey, data)
			if err := os.WriteFile(filepath.Join(getAttachmentDir(), id), encData, 0600); err != nil {
				return
			}
		}
	}
	if uiCurrentPath != "" && uiCurrentPath != newPath {
		if err := os.Remove(uiCurrentPath); err != nil {
			return
		}
	}
	if err := os.WriteFile(newPath, enc, 0600); err != nil {
		return
	}
	refreshTree(uiSearchField.GetText())
	selectTreePath(newPath)
	uiPages.SwitchToPage("main")
	loadEntry(newPath)
}

func highlightFocusedEditorItem() {
	focused := uiApp.GetFocus()
	for i := 0; i < uiEditorForm.GetFormItemCount(); i++ {
		item := uiEditorForm.GetFormItem(i)
		if input, ok := item.(*tview.InputField); ok {
			input.SetLabelColor(tcell.ColorWhite)
			input.SetFieldBackgroundColor(colorUnfocusedBg)
		} else if ta, ok := item.(*tview.TextArea); ok {
			ta.SetLabelStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite))
		}
	}
	for i := 0; i < uiEditorForm.GetFormItemCount(); i++ {
		item := uiEditorForm.GetFormItem(i)
		if p, ok := item.(tview.Primitive); ok && p == focused {
			if input, ok := item.(*tview.InputField); ok {
				input.SetLabelColor(tcell.ColorYellow)
				input.SetFieldBackgroundColor(colorFocusedBg)
			} else if ta, ok := item.(*tview.TextArea); ok {
				ta.SetLabelStyle(tcell.StyleDefault.Foreground(tcell.ColorYellow))
			}
			return
		}
	}
}
