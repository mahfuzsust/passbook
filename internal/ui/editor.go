package ui

import (
	"fmt"
	"os"
	"passbook/internal/config"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"passbook/internal/crypto"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/protobuf/proto"
)

func setupEditor() {
	uiCreateList = tview.NewList().ShowSecondaryText(false)
	uiCreateList.AddItem("Login", "Password & 2FA", 'l', func() { newEntry(TypeLogin) })
	uiCreateList.AddItem("Card", "Credit/Debit Details", 'c', func() { newEntry(TypeCard) })
	uiCreateList.AddItem("Note", "Secure Text", 'n', func() { newEntry(TypeNote) })
	uiCreateList.AddItem("File", "Encrypted Attachments", 'f', func() { newEntry(TypeFile) })
	uiCreateList.SetBorder(true).SetTitle(" Create New ")
	uiPages.AddPage("create_menu", centeredModal(uiCreateList, 30, 14), true, false)

	uiEditorForm = tview.NewForm()
	uiAttachList = tview.NewList().ShowSecondaryText(false).SetMainTextColor(tcell.ColorGreen)

	uiAttachFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	uiAttachFlex.AddItem(tview.NewTextView().SetText(" Attachments:").SetTextColor(tcell.ColorYellow), 1, 0, false)
	uiAttachFlex.AddItem(uiAttachList, 0, 1, true)

	uiEditorLayout = tview.NewFlex().SetDirection(tview.FlexRow)
	uiEditorLayout.AddItem(uiEditorForm, 0, 1, true)
	uiEditorLayout.SetBorder(true).SetTitle(" Edit Entry ")
	uiPages.AddPage("editor", centeredModal(uiEditorLayout, 70, 30), true, false)

	setupFileBrowser()
	setupPassGen()
	setupCollisionModals()
}

func setupFileBrowser() {
	uiFileBrowser = tview.NewTreeView()
	uiFileBrowser.SetBorder(true).SetTitle(" Select File (Enter to Pick, Esc Cancel) ")
	uiFileBrowser.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("editor")
		}
		return event
	})

	uiFileBrowserModal = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(nil, 0, 1, false).AddItem(uiFileBrowser, 25, 1, true).AddItem(nil, 0, 1, false), 70, 1, true).
		AddItem(nil, 0, 1, false)
	uiPages.AddPage("filebrowser", uiFileBrowserModal, true, false)
}

func setupPassGen() {
	uiPassGenPreview = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	uiPassGenForm = tview.NewForm()
	uiPassGenForm.AddInputField("Length", "28", 10, tview.InputFieldInteger, nil)
	uiPassGenForm.AddCheckbox("A-Z", true, nil)
	uiPassGenForm.AddCheckbox("a-z", true, nil)
	uiPassGenForm.AddCheckbox("Special", true, nil)
	uiPassGenForm.AddButton("Refresh", func() { updatePassPreview() })
	uiPassGenForm.AddButton("Use", func() {
		if item := uiEditorForm.GetFormItemByLabel("Password"); item != nil {
			item.(*tview.InputField).SetText(uiLastGeneratedPass)
		}
		uiPages.SwitchToPage("editor")
	})
	uiPassGenForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("editor")
		}
		return event
	})
	uiPassGenForm.AddButton("Cancel", func() { uiPages.SwitchToPage("editor") })
	styleForm(uiPassGenForm)

	uiPassGenLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Generated:").SetTextColor(tcell.ColorYellow), 1, 0, false).
		AddItem(uiPassGenPreview, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(uiPassGenForm, 0, 1, true)
	uiPassGenLayout.SetBorder(true).SetTitle(" Generator ")
	uiPages.AddPage("passgen", centeredModal(uiPassGenLayout, 45, 20), true, false)
}

