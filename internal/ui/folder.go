package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"passbook/internal/config"
	"passbook/internal/crypto"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiCurrentFolder     string
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

func setupFolderCreate() {
	uiFolderForm = tview.NewForm()
	uiFolderForm.AddInputField("Folder Name", "", 0, nil, nil)
	uiFolderForm.AddButton("Create", func() {
		nameField := uiFolderForm.GetFormItemByLabel("Folder Name").(*tview.InputField)
		name := strings.TrimSpace(nameField.GetText())
		if !isValidFolderName(name) {
			return
		}
		basePath := config.ExpandPath(uiDataDir)
		folderPath := filepath.Join(basePath, name)
		if err := os.MkdirAll(folderPath, 0700); err != nil {
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
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
			return nil
		}
		return event
	})
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
	uiFolderRenameForm.AddInputField("Folder Name", "", 0, nil, nil)
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
	uiPages.AddPage("folder_rename", newResponsiveModal(uiFolderRenameForm, 45, 9, 65, 13, 0.45, 0.3), true, false)
}

func showFolderRename() {
	if uiFolderRenameForm == nil || uiCurrentFolder == "" {
		return
	}
	nameField := uiFolderRenameForm.GetFormItemByLabel("Folder Name").(*tview.InputField)
	nameField.SetText(filepath.Base(uiCurrentFolder))
	uiPages.SwitchToPage("folder_rename")
}

func doFolderRename() {
	nameField := uiFolderRenameForm.GetFormItemByLabel("Folder Name").(*tview.InputField)
	name := strings.TrimSpace(nameField.GetText())
	if !isValidFolderName(name) {
		return
	}
	basePath := config.ExpandPath(uiDataDir)
	newPath := filepath.Join(basePath, name)
	if newPath == uiCurrentFolder {
		uiPages.SwitchToPage("main")
		uiApp.SetFocus(uiTreeView)
		return
	}
	if err := os.Rename(uiCurrentFolder, newPath); err != nil {
		return
	}
	uiCurrentFolder = newPath
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
	uiPages.AddPage("folder_delete", uiFolderDeleteModal, true, false)
}

func showFolderDeleteModal() {
	if uiCurrentFolder == "" {
		return
	}
	folderName := filepath.Base(uiCurrentFolder)
	files, _ := os.ReadDir(uiCurrentFolder)
	count := 0
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".pb") {
			count++
		}
	}

	if count > 0 {
		uiFolderDeleteModal.SetText(fmt.Sprintf(
			"Folder \"%s\" contains %d item(s).\nAll items inside will be permanently deleted.\n\nAre you sure?",
			folderName, count))
	} else {
		uiFolderDeleteModal.SetText(fmt.Sprintf("Delete empty folder \"%s\"?", folderName))
	}
	uiPages.SwitchToPage("folder_delete")
}

func doFolderDelete() {
	if uiCurrentFolder == "" {
		return
	}
	files, _ := os.ReadDir(uiCurrentFolder)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".pb") {
			continue
		}
		path := filepath.Join(uiCurrentFolder, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		dec, err := crypto.Decrypt(uiMasterKey, data)
		if err != nil {
			continue
		}
		ent, err := unmarshalEntry(dec)
		if err != nil {
			continue
		}
		for _, att := range ent.Attachments {
			_ = os.Remove(filepath.Join(getAttachmentDir(), att.Id))
		}
	}
	_ = os.RemoveAll(uiCurrentFolder)
	uiCurrentFolder = ""
	uiCurrentPath = ""
	uiCurrentEnt = nil
	refreshTree(uiSearchField.GetText())
}
