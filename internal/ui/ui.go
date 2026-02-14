package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"passbook/internal/config"
	"passbook/internal/crypto"
	"passbook/internal/pb"
	"passbook/internal/platform"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"
	"google.golang.org/protobuf/proto"
)

func NewApp(c config.AppConfig) (*AppHandle, error) {
	uiCfg = c
	uiDataDir = config.ExpandPath(uiCfg.DataDir)

	if err := os.MkdirAll(uiDataDir, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(getAttachmentDir(), 0700); err != nil {
		return nil, err
	}
	var err error
	uiKDF, err = crypto.EnsureKDFSecret(uiDataDir)
	if err != nil {
		return nil, err
	}

	setupUI()
	uiPages.SwitchToPage("login")
	return &AppHandle{}, nil
}

type AppHandle struct{}

func expandPath(p string) string { return config.ExpandPath(p) }

func getAttachmentDir() string {
	return filepath.Join(uiDataDir, "_attachments")
}

func ensureKDFSecret() {
	if len(uiKDF.Salt) == 0 {
		p, err := crypto.EnsureKDFSecret(uiDataDir)
		if err == nil {
			uiKDF = p
		}
	}
}

func deriveKey(password string) []byte { return crypto.DeriveKey(password, uiKDF) }

func encrypt(plaintext []byte) ([]byte, error) { return crypto.Encrypt(uiMasterKey, plaintext) }

func decrypt(ciphertext []byte) ([]byte, error) { return crypto.Decrypt(uiMasterKey, ciphertext) }

func openURL(url string) error { return platform.OpenURL(url) }

func marshalEntry(e *pb.Entry) ([]byte, error) { return proto.Marshal(e) }

func unmarshalEntry(data []byte) (*pb.Entry, error) {
	e := &pb.Entry{}
	err := proto.Unmarshal(data, e)
	return e, err
}

func (a *AppHandle) Run() error {
	return uiApp.SetRoot(uiPages, true).EnableMouse(true).Run()
}

func (a *AppHandle) QueueUpdateDraw(f func()) {
	uiApp.QueueUpdateDraw(f)
}

func (a *AppHandle) DrawTOTP() { drawTOTP() }

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
	uiMasterKey = deriveKey(pwd)
	refreshTree("")

	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func setupLogin() {
	uiLoginForm = tview.NewForm()
	uiLoginForm.AddPasswordField("Master Password", "", 0, '*', nil)
	uiLoginForm.AddButton("Login", func() {
		pwd := uiLoginForm.GetFormItem(0).(*tview.InputField).GetText()
		goToMain(pwd)
	})

	uiLoginForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			pwd := uiLoginForm.GetFormItem(0).(*tview.InputField).GetText()
			goToMain(pwd)
		}
		return event
	})
	uiLoginForm.AddButton("Quit", func() { uiApp.Stop() })
	uiLoginForm.SetBorder(true).SetTitle(" PassBook Login ").SetTitleAlign(tview.AlignCenter)
	styleForm(uiLoginForm)

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).AddItem(nil, 0, 1, false).AddItem(uiLoginForm, 9, 1, true).AddItem(nil, 0, 1, false), 60, 1, true).
		AddItem(nil, 0, 1, false)
	uiPages.AddPage("login", flex, true, true)
}

func setupMainLayout() {
	uiSearchField = styleInput(tview.NewInputField().SetLabel("Search: ")).SetPlaceholder("Ctrl+F")
	uiSearchField.SetChangedFunc(func(text string) { refreshTree(text) })

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
	uiPages.AddPage("history", centeredModal(uiHistoryList, 50, 15), true, false)
}

func selectTreePath(path string) {
	if uiTreeView == nil {
		return
	}
	root := uiTreeView.GetRoot()
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
		uiTreeView.SetCurrentNode(node)
		if uiApp != nil {
			uiApp.SetFocus(uiTreeView)
		}
	}
}

func refreshTree(filter string) {
	root := uiTreeView.GetRoot()
	root.ClearChildren()
	basePath := expandPath(uiDataDir)

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
	if uiCurrentPath == "" {
		uiRightPages.SwitchToPage("empty")
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
		uiCurrentEnt = ent
		uiCurrentPath = path
		uiShowSensitive = false
		updateViewPane()
		uiRightPages.SwitchToPage("content")
	}
}

