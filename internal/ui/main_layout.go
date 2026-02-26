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
	uiSearchField *tview.InputField
	uiTreeView    *tview.TreeView
	uiRightPages  *tview.Pages

	uiCurrentFolder      string
	uiFolderForm         *tview.Form
	uiFolderRenameForm   *tview.Form
	uiFolderDeleteModal  *tview.Modal
)

func setupMainLayout() {
	uiSearchField = styleInput(tview.NewInputField().SetLabel("Search: ")).SetPlaceholder("Ctrl+F")
	uiSearchField.SetChangedFunc(func(text string) { refreshTree(text) })
	uiSearchField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			uiApp.SetFocus(uiTreeView)
			return nil
		}
		return event
	})

	root := tview.NewTreeNode("").SetSelectable(false).SetExpanded(true)
	uiTreeView = tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	uiTreeView.SetTopLevel(1)
	uiTreeView.SetBorder(true).SetTitle(" Vault ")
	uiTreeView.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			node.SetExpanded(!node.IsExpanded())
			return
		}
		path := ref.(string)
		if strings.HasSuffix(path, ".pb") {
			uiCurrentFolder = ""
			loadEntry(path)
		} else {
			node.SetExpanded(!node.IsExpanded())
			uiCurrentFolder = path
			uiCurrentPath = ""
			uiCurrentEnt = nil
			uiRightPages.SwitchToPage("empty")
		}
	})

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(uiSearchField, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(uiTreeView, 0, 1, true)

	uiViewFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	uiViewTitle = tview.NewTextView().SetDynamicColors(true)
	uiViewSubtitle = tview.NewTextView().SetDynamicColors(true)
	uiViewPassword = tview.NewTextView().SetDynamicColors(true)
	uiViewDetails = tview.NewTextView().SetDynamicColors(true)
	uiViewTOTP = tview.NewTextView().SetDynamicColors(true)
	uiViewTOTPBar = tview.NewTextView().SetDynamicColors(true)
	uiViewCustom = tview.NewTextView().SetDynamicColors(true)
	uiViewStatus = tview.NewTextView().SetDynamicColors(true)
	uiAttachmentList = tview.NewList().ShowSecondaryText(false).SetMainTextColor(tcell.ColorSkyblue)

	keybindTable := tview.NewTable().SetBorders(false).SetSelectable(false, false)
	bindings := [][2]string{
		{"Ctrl+A", "Create new item"},
		{"Ctrl+E", "Edit item / rename folder"},
		{"Ctrl+D", "Delete item / folder"},
		{"Ctrl+N", "Create new folder"},
		{"Ctrl+F", "Search vault"},
		{"Ctrl+P", "Change master password"},
		{"Ctrl+Q", "Quit"},
		{"Enter", "Open item / toggle folder"},
		{"Esc", "Focus tree view"},
	}
	keybindTable.SetCell(0, 0, tview.NewTableCell("[yellow::b]Key[-::-]").SetExpansion(1).SetAlign(tview.AlignRight))
	keybindTable.SetCell(0, 1, tview.NewTableCell("  "))
	keybindTable.SetCell(0, 2, tview.NewTableCell("[yellow::b]Action[-::-]").SetExpansion(2))
	for i, b := range bindings {
		row := i + 1
		keybindTable.SetCell(row, 0, tview.NewTableCell("[skyblue]"+b[0]+"[-]").SetAlign(tview.AlignRight).SetExpansion(1))
		keybindTable.SetCell(row, 1, tview.NewTableCell("  "))
		keybindTable.SetCell(row, 2, tview.NewTableCell("[white]"+b[1]+"[-]").SetExpansion(2))
	}

	emptyView := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(keybindTable, len(bindings)+2, 0, false).
		AddItem(nil, 0, 1, false)

	uiRightPages = tview.NewPages()
	uiRightPages.SetBorder(true).SetTitle(" Keybindings ")
	uiRightPages.AddPage("empty", emptyView, true, true)
	uiRightPages.AddPage("content", uiViewFlex, true, false)

	mainFlex := newResponsiveSplit(leftFlex, uiRightPages, 0.30, 24, 40)

	mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			showCreateMenu()
			return nil
		case tcell.KeyCtrlE:
			if uiCurrentFolder != "" {
				showFolderRename()
			} else if uiCurrentEnt != nil && uiCurrentPath != "" {
				openEditor(uiCurrentEnt)
			}
			return nil
		case tcell.KeyCtrlD:
			if uiCurrentFolder != "" {
				showFolderDeleteModal()
			} else if uiCurrentPath != "" {
				showDeleteModal()
			}
			return nil
		case tcell.KeyCtrlN:
			showFolderCreate()
			return nil
		case tcell.KeyCtrlF:
			uiApp.SetFocus(uiSearchField)
			return nil
		case tcell.KeyCtrlP:
			showChangePassword()
			return nil
		case tcell.KeyCtrlQ:
			uiApp.Stop()
			return nil
		case tcell.KeyEsc:
			uiApp.SetFocus(uiTreeView)
			return nil
		default:
			return event
		}
	})

	uiPages.AddPage("main", mainFlex, true, false)
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

func isValidFolderName(name string) bool {
	return name != "" &&
		!strings.ContainsAny(name, `<>:"/\|?*`) &&
		name != "." && name != ".." &&
		!strings.HasPrefix(name, ".") &&
		!strings.HasPrefix(name, "_")
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
