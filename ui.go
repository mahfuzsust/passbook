package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"
)

func setupUI() {
	tview.Styles.ContrastBackgroundColor = colorUnfocusedBg
	tview.Styles.TitleColor = tcell.ColorLightSkyBlue

	setupLogin()
	setupMainLayout()
	setupModals()
	setupEditor() // In editor.go
}

// --- Setup Functions ---

func setupLogin() {
	loginForm = tview.NewForm()
	loginForm.AddPasswordField("Master Password", "", 0, '*', nil)
	loginForm.AddButton("Login", func() {
		pwd := loginForm.GetFormItem(0).(*tview.InputField).GetText()
		if pwd != "" {
			masterKey = deriveKey(pwd)
			refreshTree("")
			pages.SwitchToPage("main")
			app.SetFocus(treeView)
		}
	})
	loginForm.AddButton("Quit", func() { app.Stop() })
	loginForm.SetBorder(true).SetTitle(" PassBook Login ").SetTitleAlign(tview.AlignCenter)
	styleForm(loginForm)

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(nil, 0, 1, false).AddItem(loginForm, 9, 1, true).AddItem(nil, 0, 1, false), 60, 1, true).
		AddItem(nil, 0, 1, false)
	pages.AddPage("login", flex, true, true)
}

func setupMainLayout() {
	searchField = styleInput(tview.NewInputField().SetLabel("Search: ")).SetPlaceholder("Ctrl+F")
	searchField.SetChangedFunc(func(text string) { refreshTree(text) })

	root := tview.NewTreeNode("Vault").SetSelectable(false)
	treeView = tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	treeView.SetBorder(true).SetTitle(" Vault (Ctrl+A Add) ")
	treeView.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			node.SetExpanded(!node.IsExpanded())
		} else {
			loadEntry(ref.(string))
		}
	})

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(searchField, 1, 0, false).
		AddItem(tview.NewTextView().SetText(""), 1, 0, false).
		AddItem(treeView, 0, 1, true)

	viewFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	viewTitle = tview.NewTextView().SetDynamicColors(true)
	viewSubtitle = tview.NewTextView().SetDynamicColors(true)
	viewPassword = tview.NewTextView().SetDynamicColors(true)
	viewDetails = tview.NewTextView().SetDynamicColors(true)
	viewTOTP = tview.NewTextView().SetDynamicColors(true)
	viewTOTPBar = tview.NewTextView().SetDynamicColors(true)
	viewCustom = tview.NewTextView().SetDynamicColors(true)
	viewStatus = tview.NewTextView().SetDynamicColors(true)
	attachmentList = tview.NewList().ShowSecondaryText(false).SetMainTextColor(tcell.ColorSkyblue)

	emptyView := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter).
		SetText("\n\n\n[yellow]Select an item from the list to view details.[-]")

	rightPages = tview.NewPages()
	rightPages.SetBorder(true).SetTitle(" Contents (Ctrl+E Edit | Ctrl+D Delete) ")
	rightPages.AddPage("empty", emptyView, true, true)
	rightPages.AddPage("content", viewFlex, true, false)

	mainFlex := tview.NewFlex().AddItem(leftFlex, 35, 1, true).AddItem(rightPages, 0, 2, false)

	// Keybindings
	mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			showCreateMenu()
			return nil
		case tcell.KeyCtrlE:
			if currentPath != "" {
				openEditor(currentEnt)
			}
			return nil
		case tcell.KeyCtrlD:
			if currentPath != "" {
				showDeleteModal()
			}
			return nil
		case tcell.KeyCtrlF:
			app.SetFocus(searchField)
			return nil
		case tcell.KeyCtrlO:
			openSettings()
			return nil
		case tcell.KeyCtrlQ:
			app.Stop()
			return nil
		case tcell.KeyEsc:
			app.SetFocus(treeView)
			return nil
		}
		return event
	})

	pages.AddPage("main", mainFlex, true, false)
}

