package ui

import (
	"strings"
	"time"

	"passbook/internal/crypto"
	"passbook/internal/store"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type pinModel struct {
	mode string // setup, create, totp, verify

	// create
	pinInput     textinput.Model
	confirmInput textinput.Model
	createFocus  int
	createBtn    int
	createStatus string

	// totp setup
	pendingSecret string
	qrText        string
	qrLines       int
	qrCols        int
	totpCode      textinput.Model
	totpStatus    string
	totpBtn       int
	totpFocus     int // 0=code, 1=buttons

	// verify
	verifyInput  textinput.Model
	verifyStatus string
	verifyBtn    int
	verifyFocus  int
	pinConfig    *store.PinConfig

	// setup menu
	setupBtn int
}

func newPinModel() pinModel {
	return pinModel{mode: "setup", setupBtn: 0}
}

func newPinVerifyModel(cfg *store.PinConfig) pinModel {
	p := pinModel{mode: "verify", pinConfig: cfg}
	if cfg.Mode == "pin" {
		p.verifyInput = newTextInput("PIN", true)
	} else {
		p.verifyInput = newTextInput("Code", false)
	}
	p.verifyInput.CharLimit = 6
	p.verifyInput.Focus()
	return p
}

func (p *pinModel) updateInputs(msg tea.Msg) (pinModel, tea.Cmd) {
	switch p.mode {
	case "create":
		if p.createFocus == 0 {
			p.pinInput, _ = p.pinInput.Update(msg)
		} else {
			p.confirmInput, _ = p.confirmInput.Update(msg)
		}
	case "totp":
		if p.totpFocus == 0 {
			p.totpCode, _ = p.totpCode.Update(msg)
		}
	case "verify":
		p.verifyInput, _ = p.verifyInput.Update(msg)
	}
	return *p, nil
}

func (m *Model) updatePinKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch m.pin.mode {
	case "setup":
		return m.updatePinSetup(key)
	case "create":
		return m.updatePinCreate(msg)
	case "totp":
		return m.updateTotpSetup(msg)
	case "verify":
		return m.updatePinVerify(msg)
	}
	return *m, nil
}

func (m *Model) updatePinSetup(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		return *m, tea.Quit
	case "left":
		if m.pin.setupBtn > 0 {
			m.pin.setupBtn--
		}
	case "right", "tab":
		if m.pin.setupBtn < 1 {
			m.pin.setupBtn++
		}
	case "enter":
		if m.pin.setupBtn == 0 {
			m.screen = screenPinCreate
			m.pin.mode = "create"
			m.pin.pinInput = newTextInput("Enter PIN", true)
			m.pin.pinInput.CharLimit = 6
			m.pin.confirmInput = newTextInput("Confirm PIN", true)
			m.pin.confirmInput.CharLimit = 6
			m.pin.pinInput.Focus()
			m.pin.createFocus = 0
		} else {
			m.startTotpSetup()
		}
	}
	return *m, nil
}

func (m *Model) updatePinCreate(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.screen = screenPinSetup
		m.pin.mode = "setup"
		return *m, nil
	case "tab", "down":
		if m.pin.createFocus < 1 {
			m.pin.createFocus++
			if m.pin.createFocus == 0 {
				return *m, focusInput(&m.pin.pinInput)
			}
			return *m, focusInput(&m.pin.confirmInput)
		}
		m.pin.createFocus = 2
	case "shift+tab", "up":
		if m.pin.createFocus > 0 {
			m.pin.createFocus--
			if m.pin.createFocus == 0 {
				return *m, focusInput(&m.pin.pinInput)
			}
			return *m, focusInput(&m.pin.confirmInput)
		}
	case "left":
		if m.pin.createFocus == 2 && m.pin.createBtn > 0 {
			m.pin.createBtn--
		}
	case "right":
		if m.pin.createFocus == 2 && m.pin.createBtn < 1 {
			m.pin.createBtn++
		}
	case "enter":
		if m.pin.createFocus < 2 {
			m.pin.createFocus++
			if m.pin.createFocus == 1 {
				return *m, focusInput(&m.pin.confirmInput)
			}
			m.pin.createFocus = 2
			return *m, nil
		}
		if m.pin.createBtn == 0 {
			m.doSavePin()
		} else {
			m.screen = screenPinSetup
			m.pin.mode = "setup"
		}
	default:
		if m.pin.createFocus == 0 {
			m.pin.pinInput, _ = m.pin.pinInput.Update(msg)
		} else if m.pin.createFocus == 1 {
			m.pin.confirmInput, _ = m.pin.confirmInput.Update(msg)
		}
	}
	return *m, nil
}

