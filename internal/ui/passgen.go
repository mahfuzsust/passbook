package ui

import (
	"passbook/internal/utils"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiPassGenForm    *tview.Form
	uiPassGenLayout  *tview.Flex
	uiPassGenPreview *tview.TextView
)

// setupPassGen configures the password generator modal.
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

// updatePassPreview regenerates the password preview text.
func updatePassPreview() {
	lStr := uiPassGenForm.GetFormItemByLabel("Length").(*tview.InputField).GetText()
	l, _ := strconv.Atoi(lStr)
	upper := uiPassGenForm.GetFormItemByLabel("A-Z").(*tview.Checkbox).IsChecked()
	lower := uiPassGenForm.GetFormItemByLabel("a-z").(*tview.Checkbox).IsChecked()
	special := uiPassGenForm.GetFormItemByLabel("Special").(*tview.Checkbox).IsChecked()
	uiLastGeneratedPass = utils.GeneratePassword(l, upper, lower, special)
	uiPassGenPreview.SetText("[green]" + uiLastGeneratedPass)
}