func setupModals() {
	settingsForm = tview.NewForm()
	settingsForm.SetBorder(true).SetTitle(" Settings ")
	pages.AddPage("settings", centeredModal(settingsForm, 50, 10), true, false)

	deleteModal = tview.NewModal().
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(index int, label string) {
			if label == "Delete" {
				deleteEntry()
			}
			pages.SwitchToPage("main")
			app.SetFocus(treeView)
		})
	pages.AddPage("delete", deleteModal, true, false)

	historyList = tview.NewList().ShowSecondaryText(true)
	historyList.SetBorder(true).SetTitle(" History (Esc to close) ")
	historyList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.SwitchToPage("main")
			app.SetFocus(rightPages)
		}
		return event
	})
	pages.AddPage("history", centeredModal(historyList, 50, 15), true, false)
}

// --- Logic ---

func refreshTree(filter string) {
	root := treeView.GetRoot()
	root.ClearChildren()
	basePath := expandPath(dataDir)

	cats := []struct {
		T EntryType
		I string
	}{{TypeLogin, "🔐"}, {TypeCard, "💳"}, {TypeNote, "📝"}, {TypeFile, "📎"}}

	for _, c := range cats {
		catNode := tview.NewTreeNode(fmt.Sprintf("%s %ss", c.I, c.T)).SetColor(tcell.ColorSkyblue).SetSelectable(true).SetExpanded(true)
		dir := filepath.Join(basePath, strings.ToLower(string(c.T))+"s")
		os.MkdirAll(dir, 0700)
		files, _ := os.ReadDir(dir)

		count := 0
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
				name := strings.TrimSuffix(f.Name(), ".md")
				if filter == "" || strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
					child := tview.NewTreeNode(name).SetReference(filepath.Join(dir, f.Name())).SetSelectable(true)
					catNode.AddChild(child)
					count++
				}
			}
		}
		if count > 0 || filter == "" {
			root.AddChild(catNode)
		}
	}
	if currentPath == "" {
		rightPages.SwitchToPage("empty")
	}
}

func loadEntry(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	decrypted, err := decrypt(data)
	if err != nil {
		return
	}

	currentEnt = Entry{}
	if json.Unmarshal(decrypted, &currentEnt) == nil {
		currentPath = path
		showSensitive = false
		updateViewPane()
		rightPages.SwitchToPage("content")
	}
}