func (m *Model) doSavePin() {
	pin := m.pin.pinInput.Value()
	confirm := m.pin.confirmInput.Value()

	if len(pin) != 6 {
		m.pin.createStatus = "PIN must be exactly 6 digits."
		return
	}
	if pin != confirm {
		m.pin.createStatus = "PINs do not match."
		return
	}

	pinKey, err := crypto.GeneratePinKey()
	if err != nil {
		m.pin.createStatus = "Failed to generate PIN key."
		return
	}

	cfg := store.PinConfig{
		Mode:   "pin",
		PinKey: pinKey,
		PinTag: crypto.ComputePinTag(pinKey, pin),
	}
	if err := m.store.WritePinConfig(&cfg); err != nil {
		m.pin.createStatus = "Failed to save PIN."
		return
	}
	m.enterMain()
}

func (m *Model) startTotpSetup() {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "PassBook",
		AccountName: "v",
		Period:      30,
		Digits:      otp.DigitsSix,
		SecretSize:  10,
	})
	if err != nil {
		return
	}
	m.pin.pendingSecret = key.Secret()
	m.pin.qrText, m.pin.qrLines, m.pin.qrCols = renderQRCode(totpQRURL(key.Secret()))
	m.pin.totpCode = newTextInput("6-digit code", false)
	m.pin.totpCode.CharLimit = 6
	m.pin.totpCode.Focus()
	m.pin.totpFocus = 0
	m.pin.totpBtn = 0
	m.pin.totpStatus = ""
	m.screen = screenTotpSetup
	m.pin.mode = "totp"
}

func (m *Model) updateTotpSetup(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.pin.pendingSecret = ""
		m.screen = screenPinSetup
		m.pin.mode = "setup"
	case "ctrl+y":
		if m.pin.pendingSecret != "" {
			_ = clipboard.WriteAll(m.pin.pendingSecret)
			m.pin.totpStatus = "Secret copied!"
		}
	case "tab", "down":
		if m.pin.totpFocus == 0 {
			m.pin.totpFocus = 1
			m.pin.totpCode.Blur()
		}
	case "shift+tab", "up":
		if m.pin.totpFocus == 1 {
			m.pin.totpFocus = 0
			return *m, focusInput(&m.pin.totpCode)
		}
	case "left":
		if m.pin.totpFocus == 1 && m.pin.totpBtn > 0 {
			m.pin.totpBtn--
		}
	case "right":
		if m.pin.totpFocus == 1 && m.pin.totpBtn < 1 {
			m.pin.totpBtn++
		}
	case "enter":
		if m.pin.totpFocus == 0 {
			m.doSaveTotp()
		} else if m.pin.totpBtn == 0 {
			m.doSaveTotp()
		} else {
			m.pin.pendingSecret = ""
			m.screen = screenPinSetup
			m.pin.mode = "setup"
		}
	default:
		if m.pin.totpFocus == 0 {
			m.pin.totpCode, _ = m.pin.totpCode.Update(msg)
		}
	}
	return *m, nil
}

func (m *Model) doSaveTotp() {
	code := m.pin.totpCode.Value()
	if len(code) != 6 {
		m.pin.totpStatus = "Enter the 6-digit code from your app."
		return
	}
	if !validateTOTP(code, m.pin.pendingSecret) {
		m.pin.totpStatus = "Invalid code. Please try again."
		m.pin.totpCode.SetValue("")
		return
	}
	cfg := store.PinConfig{
		Mode:       "totp",
		TotpSecret: m.pin.pendingSecret,
	}
	if err := m.store.WritePinConfig(&cfg); err != nil {
		m.pin.totpStatus = "Failed to save TOTP config."
		return
	}
	m.pin.pendingSecret = ""
	m.enterMain()
}

func (m *Model) updatePinVerify(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		return *m, tea.Quit
	case "left":
		if m.pin.verifyFocus == 1 && m.pin.verifyBtn > 0 {
			m.pin.verifyBtn--
		}
	case "right", "tab":
		if m.pin.verifyFocus == 0 {
			m.pin.verifyFocus = 1
			m.pin.verifyInput.Blur()
		} else if m.pin.verifyBtn < 1 {
			m.pin.verifyBtn++
		}
	case "enter":
		if m.pin.verifyFocus == 0 {
			m.doVerifyPin()
		} else if m.pin.verifyBtn == 0 {
			m.doVerifyPin()
		} else {
			return *m, tea.Quit
		}
	default:
		if m.pin.verifyFocus == 0 {
			m.pin.verifyInput, _ = m.pin.verifyInput.Update(msg)
		}
	}
	return *m, nil
}