func updateViewPane() {
	uiViewFlex.Clear()
	uiAttachmentList.Clear()

	uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	uiViewTitle.SetText(uiCurrentEnt.Title)
	uiViewFlex.AddItem(makeRow("Title:", uiViewTitle), 1, 0, false)
	uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)

	switch EntryType(uiCurrentEnt.Type) {
	case TypeLogin:
		if uiCurrentEnt.Username != "" {
			uiViewSubtitle.SetText(uiCurrentEnt.Username)
			btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
				err := clipboard.WriteAll(uiCurrentEnt.Username)
				if err != nil {
					return
				}
				notifyCopied("Username")
			}))
			uiViewFlex.AddItem(makeRow("Username:", uiViewSubtitle, btnCopy), 1, 0, false)
		}

		if uiCurrentEnt.Password != "" {
			pass := strings.Repeat("*", len(uiCurrentEnt.Password))
			if uiShowSensitive {
				pass = uiCurrentEnt.Password
			}
			uiViewPassword.SetText(pass)
			btnPass := styleButton(tview.NewButton("cp").SetSelectedFunc(func() { copySensitive(uiCurrentEnt.Password, "Password") }))
			btnShow := styleButton(tview.NewButton("vw").SetSelectedFunc(func() { uiShowSensitive = !uiShowSensitive; updateViewPane() }))
			btnHist := styleButton(tview.NewButton("his").SetSelectedFunc(func() { showHistory() }))
			uiViewFlex.AddItem(makeRow("Password:", uiViewPassword, btnShow, btnPass, btnHist), 1, 0, false)
		} else {
			uiShowSensitive = false
		}

		if strings.TrimSpace(uiCurrentEnt.Link) != "" {
			linkText := tview.NewTextView().SetDynamicColors(true)
			linkText.SetText("[blue::u]" + uiCurrentEnt.Link + "[-:-:-]")
			btnOpen := styleButton(tview.NewButton("open").SetSelectedFunc(func() { _ = openURL(uiCurrentEnt.Link) }))
			btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
				err := clipboard.WriteAll(uiCurrentEnt.Link)
				if err != nil {
					return
				}
				notifyCopied("Link")
			}))
			uiViewFlex.AddItem(makeRow("Link:", linkText, btnOpen, btnCopy), 1, 0, false)
		}

		cleanSecret := strings.ReplaceAll(uiCurrentEnt.TotpSecret, " ", "")
		if cleanSecret != "" {
			uiViewFlex.AddItem(tview.NewTextView().SetText(""), 1, 0, false)
			btnTotp := styleButton(tview.NewButton("cp").SetSelectedFunc(func() {
				code, err := totp.GenerateCode(cleanSecret, time.Now())
				if err == nil {
					copySensitive(code, "TOTP")
				}
			}))
			uiViewFlex.AddItem(makeRow("TOTP:", uiViewTOTP, btnTotp), 1, 0, false)
			uiViewFlex.AddItem(makeRow("", uiViewTOTPBar), 1, 0, false)
			drawTOTP()
		} else {
			uiViewTOTP.SetText("")
			uiViewTOTPBar.SetText("")
		}

	case TypeCard:
		num := uiCurrentEnt.CardNumber
		if !uiShowSensitive && len(num) > 4 {
			num = "**** **** **** " + num[len(num)-4:]
		}
		uiViewSubtitle.SetText(num)
		btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() { copySensitive(uiCurrentEnt.CardNumber, "Card") }))
		btnShow := styleButton(tview.NewButton("vw").SetSelectedFunc(func() { uiShowSensitive = !uiShowSensitive; updateViewPane() }))
		uiViewFlex.AddItem(makeRow("Number:", uiViewSubtitle, btnShow, btnCopy), 1, 0, false)

		uiViewDetails.SetText(uiCurrentEnt.Expiry)
		uiViewFlex.AddItem(makeRow("Expiry:", uiViewDetails), 1, 0, false)

		cvv := "***"
		if uiShowSensitive {
			cvv = uiCurrentEnt.Cvv
		}
		uiViewPassword.SetText(cvv)
		uiViewFlex.AddItem(makeRow("CVV:", uiViewPassword), 1, 0, false)

	case TypeNote:
		uiCurrentEnt.Attachments = nil
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