func updateViewPane() {
	viewFlex.Clear()
	attachmentList.Clear() // FIX: Clear attachment data to prevent persistence

	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	// Title
	viewTitle.SetText(currentEnt.Title)
	viewFlex.AddItem(makeRow("Title:", viewTitle), 1, 0, false)
	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	switch currentEnt.Type {
	case TypeLogin:
		viewSubtitle.SetText(currentEnt.Username)
		btnCopy := styleButton(tview.NewButton("Copy").SetSelectedFunc(func() { clipboard.WriteAll(currentEnt.Username); notifyCopied("Username") }))
		viewFlex.AddItem(makeRow("Username:", viewSubtitle, btnCopy), 1, 0, false)

		pass := strings.Repeat("*", len(currentEnt.Password))
		if showSensitive {
			pass = currentEnt.Password
		}
		viewPassword.SetText(pass)
		btnPass := styleButton(tview.NewButton("Copy").SetSelectedFunc(func() { copySensitive(currentEnt.Password, "Password") }))
		btnShow := styleButton(tview.NewButton("View").SetSelectedFunc(func() { showSensitive = !showSensitive; updateViewPane() }))
		btnHist := styleButton(tview.NewButton("History").SetSelectedFunc(func() { showHistory() }))
		viewFlex.AddItem(makeRow("Password:", viewPassword, btnShow, btnPass, btnHist), 1, 0, false)

		viewDetails.SetText(currentEnt.Link)
		viewFlex.AddItem(makeRow("Link:", viewDetails), 1, 0, false)

		viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
		btnTotp := styleButton(tview.NewButton("Copy").SetSelectedFunc(func() {
			if currentEnt.TOTPSecret != "" {
				code, _ := totp.GenerateCode(strings.ReplaceAll(currentEnt.TOTPSecret, " ", ""), time.Now())
				copySensitive(code, "TOTP")
			}
		}))
		viewFlex.AddItem(makeRow("TOTP:", viewTOTP, btnTotp), 1, 0, false)
		viewFlex.AddItem(makeRow("", viewTOTPBar), 1, 0, false)
		drawTOTP()
		//currentEnt.Attachments = nil // Logins don't have attachments, ensure it's empty

	case TypeCard:
		num := currentEnt.CardNumber
		if !showSensitive && len(num) > 4 {
			num = "**** **** **** " + num[len(num)-4:]
		}
		viewSubtitle.SetText(num)
		btnCopy := styleButton(tview.NewButton("Copy").SetSelectedFunc(func() { copySensitive(currentEnt.CardNumber, "Card") }))
		btnShow := styleButton(tview.NewButton("View").SetSelectedFunc(func() { showSensitive = !showSensitive; updateViewPane() }))
		viewFlex.AddItem(makeRow("Number:", viewSubtitle, btnShow, btnCopy), 1, 0, false)

		viewDetails.SetText(currentEnt.Expiry)
		viewFlex.AddItem(makeRow("Expiry:", viewDetails), 1, 0, false)

		cvv := "***"
		if showSensitive {
			cvv = currentEnt.CVV
		}
		viewPassword.SetText(cvv)
		viewFlex.AddItem(makeRow("CVV:", viewPassword), 1, 0, false)
		//currentEnt.Attachments = nil // Cards don't have attachments, ensure it's empty

	case TypeNote:
		currentEnt.Attachments = nil
	}

	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
	viewFlex.AddItem(tview.NewTextView().SetText("[yellow]Notes:[-]").SetDynamicColors(true), 1, 0, false)
	viewCustom.SetText(currentEnt.CustomText)
	viewFlex.AddItem(viewCustom, 0, 1, false)
	viewFlex.AddItem(viewStatus, 1, 0, false)

	// Attachments (Conditional Render)
	if len(currentEnt.Attachments) > 0 {
		viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
		viewFlex.AddItem(tview.NewTextView().SetText("[yellow]Attachments:[-]").SetDynamicColors(true), 1, 0, false)

		for _, att := range currentEnt.Attachments {
			a := att
			label := fmt.Sprintf("[blue::u]➤ %s[-:-:-] [dim](%s)[-]", a.FileName, formatBytes(a.Size))
			attachmentList.AddItem(label, "", 0, func() { downloadAttachment(a) })
		}
		// Calculate height dynamically
		viewFlex.AddItem(attachmentList, len(currentEnt.Attachments)*2, 1, false)
	}
}

func deleteEntry() {
	if currentPath != "" {
		for _, att := range currentEnt.Attachments {
			os.Remove(filepath.Join(getAttachmentDir(), att.ID))
		}
		os.Remove(currentPath)
		currentPath = ""
		refreshTree(searchField.GetText())
	}
}

// --- Interaction Helpers ---

func notifyCopied(item string) {
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied![-]", item))
	go func() { time.Sleep(2 * time.Second); app.QueueUpdateDraw(func() { viewStatus.SetText("") }) }()
}

func copySensitive(text, item string) {
	clipboard.WriteAll(text)
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied (clears in 30s)[-]", item))
	go func() {
		time.Sleep(30 * time.Second)
		curr, _ := clipboard.ReadAll()
		if curr == text {
			clipboard.WriteAll("")
			app.QueueUpdateDraw(func() { viewStatus.SetText("[yellow]Clipboard cleared[-]") })
		}
	}()
}

