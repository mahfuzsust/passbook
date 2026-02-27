package ui

import (
	"strings"
	"time"

	"passbook/internal/config"
	"passbook/internal/crypto"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/rivo/tview"
	qrcode "github.com/skip2/go-qrcode"
)

var (
	uiTempMasterKey []byte

	uiPinSetupForm *tview.Form

	uiPinCreateForm   *tview.Form
	uiPinCreateStatus *tview.TextView

	uiTotpSetupFlex   *tview.Flex
	uiTotpSetupForm   *tview.Form
	uiTotpSetupStatus *tview.TextView
	uiPendingTotp     string

	uiPinVerifyForm   *tview.Form
	uiPinVerifyStatus *tview.TextView
	uiPinConfig       *crypto.PinConfig
)

func pinDigitAccept(text string, ch rune) bool {
	return ch >= '0' && ch <= '9' && len(text) <= 6
}

func setupPin() {
	setupPinSetupMenu()
	setupPinCreate()
	setupTotpSetup()
	setupPinVerify()
}

// ── PIN method selection ────────────────────────────────────────────

func setupPinSetupMenu() {
	uiPinSetupForm = tview.NewForm()
	uiPinSetupForm.AddButton("6-Digit PIN", func() { showPinCreate() })
	uiPinSetupForm.AddButton("Authenticator App", func() { showTotpSetup() })

	uiPinSetupForm.SetBorder(true).
		SetTitle(" Set Up Two-Factor Authentication ").
		SetTitleAlign(tview.AlignCenter)
	styleForm(uiPinSetupForm)

	uiPinSetupForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiApp.Stop()
			return nil
		}
		return event
	})
	enableButtonNav(uiPinSetupForm)

	uiPages.AddPage("pin_setup",
		newResponsiveModal(uiPinSetupForm, 50, 7, 70, 10, 0.5, 0.35), true, false)
}

func showPinSetup() {
	uiPages.SwitchToPage("pin_setup")
	uiApp.SetFocus(uiPinSetupForm)
}

// ── PIN creation ────────────────────────────────────────────────────

func setupPinCreate() {
	uiPinCreateForm = tview.NewForm()

	uiPinCreateForm.AddPasswordField("Enter PIN", "", 12, '*', nil)
	setPinAcceptance(uiPinCreateForm, 0)

	uiPinCreateForm.AddPasswordField("Confirm PIN", "", 12, '*', nil)
	setPinAcceptance(uiPinCreateForm, 1)

	uiPinCreateStatus = tview.NewTextView().SetDynamicColors(true)
	uiPinCreateStatus.SetLabel(" ")
	uiPinCreateStatus.SetSize(1, 0)
	uiPinCreateStatus.SetScrollable(false)
	uiPinCreateForm.AddFormItem(uiPinCreateStatus)

	uiPinCreateForm.AddButton("Save", doSavePin)
	uiPinCreateForm.AddButton("Back", func() { showPinSetup() })

	uiPinCreateForm.SetBorder(true).
		SetTitle(" Create 6-Digit PIN ").
		SetTitleAlign(tview.AlignCenter)
	styleForm(uiPinCreateForm)

	uiPinCreateForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			showPinSetup()
			return nil
		case tcell.KeyEnter:
			focused := uiApp.GetFocus()
			for i := 0; i < uiPinCreateForm.GetButtonCount(); i++ {
				if focused == uiPinCreateForm.GetButton(i) {
					return event
				}
			}
			doSavePin()
			return nil
		}
		return event
	})
	enableButtonNav(uiPinCreateForm)

	uiPages.AddPage("pin_create",
		newResponsiveModal(uiPinCreateForm, 45, 12, 65, 16, 0.45, 0.4), true, false)
}

func showPinCreate() {
	uiPinCreateForm.GetFormItem(0).(*tview.InputField).SetText("")
	uiPinCreateForm.GetFormItem(1).(*tview.InputField).SetText("")
	uiPinCreateStatus.SetText("")
	uiPages.SwitchToPage("pin_create")
	uiApp.SetFocus(uiPinCreateForm)
}

func doSavePin() {
	pin := uiPinCreateForm.GetFormItem(0).(*tview.InputField).GetText()
	confirm := uiPinCreateForm.GetFormItem(1).(*tview.InputField).GetText()

	if len(pin) != 6 {
		uiPinCreateStatus.SetText("[red]PIN must be exactly 6 digits.")
		return
	}
	if pin != confirm {
		uiPinCreateStatus.SetText("[red]PINs do not match.")
		return
	}

	pinKey, err := crypto.GeneratePinKey()
	if err != nil {
		uiPinCreateStatus.SetText("[red]Failed to generate PIN key.")
		return
	}

	dataDir := config.ExpandPath(uiDataDir)
	cfg := crypto.PinConfig{
		Mode:   "pin",
		PinKey: pinKey,
		PinTag: crypto.ComputePinTag(pinKey, pin),
	}
	if err := crypto.WritePinConfig(dataDir, uiTempMasterKey, cfg); err != nil {
		uiPinCreateStatus.SetText("[red]Failed to save PIN.")
		return
	}

	wipeTempMasterKey()
	enterMain()
}