func setupCollisionModals() {
	uiCollisionModal = tview.NewModal().
		AddButtons([]string{"Replace", "Add Suffix", "Cancel"}).
		SetDoneFunc(func(index int, label string) {
			if label == "Cancel" {
				uiPages.SwitchToPage("editor")
			} else if label == "Replace" {
				commitSave(uiPendingPath, uiPendingSaveData)
			} else if label == "Add Suffix" {
				dir, base := filepath.Dir(uiPendingPath), strings.TrimSuffix(filepath.Base(uiPendingPath), ".pb")
				counter := 1
				var newPath string
				for {
					newPath = filepath.Join(dir, fmt.Sprintf("%s_%d.pb", base, counter))
					if _, err := os.Stat(newPath); os.IsNotExist(err) {
						break
					}
					counter++
				}
				commitSave(newPath, uiPendingSaveData)
			}
		})
	uiPages.AddPage("collision", uiCollisionModal, true, false)

	uiErrorModal = tview.NewModal().AddButtons([]string{"OK"}).
		SetDoneFunc(func(i int, l string) { uiPages.SwitchToPage("editor") })
	uiPages.AddPage("error", uiErrorModal, true, false)
}

func updatePassPreview() {
	lStr := uiPassGenForm.GetFormItemByLabel("Length").(*tview.InputField).GetText()
	l, _ := strconv.Atoi(lStr)
	upper := uiPassGenForm.GetFormItemByLabel("A-Z").(*tview.Checkbox).IsChecked()
	lower := uiPassGenForm.GetFormItemByLabel("a-z").(*tview.Checkbox).IsChecked()
	special := uiPassGenForm.GetFormItemByLabel("Special").(*tview.Checkbox).IsChecked()
	uiLastGeneratedPass = crypto.GeneratePassword(l, upper, lower, special)
	uiPassGenPreview.SetText("[green]" + uiLastGeneratedPass)
}

func showCreateMenu() {
	uiPages.SwitchToPage("create_menu")
}

func newEntry(t EntryType) {
	uiCurrentPath = ""
	ent := NewEntry(t)
	openEditor(ent)
}

