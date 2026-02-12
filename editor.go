package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func setupEditor() {
	// 1. Create Menu (Vertical List)
	createList = tview.NewList().ShowSecondaryText(false)
	createList.AddItem("Login", "Password & 2FA", 'l', func() { newEntry(TypeLogin) })
	createList.AddItem("Card", "Credit/Debit Details", 'c', func() { newEntry(TypeCard) })
	createList.AddItem("Note", "Secure Text", 'n', func() { newEntry(TypeNote) })
	createList.AddItem("File", "Encrypted Attachments", 'f', func() { newEntry(TypeFile) })
	createList.SetBorder(true).SetTitle(" Create New ")
	pages.AddPage("create_menu", centeredModal(createList, 30, 14), true, false)

	// 2. Editor Layout
	editorForm = tview.NewForm()
	attachList = tview.NewList().ShowSecondaryText(false).SetMainTextColor(tcell.ColorGreen)

	attachFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	attachFlex.AddItem(tview.NewTextView().SetText(" Attachments:").SetTextColor(tcell.ColorYellow), 1, 0, false)
	attachFlex.AddItem(attachList, 0, 1, true)

	editorLayout = tview.NewFlex().SetDirection(tview.FlexRow)
	editorLayout.AddItem(editorForm, 0, 1, true)
	editorLayout.AddItem(attachFlex, 0, 0, false) // Hidden by default
	editorLayout.SetBorder(true).SetTitle(" Edit Entry ")
	pages.AddPage("editor", centeredModal(editorLayout, 70, 30), true, false)

	setupFileBrowser()
	setupPassGen()
	setupCollisionModals()
}

func setupFileBrowser() {
	fileBrowser = tview.NewTreeView()
	fileBrowser.SetBorder(true).SetTitle(" Select File (Enter to Pick, Esc Cancel) ")
	fileBrowser.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.SwitchToPage("editor")
		}
		return event
	})

	fileBrowserModal = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(nil, 0, 1, false).AddItem(fileBrowser, 25, 1, true).AddItem(nil, 0, 1, false), 70, 1, true).
		AddItem(nil, 0, 1, false)
	pages.AddPage("filebrowser", fileBrowserModal, true, false)
}

