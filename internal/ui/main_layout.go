package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

	root := tview.NewTreeNode("Vault").SetSelectable(false)
	uiTreeView = tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	uiTreeView.SetBorder(true).SetTitle(" Vault (Ctrl+A Add) ")
	uiTreeView.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			node.SetExpanded(!node.IsExpanded())
		} else {
			loadEntry(ref.(string))
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

	emptyView := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter).
		SetText("\n\n\n[yellow]Select an item from the list to view details.[-]")

	uiRightPages = tview.NewPages()
	uiRightPages.SetBorder(true).SetTitle(" Contents (Ctrl+E Edit | Ctrl+D Delete) ")
	uiRightPages.AddPage("empty", emptyView, true, true)
	uiRightPages.AddPage("content", uiViewFlex, true, false)

	mainFlex := newResponsiveSplit(leftFlex, uiRightPages, 0.30, 24, 40)

	mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			showCreateMenu()
			return nil
		case tcell.KeyCtrlE:
			if uiCurrentEnt != nil && uiCurrentPath != "" {
				openEditor(uiCurrentEnt)
			}
			return nil
		case tcell.KeyCtrlD:
			if uiCurrentPath != "" {
				showDeleteModal()
			}
			return nil
		case tcell.KeyCtrlF:
			uiApp.SetFocus(uiSearchField)
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