func (m *Model) doVerifyPin() {
	code := m.pin.verifyInput.Value()
	if len(code) != 6 {
		m.pin.verifyStatus = "Enter a 6-digit code."
		return
	}

	switch m.pin.pinConfig.Mode {
	case "pin":
		if !crypto.VerifyPinTag(m.pin.pinConfig.PinKey, code, m.pin.pinConfig.PinTag) {
			m.pin.verifyStatus = "Wrong PIN."
			m.pin.verifyInput.SetValue("")
			return
		}
	case "totp":
		if !validateTOTP(code, m.pin.pinConfig.TotpSecret) {
			m.pin.verifyStatus = "Invalid code."
			m.pin.verifyInput.SetValue("")
			return
		}
	}
	m.enterMain()
}

func (m *Model) enterMain() {
	m.screen = screenMain
	m.main = newMainModel()
	m.main.refreshTree(m.store, "")
}

func validateTOTP(code, secret string) bool {
	ok, _ := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      2,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return ok
}

func (m Model) viewPinSetup() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Set Up Two-Factor Authentication "))
	b.WriteString("\n\n")
	pinFocused := m.pin.setupBtn == 0
	totpFocused := m.pin.setupBtn == 1
	b.WriteString(renderButton("6-Digit PIN", pinFocused, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Authenticator App", totpFocused, false))
	return centerModal(b.String(), m.width, m.height, 50, 7, 70, 10, 0.5, 0.35)
}

func (m Model) viewPinCreate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create 6-Digit PIN "))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Enter PIN:"))
	b.WriteString("\n")
	b.WriteString(m.pin.pinInput.View())
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Confirm PIN:"))
	b.WriteString("\n")
	b.WriteString(m.pin.confirmInput.View())
	b.WriteString("\n")
	if m.pin.createStatus != "" {
		b.WriteString(errorStyle.Render(m.pin.createStatus))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	saveFocused := m.pin.createFocus == 2 && m.pin.createBtn == 0
	backFocused := m.pin.createFocus == 2 && m.pin.createBtn == 1
	b.WriteString(renderButton("Save", saveFocused, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Back", backFocused, false))
	return centerModal(b.String(), m.width, m.height, 45, 12, 65, 16, 0.45, 0.4)
}

func (m Model) viewTotpSetup() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Authenticator App Setup "))
	b.WriteString("\n\n")
	if m.pin.qrText != "" {
		b.WriteString(m.pin.qrText)
		b.WriteString("\n")
	}
	secret := formatTotpSecret(m.pin.pendingSecret)
	b.WriteString(skyStyle.Render(secret))
	b.WriteString(" | ")
	b.WriteString(successStyle.Render("Ctrl+Y") + " to copy")
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Code:"))
	b.WriteString("\n")
	b.WriteString(m.pin.totpCode.View())
	b.WriteString("\n")
	if m.pin.totpStatus != "" {
		if strings.Contains(m.pin.totpStatus, "copied") {
			b.WriteString(successStyle.Render(m.pin.totpStatus))
		} else {
			b.WriteString(errorStyle.Render(m.pin.totpStatus))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	verifyFocused := m.pin.totpFocus == 1 && m.pin.totpBtn == 0
	backFocused := m.pin.totpFocus == 1 && m.pin.totpBtn == 1
	b.WriteString(renderButton("Verify", verifyFocused, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Back", backFocused, false))
	return centerModal(b.String(), m.width, m.height, 55, 34, 75, 44, 0.8, 0.9)
}

func (m Model) viewPinVerify() string {
	title := " Enter 6-Digit PIN "
	if m.pin.pinConfig != nil && m.pin.pinConfig.Mode == "totp" {
		title = " Enter Authenticator Code "
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(m.pin.verifyInput.View())
	b.WriteString("\n")
	if m.pin.verifyStatus != "" {
		b.WriteString(errorStyle.Render(m.pin.verifyStatus))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	verifyFocused := m.pin.verifyFocus == 1 && m.pin.verifyBtn == 0
	quitFocused := m.pin.verifyFocus == 1 && m.pin.verifyBtn == 1
	b.WriteString(renderButton("Verify", verifyFocused, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Quit", quitFocused, false))
	return centerModal(b.String(), m.width, m.height, 45, 10, 65, 14, 0.45, 0.35)
}