// ── TOTP setup ──────────────────────────────────────────────────────

func setupTotpSetup() {
	uiTotpSetupFlex = tview.NewFlex().SetDirection(tview.FlexRow)
	uiTotpSetupFlex.SetBorder(true).
		SetTitle(" Authenticator App Setup ").
		SetTitleAlign(tview.AlignCenter)

	uiTotpSetupForm = tview.NewForm()
	uiTotpSetupForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			uiPendingTotp = ""
			showPinSetup()
			return nil
		case tcell.KeyCtrlY:
			if uiPendingTotp != "" {
				_ = clipboard.WriteAll(uiPendingTotp)
				if uiTotpSetupStatus != nil {
					uiTotpSetupStatus.SetText("[green]Secret copied!")
				}
			}
			return nil
		case tcell.KeyEnter:
			focused := uiApp.GetFocus()
			for i := 0; i < uiTotpSetupForm.GetButtonCount(); i++ {
				if focused == uiTotpSetupForm.GetButton(i) {
					return event
				}
			}
			doSaveTotp()
			return nil
		}
		return event
	})

	enableButtonNav(uiTotpSetupForm)

	uiPages.AddPage("totp_setup",
		newResponsiveModal(uiTotpSetupFlex, 55, 34, 75, 44, 0.8, 0.9), true, false)
}

func showTotpSetup() {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "PassBook",
		AccountName: "vault",
		Period:      30,
		Digits:      otp.DigitsSix,
	})
	if err != nil {
		return
	}
	uiPendingTotp = key.Secret()

	uiTotpSetupFlex.Clear()

	qrStr, qrLines := renderQRCode(key.URL())
	if qrLines > 0 {
		qrTV := tview.NewTextView().SetDynamicColors(true)
		qrTV.SetText(qrStr)
		uiTotpSetupFlex.AddItem(qrTV, qrLines, 0, false)
	}

	secretRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	secretTV := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	secretTV.SetText("[yellow]" + formatTotpSecret(uiPendingTotp) + " [white]| [green::b]Ctrl+Y[white::B] to copy")
	secretRow.AddItem(secretTV, 0, 1, false)
	uiTotpSetupFlex.AddItem(secretRow, 1, 0, false)

	uiTotpSetupForm.Clear(true)

	uiTotpSetupForm.AddInputField("Code", "", 12, pinDigitAccept, nil)

	uiTotpSetupStatus = tview.NewTextView().SetDynamicColors(true)
	uiTotpSetupStatus.SetLabel(" ")
	uiTotpSetupStatus.SetSize(1, 0)
	uiTotpSetupStatus.SetScrollable(false)
	uiTotpSetupForm.AddFormItem(uiTotpSetupStatus)

	uiTotpSetupForm.AddButton("Verify", doSaveTotp)
	uiTotpSetupForm.AddButton("Back", func() {
		uiPendingTotp = ""
		showPinSetup()
	})

	styleForm(uiTotpSetupForm)
	uiTotpSetupFlex.AddItem(uiTotpSetupForm, 0, 1, true)

	uiPages.SwitchToPage("totp_setup")
	uiApp.SetFocus(uiTotpSetupForm)
}

func doSaveTotp() {
	codeItem := uiTotpSetupForm.GetFormItemByLabel("Code")
	if codeItem == nil {
		return
	}
	code := codeItem.(*tview.InputField).GetText()

	if len(code) != 6 {
		uiTotpSetupStatus.SetText("[red]Enter the 6-digit code from your app.")
		return
	}

	if !validateTOTP(code, uiPendingTotp) {
		uiTotpSetupStatus.SetText("[red]Invalid code. Please try again.")
		codeItem.(*tview.InputField).SetText("")
		return
	}

	dataDir := config.ExpandPath(uiDataDir)
	cfg := crypto.PinConfig{
		Mode:       "totp",
		TotpSecret: uiPendingTotp,
	}
	if err := crypto.WritePinConfig(dataDir, uiTempMasterKey, cfg); err != nil {
		uiTotpSetupStatus.SetText("[red]Failed to save TOTP config.")
		return
	}

	uiPendingTotp = ""
	wipeTempMasterKey()
	enterMain()
}

