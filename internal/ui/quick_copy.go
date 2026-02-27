package ui

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"
)

var uiQuickCopyList *tview.List

func setupQuickCopy() {
	uiQuickCopyList = tview.NewList().ShowSecondaryText(false)
	uiQuickCopyList.SetBorder(true).SetTitle(" Quick Copy ")
	uiQuickCopyList.SetHighlightFullLine(true)
	uiQuickCopyList.SetMainTextColor(tcell.ColorWhite)
	uiQuickCopyList.SetSelectedTextColor(tcell.ColorBlack)
	uiQuickCopyList.SetSelectedBackgroundColor(tcell.ColorSkyblue)
	uiQuickCopyList.SetShortcutColor(tcell.ColorYellow)
	uiQuickCopyList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			dismissQuickCopy()
			return nil
		}
		return event
	})
	uiPages.AddPage("quick_copy", newResponsiveModal(uiQuickCopyList, 35, 8, 50, 14, 0.35, 0.35), true, false)
}

func showQuickCopy() {
	if uiCurrentEnt == nil || uiCurrentPath == "" {
		return
	}

	uiQuickCopyList.Clear()

	switch EntryType(uiCurrentEnt.Type) {
	case TypeLogin:
		if uiCurrentEnt.Username != "" {
			uiQuickCopyList.AddItem("Username", "", 'u', func() {
				_ = clipboard.WriteAll(uiCurrentEnt.Username)
				dismissQuickCopy()
				notifyCopied("Username")
			})
		}
		if uiCurrentEnt.Password != "" {
			uiQuickCopyList.AddItem("Password", "", 'p', func() {
				dismissQuickCopy()
				copySensitive(uiCurrentEnt.Password, "Password")
			})
		}
		secret := strings.ReplaceAll(uiCurrentEnt.TotpSecret, " ", "")
		if secret != "" {
			uiQuickCopyList.AddItem("TOTP Code", "", 't', func() {
				code, err := totp.GenerateCode(secret, time.Now())
				if err == nil {
					dismissQuickCopy()
					copySensitive(code, "TOTP")
				}
			})
		}

	case TypeCard:
		if uiCurrentEnt.CardNumber != "" {
			uiQuickCopyList.AddItem("Card Number", "", 'c', func() {
				dismissQuickCopy()
				copySensitive(uiCurrentEnt.CardNumber, "Card Number")
			})
		}
		if uiCurrentEnt.Cvv != "" {
			uiQuickCopyList.AddItem("CVV", "", 'v', func() {
				dismissQuickCopy()
				copySensitive(uiCurrentEnt.Cvv, "CVV")
			})
		}

	case TypeNote:
		if strings.TrimSpace(uiCurrentEnt.CustomText) != "" {
			uiQuickCopyList.AddItem("Note", "", 'n', func() {
				_ = clipboard.WriteAll(uiCurrentEnt.CustomText)
				dismissQuickCopy()
				notifyCopied("Note")
			})
		}
	}

	if uiQuickCopyList.GetItemCount() == 0 {
		return
	}

	uiQuickCopyList.SetCurrentItem(0)
	uiPages.SwitchToPage("quick_copy")
	uiApp.SetFocus(uiQuickCopyList)
}

func dismissQuickCopy() {
	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}