func openEditor(ent *Entry) {
	uiEditingEnt = ent
	uiPendingAttachments = append([]*Attachment{}, ent.Attachments...)
	uiPendingFilePaths = make(map[string]string)

	uiEditorForm.Clear(true)
	uiEditorForm.AddInputField("Title", ent.Title, 40, nil, nil)

	uiEditorLayout.RemoveItem(uiAttachFlex)
	switch EntryType(ent.Type) {
	case TypeLogin:
		ent.Attachments = nil
		uiEditorForm.AddInputField("Username", ent.Username, 40, nil, nil)
		uiEditorForm.AddInputField("Password", ent.Password, 40, nil, nil)
		uiEditorForm.AddButton("Generate Password", func() { updatePassPreview(); uiPages.SwitchToPage("passgen") })
		uiEditorForm.AddInputField("Link", ent.Link, 40, nil, nil)
		uiEditorForm.AddInputField("TOTP Secret", ent.TotpSecret, 40, nil, nil)
	case TypeCard:
		ent.Attachments = nil
		uiEditorForm.AddInputField("Card Number", ent.CardNumber, 40, nil, nil)
		uiEditorForm.AddInputField("Expiry (MM/YY)", ent.Expiry, 10, nil, nil)
		uiEditorForm.AddInputField("CVV", ent.Cvv, 5, nil, nil)
	case TypeFile:
		uiEditorLayout.AddItem(uiAttachFlex, 0, 0, false)

		uiEditorForm.AddButton("Browse Filesystem", func() {
			home, _ := os.UserHomeDir()
			openFileBrowser(home)
		})

		dropZone := tview.NewTextArea().
			SetLabel("Drag File Here").
			SetPlaceholder("Click here, then drop/paste a file path, then press Enter to attach").
			SetSize(5, 40)

		dropZone.SetBorder(true)
		dropZone.SetTitle(" Dropzone ")
		dropZone.SetTitleColor(tcell.ColorYellow)
		dropZone.SetBackgroundColor(tcell.ColorBlack)

		resetDropZone := func() {
			dropZone.SetText("", true)
			dropZone.SetLabel("Drag File Here")
			dropZone.SetPlaceholder("Click here, then drop/paste a file path, then press Enter to attach")
			dropZone.SetBorder(true)
			dropZone.SetTitle(" Dropzone ")
			dropZone.SetTitleColor(tcell.ColorYellow)
			dropZone.SetBackgroundColor(tcell.ColorBlack)
		}

		attachFromDropZone := func() {
			rawPath := dropZone.GetText()
			cleanPath := strings.Trim(rawPath, "\"' \n\r\t")
			if cleanPath == "" {
				return
			}

			if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
				cleanPath = strings.ReplaceAll(cleanPath, "\\ ", " ")
			}

			if strings.Contains(cleanPath, "\n") || strings.Contains(cleanPath, "\r") {
				return
			}

			fi, err := os.Stat(cleanPath)
			if err != nil || fi.IsDir() {
				return
			}

			id := fmt.Sprintf("%d", time.Now().UnixNano())
			att := &Attachment{Id: id, FileName: filepath.Base(cleanPath), Size: fi.Size()}
			uiPendingAttachments = append(uiPendingAttachments, att)
			uiPendingFilePaths[id] = cleanPath

			resetDropZone()
			refreshAttachmentList(TypeFile)
			uiApp.SetFocus(uiAttachList)
		}

		dropZone.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEnter {
				attachFromDropZone()
				return nil
			}
			return event
		})

		uiEditorForm.AddFormItem(dropZone)

		resetDropZone()
		refreshAttachmentList(TypeFile)
	}

	uiEditorForm.AddTextArea("Notes", ent.CustomText, 50, 5, 0, nil)
	uiEditorForm.AddButton("Save", func() { saveEntry(EntryType(ent.Type)) })
	uiEditorForm.AddButton("Cancel", func() { uiPages.SwitchToPage("main"); uiApp.SetFocus(uiTreeView) })
	styleForm(uiEditorForm)
	uiEditorForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("main")
		}
		return event
	})
	uiPages.SwitchToPage("editor")
}

func refreshAttachmentList(t EntryType) {
	uiAttachList.Clear()
	size := 0

	if t == TypeFile || len(uiPendingAttachments) > 0 {
		if len(uiPendingAttachments) > 0 {
			size = 6
			for i, att := range uiPendingAttachments {
				label := att.FileName
				if _, isNew := uiPendingFilePaths[att.Id]; isNew {
					label += " [green](New)[-]"
				}
				idx := i
				uiAttachList.AddItem(label, "Press Enter to Remove", 0, func() {
					uiPendingAttachments = append(uiPendingAttachments[:idx], uiPendingAttachments[idx+1:]...)
					refreshAttachmentList(t)
				})
			}
		}
	}
	uiEditorLayout.ResizeItem(uiAttachFlex, size, 0)
}
func openFileBrowser(path string) {
	rootDir, _ := filepath.Abs(path)
	rootNode := tview.NewTreeNode(rootDir).SetColor(tcell.ColorYellow).SetReference(rootDir)
	uiFileBrowser.SetRoot(rootNode).SetCurrentNode(rootNode)
	addNodes(rootNode, rootDir)

	uiFileBrowser.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			return
		}
		path := ref.(string)
		fi, err := os.Stat(path)
		if err != nil {
			return
		}

		if fi.IsDir() {
			if len(node.GetChildren()) == 0 {
				addNodes(node, path)
			}
			node.SetExpanded(!node.IsExpanded())
		} else {
			id := fmt.Sprintf("%d", time.Now().UnixNano())
			att := &Attachment{Id: id, FileName: filepath.Base(path), Size: fi.Size()}
			uiPendingAttachments = append(uiPendingAttachments, att)
			uiPendingFilePaths[id] = path
			refreshAttachmentList(TypeFile)
			uiPages.SwitchToPage("editor")
		}
	})
	uiPages.SwitchToPage("filebrowser")
}

