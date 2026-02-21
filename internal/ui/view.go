package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"
)

func updateViewPane() {
	uiViewFlex.Clear()
	uiAttachmentList.Clear()

	uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	uiViewTitle.SetText(uiCurrentEnt.Title)
	uiViewFlex.AddItem(makeRow("Title:", uiViewTitle), 1, 0, false)
	uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	switch EntryType(uiCurrentEnt.Type) {
	case TypeLogin:
		renderLoginView()
	case TypeCard:
		renderCardView()
	case TypeNote:
		renderNoteView()
	case TypeFile:
		renderFileView()
	}

	if len(uiCurrentEnt.Attachments) > 0 {
		uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
		uiViewFlex.AddItem(tview.NewTextView().SetText("[yellow]Attachments:[-]").SetDynamicColors(true), 1, 0, false)

		for _, att := range uiCurrentEnt.Attachments {
			a := att
			label := fmt.Sprintf("[blue::u]➤ %s[-:-:-] [dim](%s)[-]", a.FileName, formatBytes(a.Size))
			uiAttachmentList.AddItem(label, "", 0, func() { downloadAttachment(a) })
		}

		h := len(uiCurrentEnt.Attachments) * 2
		if h < 3 {
			h = 3
		}
		if h > 10 {
			h = 10
		}
		uiViewFlex.AddItem(uiAttachmentList, h, 0, false)
	}

	uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
	if strings.TrimSpace(uiCurrentEnt.CustomText) != "" {
		header := tview.NewTextView().SetText("[yellow]Notes:[-]").SetDynamicColors(true)
		btnNotesCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
			err := clipboard.WriteAll(uiCurrentEnt.CustomText)
			if err != nil {
				return
			}
			notifyCopied("Notes")
		}))
		uiViewFlex.AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(header, 0, 1, false).
			AddItem(tview.NewTextView().SetText(" "), 1, 0, false).
			AddItem(btnNotesCopy, 5, 0, false), 1, 0, false)
		uiViewCustom.SetText(uiCurrentEnt.CustomText)
		uiViewFlex.AddItem(uiViewCustom, 0, 1, false)
	} else {
		uiViewCustom.SetText("")
	}
	uiViewFlex.AddItem(uiViewStatus, 1, 0, false)
}

func deleteEntry() {
	if uiCurrentPath != "" {
		for _, att := range uiCurrentEnt.Attachments {
			err := os.Remove(filepath.Join(getAttachmentDir(), att.Id))
			if err != nil {
				return
			}
		}
		err := os.Remove(uiCurrentPath)
		if err != nil {
			return
		}
		uiCurrentPath = ""
		refreshTree(uiSearchField.GetText())
	}
}

func notifyCopied(item string) {
	uiViewStatus.SetText(fmt.Sprintf("[green]✓ %s copied![-]", item))
	go func() { time.Sleep(2 * time.Second); uiApp.QueueUpdateDraw(func() { uiViewStatus.SetText("") }) }()
}

func copySensitive(text, item string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		return
	}
	uiViewStatus.SetText(fmt.Sprintf("[green]✓ %s copied (clears in 30s)[-]", item))
	go func() {
		time.Sleep(30 * time.Second)
		curr, _ := clipboard.ReadAll()
		if curr == text {
			err := clipboard.WriteAll("")
			if err != nil {
				return
			}
			uiApp.QueueUpdateDraw(func() { uiViewStatus.SetText("[yellow]Clipboard cleared[-]") })
		}
	}()
}

func drawTOTP() {
	if uiViewTOTP == nil || uiViewTOTPBar == nil {
		return
	}
	if uiPages != nil {
		if name, _ := uiPages.GetFrontPage(); name == "login" {
			uiViewTOTP.SetText("")
			uiViewTOTPBar.SetText("")
			return
		}
	}

	if uiCurrentEnt != nil && EntryType(uiCurrentEnt.Type) == TypeLogin {
		secret := strings.ReplaceAll(uiCurrentEnt.TotpSecret, " ", "")
		if secret != "" {
			code, err := totp.GenerateCode(secret, time.Now())
			if err == nil {
				uiViewTOTP.SetText(code)
				sec := time.Now().Unix() % 30
				remain := 30 - sec
				bars := int((float64(remain) / 30.0) * 20.0)
				barStr := strings.Repeat("█", bars) + strings.Repeat("▒", 20-bars)
				color := "green"
				if remain <= 5 {
					color = "red"
				} else if remain <= 10 {
					color = "yellow"
				}
				uiViewTOTPBar.SetText(fmt.Sprintf("[%s]%02ds [%s][-]", color, remain, barStr))
				return
			}
		}
	}
	uiViewTOTP.SetText("")
	uiViewTOTPBar.SetText("")
}