func setupPassGen() {
	passGenPreview = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	passGenForm = tview.NewForm()
	passGenForm.AddInputField("Length", "16", 10, tview.InputFieldInteger, nil)
	passGenForm.AddCheckbox("A-Z", true, nil)
	passGenForm.AddCheckbox("a-z", true, nil)
	passGenForm.AddCheckbox("Special", true, nil)
	passGenForm.AddButton("Refresh", func() { updatePassPreview() })
	passGenForm.AddButton("Use", func() {
		if item := editorForm.GetFormItemByLabel("Password"); item != nil {
			item.(*tview.InputField).SetText(lastGeneratedPass)
		}
		pages.SwitchToPage("editor")
	})
	passGenForm.AddButton("Cancel", func() { pages.SwitchToPage("editor") })
	styleForm(passGenForm)

	passGenLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Generated:").SetTextColor(tcell.ColorYellow), 1, 0, false).
		AddItem(passGenPreview, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(passGenForm, 0, 1, true)
	passGenLayout.SetBorder(true).SetTitle(" Generator ")
	pages.AddPage("passgen", centeredModal(passGenLayout, 45, 20), true, false)
}

func setupCollisionModals() {
	collisionModal = tview.NewModal().
		AddButtons([]string{"Replace", "Add Suffix", "Cancel"}).
		SetDoneFunc(func(index int, label string) {
			if label == "Cancel" {
				pages.SwitchToPage("editor")
			} else if label == "Replace" {
				commitSave(pendingPath, pendingSaveData)
			} else if label == "Add Suffix" {
				dir, base := filepath.Dir(pendingPath), strings.TrimSuffix(filepath.Base(pendingPath), ".md")
				counter := 1
				var newPath string
				for {
					newPath = filepath.Join(dir, fmt.Sprintf("%s_%d.md", base, counter))
					if _, err := os.Stat(newPath); os.IsNotExist(err) {
						break
					}
					counter++
				}
				commitSave(newPath, pendingSaveData)
			}
		})
	pages.AddPage("collision", collisionModal, true, false)

	errorModal = tview.NewModal().AddButtons([]string{"OK"}).
		SetDoneFunc(func(i int, l string) { pages.SwitchToPage("editor") })
	pages.AddPage("error", errorModal, true, false)
}

func updatePassPreview() {
	lStr := passGenForm.GetFormItemByLabel("Length").(*tview.InputField).GetText()
	l, _ := strconv.Atoi(lStr)
	upper := passGenForm.GetFormItemByLabel("A-Z").(*tview.Checkbox).IsChecked()
	lower := passGenForm.GetFormItemByLabel("a-z").(*tview.Checkbox).IsChecked()
	special := passGenForm.GetFormItemByLabel("Special").(*tview.Checkbox).IsChecked()
	lastGeneratedPass = generatePassword(l, upper, lower, special)
	passGenPreview.SetText("[green]" + lastGeneratedPass)
}

func showCreateMenu() {
	pages.SwitchToPage("create_menu")
}

func newEntry(t EntryType) {
	currentPath = ""
	openEditor(Entry{Type: t})
}

func openEditor(ent Entry) {
	editingEnt = ent
	pendingAttachments = append([]Attachment{}, ent.Attachments...)
	pendingFilePaths = make(map[string]string)

	editorForm.Clear(true)
	editorForm.AddInputField("Title", ent.Title, 40, nil, nil)

	switch ent.Type {
	case TypeLogin:
		editorForm.AddInputField("Username", ent.Username, 40, nil, nil)
		editorForm.AddInputField("Password", ent.Password, 40, nil, nil)
		editorForm.AddButton("Generate Password", func() { updatePassPreview(); pages.SwitchToPage("passgen") })
		editorForm.AddInputField("Link", ent.Link, 40, nil, nil)
		editorForm.AddInputField("TOTP Secret", ent.TOTPSecret, 40, nil, nil)
	case TypeCard:
		editorForm.AddInputField("Card Number", ent.CardNumber, 40, nil, nil)
		editorForm.AddInputField("Expiry (MM/YY)", ent.Expiry, 10, nil, nil)
		editorForm.AddInputField("CVV", ent.CVV, 5, nil, nil)
	case TypeFile:
		editorForm.AddButton("Add Attachment", func() {
			home, _ := os.UserHomeDir()
			openFileBrowser(home)
		})
	}

	editorForm.AddTextArea("Notes", ent.CustomText, 50, 5, 0, nil)
	editorForm.AddButton("Save", func() { saveEntry(ent.Type) })
	editorForm.AddButton("Cancel", func() { pages.SwitchToPage("main"); app.SetFocus(treeView) })
	styleForm(editorForm)

	refreshAttachmentList(ent.Type) // Pass type to handle visibility
	pages.SwitchToPage("editor")
}

func refreshAttachmentList(t EntryType) {
	attachList.Clear()
	size := 0

	if t == TypeFile || len(pendingAttachments) > 0 {
		if len(pendingAttachments) > 0 {
			size = 6
			for i, att := range pendingAttachments {
				label := att.FileName
				if _, isNew := pendingFilePaths[att.ID]; isNew {
					label += " [green](New)[-]"
				}
				idx := i
				attachList.AddItem(label, "Press Enter to Remove", 0, func() {
					pendingAttachments = append(pendingAttachments[:idx], pendingAttachments[idx+1:]...)
					refreshAttachmentList(t)
				})
			}
		}
	}
	editorLayout.ResizeItem(attachFlex, size, 0)
}

func getCurrentEditorState() Entry {
	e := editingEnt
	e.Title = editorForm.GetFormItemByLabel("Title").(*tview.InputField).GetText()
	e.CustomText = editorForm.GetFormItemByLabel("Notes").(*tview.TextArea).GetText()
	e.Attachments = pendingAttachments
	return e
}

func openFileBrowser(path string) {
	rootDir, _ := filepath.Abs(path)
	rootNode := tview.NewTreeNode(rootDir).SetColor(tcell.ColorYellow).SetReference(rootDir)
	fileBrowser.SetRoot(rootNode).SetCurrentNode(rootNode)
	addNodes(rootNode, rootDir)

	fileBrowser.SetSelectedFunc(func(node *tview.TreeNode) {
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
			att := Attachment{ID: id, FileName: filepath.Base(path), Size: fi.Size()}
			pendingAttachments = append(pendingAttachments, att)
			pendingFilePaths[id] = path
			refreshAttachmentList(TypeFile)
			pages.SwitchToPage("editor")
		}
	})
	pages.SwitchToPage("filebrowser")
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
	title := editorForm.GetFormItemByLabel("Title").(*tview.InputField).GetText()
	if title == "" {
		title = "Untitled"
	}

	ent := Entry{Type: eType, Title: title, CustomText: editorForm.GetFormItemByLabel("Notes").(*tview.TextArea).GetText(), History: editingEnt.History, Attachments: pendingAttachments}

	switch eType {
	case TypeLogin:
		ent.Username = editorForm.GetFormItemByLabel("Username").(*tview.InputField).GetText()
		ent.Password = editorForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()
		ent.Link = editorForm.GetFormItemByLabel("Link").(*tview.InputField).GetText()
		ent.TOTPSecret = editorForm.GetFormItemByLabel("TOTP Secret").(*tview.InputField).GetText()
		if editingEnt.Password != "" && editingEnt.Password != ent.Password {
			ent.History = append(ent.History, PasswordHistory{Password: editingEnt.Password, Date: time.Now().Format("2006-01-02 15:04")})
		}
	case TypeCard:
		ent.CardNumber = editorForm.GetFormItemByLabel("Card Number").(*tview.InputField).GetText()
		ent.Expiry = editorForm.GetFormItemByLabel("Expiry (MM/YY)").(*tview.InputField).GetText()
		ent.CVV = editorForm.GetFormItemByLabel("CVV").(*tview.InputField).GetText()
	}

	bytes, _ := json.Marshal(ent)
	enc, _ := encrypt(bytes)

	subDir := strings.ToLower(string(eType)) + "s"
	fullDir := filepath.Join(expandPath(dataDir), subDir)
	os.MkdirAll(fullDir, 0700)
	filename := ent.Title + ".md"
	newPath := filepath.Join(fullDir, filename)

	_, err := os.Stat(newPath)
	if !os.IsNotExist(err) && currentPath != newPath {
		if currentPath == "" {
			pendingSaveData, pendingPath = enc, newPath
			collisionModal.SetText(fmt.Sprintf("'%s' exists. Replace or Add Suffix?", filename))
			pages.SwitchToPage("collision")
		} else {
			errorModal.SetText(fmt.Sprintf("Cannot rename to '%s': File exists.", ent.Title))
			pages.SwitchToPage("error")
		}
		return
	}
	commitSave(newPath, enc)
}

func commitSave(newPath string, enc []byte) {
	for id, localPath := range pendingFilePaths {
		data, err := os.ReadFile(localPath)
		if err == nil {
			encData, _ := encrypt(data)
			os.WriteFile(filepath.Join(getAttachmentDir(), id), encData, 0600)
		}
	}
	if currentPath != "" && currentPath != newPath {
		os.Remove(currentPath)
	}
	os.WriteFile(newPath, enc, 0600)
	refreshTree(searchField.GetText())
	pages.SwitchToPage("main")
	loadEntry(newPath)
}