// ── PIN / TOTP verification ─────────────────────────────────────────

func setupPinVerify() {
	uiPinVerifyForm = tview.NewForm()
	uiPinVerifyForm.SetBorder(true).SetTitleAlign(tview.AlignCenter)
	styleForm(uiPinVerifyForm)

	uiPinVerifyForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			uiApp.Stop()
			return nil
		case tcell.KeyEnter:
			focused := uiApp.GetFocus()
			for i := 0; i < uiPinVerifyForm.GetButtonCount(); i++ {
				if focused == uiPinVerifyForm.GetButton(i) {
					return event
				}
			}
			doVerifyPin()
			return nil
		}
		return event
	})
	enableButtonNav(uiPinVerifyForm)

	uiPages.AddPage("pin_verify",
		newResponsiveModal(uiPinVerifyForm, 45, 10, 65, 14, 0.45, 0.35), true, false)
}

func showPinVerify(cfg *crypto.PinConfig) {
	uiPinConfig = cfg
	uiPinVerifyForm.Clear(true)

	var title, label string
	if cfg.Mode == "pin" {
		title = " Enter 6-Digit PIN "
		label = "PIN"
	} else {
		title = " Enter Authenticator Code "
		label = "Code"
	}
	uiPinVerifyForm.SetTitle(title)

	uiPinVerifyForm.AddPasswordField(label, "", 12, '*', nil)
	setPinAcceptance(uiPinVerifyForm, 0)

	uiPinVerifyStatus = tview.NewTextView().SetDynamicColors(true)
	uiPinVerifyStatus.SetLabel(" ")
	uiPinVerifyStatus.SetSize(1, 0)
	uiPinVerifyStatus.SetScrollable(false)
	uiPinVerifyForm.AddFormItem(uiPinVerifyStatus)

	uiPinVerifyForm.AddButton("Verify", doVerifyPin)
	uiPinVerifyForm.AddButton("Quit", func() { uiApp.Stop() })

	styleForm(uiPinVerifyForm)
	uiPages.SwitchToPage("pin_verify")
	uiApp.SetFocus(uiPinVerifyForm)
}

func doVerifyPin() {
	code := uiPinVerifyForm.GetFormItem(0).(*tview.InputField).GetText()
	if len(code) != 6 {
		uiPinVerifyStatus.SetText("[red]Enter a 6-digit code.")
		return
	}

	if uiPinConfig.Mode == "pin" {
		if !crypto.VerifyPinTag(uiPinConfig.PinKey, code, uiPinConfig.PinTag) {
			uiPinVerifyStatus.SetText("[red]Wrong PIN.")
			uiPinVerifyForm.GetFormItem(0).(*tview.InputField).SetText("")
			return
		}
	} else if uiPinConfig.Mode == "totp" {
		if !validateTOTP(code, uiPinConfig.TotpSecret) {
			uiPinVerifyStatus.SetText("[red]Invalid code.")
			uiPinVerifyForm.GetFormItem(0).(*tview.InputField).SetText("")
			return
		}
	}

	wipeTempMasterKey()
	enterMain()
}

// ── helpers ─────────────────────────────────────────────────────────

func setPinAcceptance(form *tview.Form, idx int) {
	field := form.GetFormItem(idx).(*tview.InputField)
	field.SetAcceptanceFunc(pinDigitAccept)
}

func wipeTempMasterKey() {
	if uiTempMasterKey != nil {
		crypto.WipeBytes(uiTempMasterKey)
		uiTempMasterKey = nil
	}
}

func enterMain() {
	refreshTree("")
	uiPages.SwitchToPage("main")
	uiApp.SetFocus(uiTreeView)
}

func formatTotpSecret(secret string) string {
	return strings.ToUpper(secret)
}

func validateTOTP(code, secret string) bool {
	ok, _ := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:     2,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return ok
}

func renderQRCode(url string) (string, int) {
	qr, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return "", 0
	}

	bmp := qr.Bitmap()
	rows := len(bmp)
	if rows == 0 {
		return "", 0
	}
	cols := len(bmp[0])

	var buf strings.Builder
	lines := 0

	for y := 0; y < rows; y += 2 {
		for x := 0; x < cols; x++ {
			top := bmp[y][x]
			bot := y+1 < rows && bmp[y+1][x]

			switch {
			case top && bot:
				buf.WriteString("[black:black] [-:-]")
			case top && !bot:
				buf.WriteString("[black:white]▀[-:-]")
			case !top && bot:
				buf.WriteString("[white:black]▀[-:-]")
			default:
				buf.WriteString("[white:white] [-:-]")
			}
		}
		buf.WriteString("\n")
		lines++
	}

	return buf.String(), lines
}
