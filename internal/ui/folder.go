package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiCurrentFolderID   int64
	uiFolderForm        *tview.Form
	uiFolderRenameForm  *tview.Form
	uiFolderDeleteModal *tview.Modal
)

func isValidFolderName(name string) bool {
	return name != "" &&
		!strings.ContainsAny(name, `<>:"/\|?*`) &&
		name != "." && name != ".." &&
		!strings.HasPrefix(name, ".") &&
		!strings.HasPrefix(name, "_")
}

func folderNameAcceptFunc(text string, ch rune) bool {
	return !strings.ContainsRune(`<>:"/\|?*`, ch) && ch != '/'
}

func setupFolderCreate() {
	uiFolderForm = tview.NewForm()
	uiFolderForm.AddInputField("Folder Name", "", 0, folderNameAcceptFunc, nil)
	uiFolderForm.AddButton("Create", func() {
		nameField := uiFolderForm.GetFormItemByLabel("Folder Name").(*tview.InputField)
		name := strings.TrimSpace(nameField.GetText())
		if !isValidFolderName(name) {
			return
		}
		if _, err := uiStore.CreateFolder(name); err != nil {
			return
		}
		refreshTree(uiSearchField.GetText())
		uiPages.SwitchToPage("main")
		uiApp.SetFocus(uiTreeView)
	})
	uiFolderForm.AddButton("Cancel", func() {
		uiPages.SwitchToPage("main")
		uiApp.SetFocus(uiTreeView)
	})
	uiFolderForm.SetBorder(true).SetTitle(" New Folder ")
	styleForm(uiFolderForm)
	uiFolderForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
			return nil
		case tcell.KeyEnter:
			if uiApp.GetFocus() != uiFolderForm.GetButton(0) &&
				uiApp.GetFocus() != uiFolderForm.GetButton(1) {
				uiFolderForm.GetButton(0).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), nil)
				return nil
			}
		}
		return event
	})
	enableButtonNav(uiFolderForm)
	uiPages.AddPage("folder_create", newResponsiveModal(uiFolderForm, 45, 9, 65, 13, 0.45, 0.3), true, false)
}

func showFolderCreate() {
	if uiFolderForm != nil {
		nameField := uiFolderForm.GetFormItemByLabel("Folder Name").(*tview.InputField)
		nameField.SetText("")
	}
	uiPages.SwitchToPage("folder_create")
}

func setupFolderRename() {
	uiFolderRenameForm = tview.NewForm()
	uiFolderRenameForm.AddInputField("Folder Name", "", 0, folderNameAcceptFunc, nil)
	uiFolderRenameForm.AddButton("Rename", doFolderRename)
	uiFolderRenameForm.AddButton("Cancel", func() {
		uiPages.SwitchToPage("main")
		uiApp.SetFocus(uiTreeView)
	})
	uiFolderRenameForm.SetBorder(true).SetTitle(" Rename Folder ")
	styleForm(uiFolderRenameForm)
	uiFolderRenameForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
			return nil
		}
		return event
	})
	enableButtonNav(uiFolderRenameForm)
	uiPages.AddPage("folder_rename", newResponsiveModal(uiFolderRenameForm, 45, 9, 65, 13, 0.45, 0.3), true, false)
}

func showFolderRename() {
	if uiFolderRenameForm == nil || uiCurrentFolderID == 0 {
		return
	}
	folder, _ := uiStore.GetFolder(uiCurrentFolderID)
	if folder == nil {
		return
	}
	nameField := uiFolderRenameForm.GetFormItemByLabel("Folder Name").(*tview.InputField)
	nameField.SetText(folder.Name)
	uiPages.SwitchToPage("folder_rename")
}

func doFolderRename() {
	nameField := uiFolderRenameForm.GetFormItemByLabel("Folder Name").(*tview.InputField)
	name := strings.TrimSpace(nameField.GetText())
	if !isValidFolderName(name) {
		return
	}
	if err := uiStore.RenameFolder(uiCurrentFolderID, name); err != nil {
		return
	}
	refreshTree(uiSearchField.GetText())
	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func setupFolderDelete() {
	uiFolderDeleteModal = tview.NewModal().
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(index int, label string) {
			if label == "Delete" {
				doFolderDelete()
			}
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
		})
	enableModalButtonNav(uiFolderDeleteModal)
	uiPages.AddPage("folder_delete", uiFolderDeleteModal, true, false)
}

func showFolderDeleteModal() {
	if uiCurrentFolderID == 0 {
		return
	}
	folder, _ := uiStore.GetFolder(uiCurrentFolderID)
	if folder == nil {
		return
	}
	count := uiStore.CountEntriesInFolder(uiCurrentFolderID)

	if count > 0 {
		uiFolderDeleteModal.SetText(fmt.Sprintf(
			"Folder \"%s\" contains %d item(s).\nAll items inside will be permanently deleted.\n\nAre you sure?",
			folder.Name, count))
	} else {
		uiFolderDeleteModal.SetText(fmt.Sprintf("Delete empty folder \"%s\"?", folder.Name))
	}
	uiPages.SwitchToPage("folder_delete")
}

func doFolderDelete() {
	if uiCurrentFolderID == 0 {
		return
	}
	_ = uiStore.DeleteFolder(uiCurrentFolderID)
	uiCurrentFolderID = 0
	uiCurrentEntryID = 0
	uiCurrentEnt = nil
	refreshTree(uiSearchField.GetText())
}
