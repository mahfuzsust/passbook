package ui

import "github.com/rivo/tview"

func resetEditorTestState() {
	uiEditingEnt = nil
	uiCurrentEnt = nil
	uiEditorForm = tview.NewForm()
	uiEditorLayout = tview.NewFlex()
	uiAttachFlex = tview.NewFlex()
	uiAttachList = tview.NewList()

	uiEditorTitleField = nil
	uiEditorPasswordField = nil
	uiEditorCardNumber = nil
	uiEditorExpiry = nil
	uiEditorCVV = nil
	uiEditorSaveButton = nil

	uiPendingAttachments = nil
	uiPendingFilePaths = map[string]string{}
}