func downloadAttachment(att Attachment) {
	data, err := os.ReadFile(filepath.Join(getAttachmentDir(), att.ID))
	if err != nil {
		return
	}
	dec, err := decrypt(data)
	if err != nil {
		return
	}

	var downDir string
	if runtime.GOOS == "windows" {
		downDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	} else {
		home, _ := os.UserHomeDir()
		downDir = filepath.Join(home, "Downloads")
	}

	os.WriteFile(filepath.Join(downDir, att.FileName), dec, 0644)
	notifyCopied(att.FileName + " downloaded")
}

// --- Utils ---

func lockApp() {
	masterKey = nil
	currentPath = ""
	currentEnt = Entry{}
	refreshTree("")
	loginForm.GetFormItem(0).(*tview.InputField).SetText("")
	pages.SwitchToPage("login")
	app.SetFocus(loginForm)
}

func drawTOTP() {
	if currentEnt.TOTPSecret != "" && currentEnt.Type == TypeLogin {
		code, err := totp.GenerateCode(strings.ReplaceAll(currentEnt.TOTPSecret, " ", ""), time.Now())
		if err == nil {
			viewTOTP.SetText(code)
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
			viewTOTPBar.SetText(fmt.Sprintf("[%s]%02ds [%s][-]", color, remain, barStr))
		}
	} else {
		viewTOTP.SetText("None")
		viewTOTPBar.SetText("")
	}
}

func showHistory() {
	historyList.Clear()
	for i := len(currentEnt.History) - 1; i >= 0; i-- {
		historyList.AddItem(currentEnt.History[i].Password, currentEnt.History[i].Date, 0, nil)
	}
	pages.SwitchToPage("history")
}

func openSettings() {
	settingsForm.Clear(true)
	settingsForm.AddInputField("Data Directory", dataDir, 40, nil, nil)
	settingsForm.AddButton("Save", func() {
		dataDir = settingsForm.GetFormItem(0).(*tview.InputField).GetText()
		saveConfig()
		refreshTree(searchField.GetText())
		pages.SwitchToPage("main")
	})
	settingsForm.AddButton("Cancel", func() { pages.SwitchToPage("main") })
	styleForm(settingsForm)
	pages.SwitchToPage("settings")
}

func showDeleteModal() {
	deleteModal.SetText("Delete " + currentEnt.Title + "?")
	pages.SwitchToPage("delete")
}

// --- Styling Helpers ---

func styleButton(b *tview.Button) *tview.Button {
	b.SetBackgroundColor(colorUnfocusedBg)
	b.SetLabelColor(tcell.ColorWhite)
	b.SetFocusFunc(func() {
		b.SetLabelColor(colorFocusedBg)
		b.SetBackgroundColor(tcell.ColorWhite)
	})
	b.SetBlurFunc(func() {
		b.SetBackgroundColor(colorUnfocusedBg)
		b.SetLabelColor(tcell.ColorWhite)
	})
	return b
}

func styleInput(f *tview.InputField) *tview.InputField {
	f.SetFieldBackgroundColor(colorUnfocusedBg)
	f.SetFocusFunc(func() { f.SetFieldBackgroundColor(colorFocusedBg) })
	f.SetBlurFunc(func() { f.SetFieldBackgroundColor(colorUnfocusedBg) })
	return f
}

func styleForm(f *tview.Form) {
	for i := 0; i < f.GetFormItemCount(); i++ {
		if input, ok := f.GetFormItem(i).(*tview.InputField); ok {
			styleInput(input)
		}
	}
	for i := 0; i < f.GetButtonCount(); i++ {
		styleButton(f.GetButton(i))
	}
}

func makeRow(label string, content *tview.TextView, buttons ...*tview.Button) *tview.Flex {
	f := tview.NewFlex().SetDirection(tview.FlexColumn)
	f.AddItem(tview.NewTextView().SetText(label).SetTextColor(tcell.ColorYellow), 12, 0, false)
	f.AddItem(content, 0, 1, false)
	for _, b := range buttons {
		f.AddItem(tview.NewTextView().SetText(" "), 1, 0, false)
		f.AddItem(b, 9, 0, false)
	}
	return f
}

func centeredModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(nil, 0, 1, false).AddItem(p, height, 1, true).AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
