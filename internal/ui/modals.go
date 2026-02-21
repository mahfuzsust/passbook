package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func setupModals() {
	uiDeleteModal = tview.NewModal().
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(index int, label string) {
			if label == "Delete" {
				deleteEntry()
			}
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiTreeView)
		})
	uiPages.AddPage("delete", uiDeleteModal, true, false)

	uiHistoryList = tview.NewList().ShowSecondaryText(true)
	uiHistoryList.SetBorder(true).SetTitle(" History (Esc to close) ")
	uiHistoryList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("main")
			uiApp.SetFocus(uiRightPages)
		}
		return event
	})
	uiPages.AddPage("history", newResponsiveModal(uiHistoryList, 50, 15, 80, 25, 0.6, 0.65), true, false)
}

func showHistory() {
	if uiHistoryList == nil || uiPages == nil {
		return
	}
	uiHistoryList.Clear()
	for i := len(uiCurrentEnt.History) - 1; i >= 0; i-- {
		uiHistoryList.AddItem(uiCurrentEnt.History[i].Password, uiCurrentEnt.History[i].Date, 0, nil)
	}
	uiPages.SwitchToPage("history")
}

func showDeleteModal() {
	uiDeleteModal.SetText("Delete " + uiCurrentEnt.Title + "?")
	uiPages.SwitchToPage("delete")
}
