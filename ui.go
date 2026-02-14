package main

import (
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
	setupEditor()
}

func goToMain(pwd string) {
	if pwd == "" {
		return
	}
	ensureKDFSecret()
	masterKey = deriveKey(pwd)
	refreshTree("")

	pages.SwitchToPage("main")
	app.SetFocus(treeView)
}

func setupLogin() {
	loginForm = tview.NewForm()
	loginForm.AddPasswordField("Master Password", "", 0, '*', nil)
	loginForm.AddButton("Login", func() {
		pwd := loginForm.GetFormItem(0).(*tview.InputField).GetText()
		goToMain(pwd)
	})

	loginForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			pwd := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			goToMain(pwd)
		}
		return event
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

	mainFlex := newResponsiveSplit(leftFlex, rightPages, 0.30, 24, 40)

	mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			showCreateMenu()
			return nil
		case tcell.KeyCtrlE:
			if currentEnt != nil && currentPath != "" {
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
		case tcell.KeyCtrlQ:
			app.Stop()
			return nil
		case tcell.KeyEsc:
			app.SetFocus(treeView)
			return nil
		default:
			return event
		}
	})

	pages.AddPage("main", mainFlex, true, false)
}

func setupModals() {
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

func selectTreePath(path string) {
	if treeView == nil {
		return
	}
	root := treeView.GetRoot()
	if root == nil {
		return
	}

	var dfs func(n *tview.TreeNode) *tview.TreeNode
	dfs = func(n *tview.TreeNode) *tview.TreeNode {
		if n == nil {
			return nil
		}
		if ref := n.GetReference(); ref != nil {
			if s, ok := ref.(string); ok && s == path {
				return n
			}
		}
		for _, ch := range n.GetChildren() {
			if found := dfs(ch); found != nil {
				return found
			}
		}
		return nil
	}

	if node := dfs(root); node != nil {
		treeView.SetCurrentNode(node)
		if app != nil {
			app.SetFocus(treeView)
		}
	}
}

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
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			return
		}
		files, _ := os.ReadDir(dir)

		count := 0
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".pb") {
				name := strings.TrimSuffix(f.Name(), ".pb")
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

	ent, err := unmarshalEntry(decrypted)
	if err == nil {
		currentEnt = ent
		currentPath = path
		showSensitive = false
		updateViewPane()
		rightPages.SwitchToPage("content")
	}
}