func addNodes(target *tview.TreeNode, path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		node := tview.NewTreeNode(f.Name()).SetReference(filepath.Join(path, f.Name()))
		if f.IsDir() {
			node.SetColor(tcell.ColorSkyblue)
		}
		target.AddChild(node)
	}
}

func saveEntry(eType EntryType) {
	title := uiEditorForm.GetFormItemByLabel("Title").(*tview.InputField).GetText()
	if title == "" {
		title = "Untitled"
	}

	var priorPassword string
	var priorHistory []*PasswordHistory
	if uiEditingEnt != nil {
		priorPassword = uiEditingEnt.Password
		priorHistory = uiEditingEnt.History
	}

	ent := &Entry{
		Type:        string(eType),
		Title:       title,
		CustomText:  uiEditorForm.GetFormItemByLabel("Notes").(*tview.TextArea).GetText(),
		History:     priorHistory,
		Attachments: uiPendingAttachments,
	}

	switch eType {
	case TypeLogin:
		ent.Username = uiEditorForm.GetFormItemByLabel("Username").(*tview.InputField).GetText()
		ent.Password = uiEditorForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()
		ent.Link = uiEditorForm.GetFormItemByLabel("Link").(*tview.InputField).GetText()
		ent.TotpSecret = uiEditorForm.GetFormItemByLabel("TOTP Secret").(*tview.InputField).GetText()
		if priorPassword != "" && priorPassword != ent.Password {
			ent.History = append(ent.History, &PasswordHistory{Password: priorPassword, Date: time.Now().Format("2006-01-02 15:04")})
		}
	case TypeCard:
		ent.CardNumber = uiEditorForm.GetFormItemByLabel("Card Number").(*tview.InputField).GetText()
		ent.Expiry = uiEditorForm.GetFormItemByLabel("Expiry (MM/YY)").(*tview.InputField).GetText()
		ent.Cvv = uiEditorForm.GetFormItemByLabel("CVV").(*tview.InputField).GetText()
	}

	bytes, _ := proto.Marshal(ent)
	enc, _ := crypto.Encrypt(uiMasterKey, bytes)

	subDir := strings.ToLower(string(eType)) + "s"
	fullDir := filepath.Join(config.ExpandPath(uiDataDir), subDir)
	err := os.MkdirAll(fullDir, 0700)
	if err != nil {
		return
	}
	filename := ent.Title + ".pb"
	newPath := filepath.Join(fullDir, filename)

	_, err = os.Stat(newPath)
	if !os.IsNotExist(err) && uiCurrentPath != newPath {
		if uiCurrentPath == "" {
			uiPendingSaveData, uiPendingPath = enc, newPath
			uiCollisionModal.SetText(fmt.Sprintf("'%s' exists. Replace or Add Suffix?", filename))
			uiPages.SwitchToPage("collision")
		} else {
			uiErrorModal.SetText(fmt.Sprintf("Cannot rename to '%s': File exists.", ent.Title))
			uiPages.SwitchToPage("error")
		}
		return
	}
	commitSave(newPath, enc)
}

func commitSave(newPath string, enc []byte) {
	for id, localPath := range uiPendingFilePaths {
		data, err := os.ReadFile(localPath)
		if err == nil {
			encData, _ := crypto.Encrypt(uiMasterKey, data)
			err := os.WriteFile(filepath.Join(getAttachmentDir(), id), encData, 0600)
			if err != nil {
				return
			}
		}
	}
	if uiCurrentPath != "" && uiCurrentPath != newPath {
		err := os.Remove(uiCurrentPath)
		if err != nil {
			return
		}
	}
	err := os.WriteFile(newPath, enc, 0600)
	if err != nil {
		return
	}
	refreshTree(uiSearchField.GetText())
	selectTreePath(newPath)
	uiPages.SwitchToPage("main")
	loadEntry(newPath)
}
