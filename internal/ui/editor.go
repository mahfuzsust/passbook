package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiCurrentEntryID int64
	uiCurrentEnt     *Entry
	uiEditingEnt     *Entry

	uiPendingAttachments []Attachment
	uiPendingFilePaths   map[string]string
	uiLastGeneratedPass  string

	uiEditorForm          *tview.Form
	uiEditorLayout        *tview.Flex
	uiEditorTitleField    *tview.InputField
	uiEditorFolderField   *tview.DropDown
	uiEditorSaveButton    *tview.Button
	uiEditorPasswordField *tview.InputField
	uiEditorCardNumber    *tview.InputField
	uiEditorExpiry        *tview.InputField
	uiEditorCVV           *tview.InputField
	uiAttachFlex          *tview.Flex
	uiAttachList          *tview.List
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
	uiErrorModal = tview.NewModal().AddButtons([]string{"OK"}).
		SetDoneFunc(func(i int, l string) { uiPages.SwitchToPage("editor") })
	enableModalButtonNav(uiErrorModal)
	uiPages.AddPage("error", uiErrorModal, true, false)
}

func newEntry(t EntryType) {
	uiCurrentEntryID = 0
	ent := NewEntry(t)
	openEditor(ent)
}

func openEditor(ent *Entry) {
	uiEditingEnt = ent
	uiPendingAttachments = append([]Attachment{}, ent.Attachments...)
	uiPendingFilePaths = make(map[string]string)

	uiEditorForm.Clear(true)
	uiEditorTitleField, uiEditorPasswordField, uiEditorSaveButton = nil, nil, nil
	uiEditorCardNumber, uiEditorExpiry, uiEditorCVV = nil, nil, nil
	uiEditorFolderField = nil

	if uiCurrentEntryID == 0 {
		uiEditorLayout.SetTitle(" Add Entry ")
	} else {
		uiEditorLayout.SetTitle(" Edit Entry ")
	}

	titleField := tview.NewInputField().SetLabel("Title").SetText(ent.Title).SetFieldWidth(40)
	titleField.SetChangedFunc(func(text string) { updateEditorSaveState() })
	uiEditorTitleField = titleField
	uiEditorForm.AddFormItem(titleField)

	folders := listFolders()
	folderOptions := []string{"— (root)"}
	folderOptions = append(folderOptions, folders...)
	currentFolderIdx := 0
	if uiCurrentEntryID != 0 {
		meta, _ := uiStore.GetEntryMeta(uiCurrentEntryID)
		if meta != nil && meta.FolderID != 0 {
			folder, _ := uiStore.GetFolder(meta.FolderID)
			if folder != nil {
				for i, opt := range folderOptions {
					if opt == folder.Name {
						currentFolderIdx = i
						break
					}
				}
			}
		}
	} else if uiCurrentFolderID != 0 {
		folder, _ := uiStore.GetFolder(uiCurrentFolderID)
		if folder != nil {
			for i, opt := range folderOptions {
				if opt == folder.Name {
					currentFolderIdx = i
					break
				}
			}
		}
	}
	folderDrop := tview.NewDropDown().SetLabel("Folder").SetFieldWidth(30)
	folderDrop.SetOptions(folderOptions, nil)
	folderDrop.SetCurrentOption(currentFolderIdx)
	uiEditorFolderField = folderDrop
	uiEditorForm.AddFormItem(folderDrop)

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
	enableButtonNav(uiEditorForm)
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
	var priorHistory []PasswordHistory
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
		ent.CVV = cvv
	case TypeNote:
		collectNoteFields(ent)
	case TypeFile:
		collectFileFields(ent)
	}

	var folderID int64
	if uiEditorFolderField != nil {
		_, folderName := uiEditorFolderField.GetCurrentOption()
		if folderName != "— (root)" {
			f, _ := uiStore.GetFolderByName(folderName)
			if f != nil {
				folderID = f.ID
			}
		}
	}

	if uiCurrentEntryID != 0 {
		if uiStore.EntryExistsInFolderExcluding(folderID, title, uiCurrentEntryID) {
			uiErrorModal.SetText("Title already exists in this folder. Please change the title.")
			uiPages.SwitchToPage("error")
			return
		}
		commitSave(uiCurrentEntryID, folderID, ent)
	} else {
		if uiStore.EntryExistsInFolder(folderID, title) {
			uiErrorModal.SetText("Title already exists in this folder. Please change the title.")
			uiPages.SwitchToPage("error")
			return
		}
		commitSaveNew(folderID, ent)
	}
}

func commitSaveNew(folderID int64, ent *Entry) {
	entryID, err := uiStore.SaveEntry(folderID, ent)
	if err != nil {
		return
	}

	saveAttachments(entryID)

	refreshTree(uiSearchField.GetText())
	selectTreeNode(nodeRef{IsFolder: false, ID: entryID})
	uiPages.SwitchToPage("main")
	loadEntry(entryID)
}

func commitSave(entryID int64, folderID int64, ent *Entry) {
	saveAttachments(entryID)

	if err := uiStore.UpdateEntryFull(entryID, folderID, ent); err != nil {
		return
	}

	refreshTree(uiSearchField.GetText())
	selectTreeNode(nodeRef{IsFolder: false, ID: entryID})
	uiPages.SwitchToPage("main")
	loadEntry(entryID)
}

func saveAttachments(entryID int64) {
	for id, localPath := range uiPendingFilePaths {
		data, err := os.ReadFile(localPath)
		if err != nil {
			continue
		}
		var fileName string
		var size int64
		for _, att := range uiPendingAttachments {
			if att.ID == id {
				fileName = att.FileName
				size = att.Size
				break
			}
		}
		if fileName == "" {
			parts := strings.Split(localPath, "/")
			fileName = parts[len(parts)-1]
			size = int64(len(data))
		}
		_ = uiStore.WriteAttachment(id, entryID, fileName, size, data)
	}
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