func updateViewPane() {
	viewFlex.Clear()
	attachmentList.Clear()

	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	viewTitle.SetText(currentEnt.Title)
	viewFlex.AddItem(makeRow("Title:", viewTitle), 1, 0, false)
	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	switch EntryType(currentEnt.Type) {
	case TypeLogin:
		if currentEnt.Username != "" {
			viewSubtitle.SetText(currentEnt.Username)
			btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
				err := clipboard.WriteAll(currentEnt.Username)
				if err != nil {
					return
				}
				notifyCopied("Username")
			}))
			viewFlex.AddItem(makeRow("Username:", viewSubtitle, btnCopy), 1, 0, false)
		}

		if currentEnt.Password != "" {
			pass := strings.Repeat("*", len(currentEnt.Password))
			if showSensitive {
				pass = currentEnt.Password
			}
			viewPassword.SetText(pass)
			btnPass := styleButton(tview.NewButton("cp").SetSelectedFunc(func() { copySensitive(currentEnt.Password, "Password") }))
			btnShow := styleButton(tview.NewButton("vw").SetSelectedFunc(func() { showSensitive = !showSensitive; updateViewPane() }))
			btnHist := styleButton(tview.NewButton("his").SetSelectedFunc(func() { showHistory() }))
			viewFlex.AddItem(makeRow("Password:", viewPassword, btnShow, btnPass, btnHist), 1, 0, false)
		} else {
			showSensitive = false
		}

		if strings.TrimSpace(currentEnt.Link) != "" {
			linkText := tview.NewTextView().SetDynamicColors(true)
			linkText.SetText("[blue::u]" + currentEnt.Link + "[-:-:-]")
			btnOpen := styleButton(tview.NewButton("open").SetSelectedFunc(func() { _ = openURL(currentEnt.Link) }))
			btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
				err := clipboard.WriteAll(currentEnt.Link)
				if err != nil {
					return
				}
				notifyCopied("Link")
			}))
			viewFlex.AddItem(makeRow("Link:", linkText, btnOpen, btnCopy), 1, 0, false)
		}

		cleanSecret := strings.ReplaceAll(currentEnt.TotpSecret, " ", "")
		if cleanSecret != "" {
			viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
			btnTotp := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
				code, err := totp.GenerateCode(cleanSecret, time.Now())
				if err == nil {
					copySensitive(code, "TOTP")
				}
			}))
			viewFlex.AddItem(makeRow("TOTP:", viewTOTP, btnTotp), 1, 0, false)
			viewFlex.AddItem(makeRow("", viewTOTPBar), 1, 0, false)
			drawTOTP()
		} else {
			viewTOTP.SetText("")
			viewTOTPBar.SetText("")
		}

	case TypeCard:
		num := currentEnt.CardNumber
		if !showSensitive && len(num) > 4 {
			num = "**** **** **** " + num[len(num)-4:]
		}
		viewSubtitle.SetText(num)
		btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() { copySensitive(currentEnt.CardNumber, "Card") }))
		btnShow := styleButton(tview.NewButton("vw").SetSelectedFunc(func() { showSensitive = !showSensitive; updateViewPane() }))
		viewFlex.AddItem(makeRow("Number:", viewSubtitle, btnShow, btnCopy), 1, 0, false)

		viewDetails.SetText(currentEnt.Expiry)
		viewFlex.AddItem(makeRow("Expiry:", viewDetails), 1, 0, false)

		cvv := "***"
		if showSensitive {
			cvv = currentEnt.Cvv
		}
		viewPassword.SetText(cvv)
		viewFlex.AddItem(makeRow("CVV:", viewPassword), 1, 0, false)

	case TypeNote:
		currentEnt.Attachments = nil
	}

	if len(currentEnt.Attachments) > 0 {
		viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
		viewFlex.AddItem(tview.NewTextView().SetText("[yellow]Attachments:[-]").SetDynamicColors(true), 1, 0, false)

		for _, att := range currentEnt.Attachments {
			a := att
			label := fmt.Sprintf("[blue::u]➤ %s[-:-:-] [dim](%s)[-]", a.FileName, formatBytes(a.Size))
			attachmentList.AddItem(label, "", 0, func() { downloadAttachment(a) })
		}

		h := len(currentEnt.Attachments) * 2
		if h < 3 {
			h = 3
		}
		if h > 10 {
			h = 10
		}
		viewFlex.AddItem(attachmentList, h, 0, false)
	}

	viewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
	if strings.TrimSpace(currentEnt.CustomText) != "" {
		header := tview.NewTextView().SetText("[yellow]Notes:[-]").SetDynamicColors(true)
		btnNotesCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
			err := clipboard.WriteAll(currentEnt.CustomText)
			if err != nil {
				return
			}
			notifyCopied("Notes")
		}))
		viewFlex.AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(header, 0, 1, false).
			AddItem(tview.NewTextView().SetText(" "), 1, 0, false).
			AddItem(btnNotesCopy, 5, 0, false), 1, 0, false)
		viewCustom.SetText(currentEnt.CustomText)
		viewFlex.AddItem(viewCustom, 0, 1, false)
	} else {
		viewCustom.SetText("")
	}
	viewFlex.AddItem(viewStatus, 1, 0, false)
}

func deleteEntry() {
	if currentPath != "" {
		for _, att := range currentEnt.Attachments {
			err := os.Remove(filepath.Join(getAttachmentDir(), att.Id))
			if err != nil {
				return
			}
		}
		err := os.Remove(currentPath)
		if err != nil {
			return
		}
		currentPath = ""
		refreshTree(searchField.GetText())
	}
}

func notifyCopied(item string) {
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied![-]", item))
	go func() { time.Sleep(2 * time.Second); app.QueueUpdateDraw(func() { viewStatus.SetText("") }) }()
}

