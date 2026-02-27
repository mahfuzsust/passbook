package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiSearchField *tview.InputField
	uiTreeView    *tview.TreeView
	uiRightPages  *tview.Pages
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
	uiTreeView.SetChangedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			uiCurrentFolder = ""
			uiCurrentPath = ""
			uiCurrentEnt = nil
			uiRightPages.SetTitle(" Keybindings ")
			uiRightPages.SwitchToPage("empty")
			return
		}
		path := ref.(string)
		if strings.HasSuffix(path, ".pb") {
			uiCurrentFolder = ""
			loadEntry(path)
		} else {
			uiCurrentFolder = path
			uiCurrentPath = ""
			uiCurrentEnt = nil
			uiRightPages.SetTitle(" Keybindings ")
			uiRightPages.SwitchToPage("empty")
		}
	})
	uiTreeView.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			node.SetExpanded(!node.IsExpanded())
			return
		}
		if !strings.HasSuffix(ref.(string), ".pb") {
			node.SetExpanded(!node.IsExpanded())
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
		{"Ctrl+Y", "Quick copy to clipboard"},
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
		case tcell.KeyCtrlY:
			showQuickCopy()
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