func copySensitive(text, item string) {
	err := clipboard.WriteAll(text)
	if err != nil {
		return
	}
	viewStatus.SetText(fmt.Sprintf("[green]✓ %s copied (clears in 30s)[-]", item))
	go func() {
		time.Sleep(30 * time.Second)
		curr, _ := clipboard.ReadAll()
		if curr == text {
			err := clipboard.WriteAll("")
			if err != nil {
				return
			}
			app.QueueUpdateDraw(func() { viewStatus.SetText("[yellow]Clipboard cleared[-]") })
		}
	}()
}

func downloadAttachment(att *Attachment) {
	data, err := os.ReadFile(filepath.Join(getAttachmentDir(), att.Id))
	if err != nil {
		viewStatus.SetText("[red]Failed to read attachment[-]")
		return
	}
	dec, err := decrypt(data)
	if err != nil {
		viewStatus.SetText("[red]Failed to decrypt attachment (wrong key?)[-]")
		return
	}

	var downDir string
	if runtime.GOOS == "windows" {
		downDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	} else {
		home, _ := os.UserHomeDir()
		downDir = filepath.Join(home, "Downloads")
	}

	dest := filepath.Join(downDir, att.FileName)
	err = os.WriteFile(dest, dec, 0644)
	if err != nil {
		viewStatus.SetText("[red]Failed to save to Downloads[-]")
		return
	}
	viewStatus.SetText(fmt.Sprintf("[green]✓ Saved to Downloads: %s[-]", att.FileName))
}

func drawTOTP() {
	if viewTOTP == nil || viewTOTPBar == nil {
		return
	}
	if pages != nil {
		if name, _ := pages.GetFrontPage(); name == "login" {
			viewTOTP.SetText("")
			viewTOTPBar.SetText("")
			return
		}
	}

	if currentEnt != nil && EntryType(currentEnt.Type) == TypeLogin {
		secret := strings.ReplaceAll(currentEnt.TotpSecret, " ", "")
		if secret != "" {
			code, err := totp.GenerateCode(secret, time.Now())
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
				return
			}
		}
	}
	viewTOTP.SetText("")
	viewTOTPBar.SetText("")
}

func showHistory() {
	if historyList == nil || pages == nil {
		return
	}
	historyList.Clear()
	for i := len(currentEnt.History) - 1; i >= 0; i-- {
		historyList.AddItem(currentEnt.History[i].Password, currentEnt.History[i].Date, 0, nil)
	}
	pages.SwitchToPage("history")
}

func showDeleteModal() {
	deleteModal.SetText("Delete " + currentEnt.Title + "?")
	pages.SwitchToPage("delete")
}

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
		f.AddItem(b, 5, 0, false)
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

type responsiveSplit struct {
	*tview.Flex
	left, right tview.Primitive
	leftRatio   float64
	minLeft     int
	minRight    int
	lastW       int
	lastH       int
}

func newResponsiveSplit(left, right tview.Primitive, leftRatio float64, minLeft, minRight int) *responsiveSplit {
	r := &responsiveSplit{
		Flex:      tview.NewFlex(),
		left:      left,
		right:     right,
		leftRatio: leftRatio,
		minLeft:   minLeft,
		minRight:  minRight,
	}
	r.Flex.AddItem(left, 0, 1, true)
	r.Flex.AddItem(right, 0, 1, false)
	return r
}

func (r *responsiveSplit) Draw(screen tcell.Screen) {
	x, y, w, h := r.GetRect()
	if w != r.lastW || h != r.lastH {
		leftW := int(float64(w) * r.leftRatio)
		if leftW < r.minLeft {
			leftW = r.minLeft
		}
		if w-leftW < r.minRight {
			leftW = w - r.minRight
		}
		if leftW < 0 {
			leftW = 0
		}
		if w-leftW < 0 {
			leftW = 0
		}

		r.Flex.SetRect(x, y, w, h)
		r.Flex.ResizeItem(r.left, leftW, 0)
		r.Flex.ResizeItem(r.right, 0, 1)

		r.lastW, r.lastH = w, h
	}
	r.Flex.Draw(screen)
}
