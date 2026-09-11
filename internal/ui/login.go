package ui

import (
	"strings"

	"passbook/internal/store"
	"passbook/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type loginModel struct {
	freshInstall bool
	password     textinput.Model
	strength     string
	errorMsg     string
	buttonFocus  int // 0=submit, 1=quit
	focusField   int // 0=password, 1=buttons
}

func newLoginModel(fresh bool) loginModel {
	pw := newTextInput("Master Password", true)
	pw.Focus()
	return loginModel{
		freshInstall: fresh,
		password:     pw,
		focusField:   0,
	}
}

func (m *Model) updateLoginKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		return *m, tea.Quit
	case "tab", "down":
		if m.login.focusField == 0 {
			m.login.focusField = 1
			m.login.password.Blur()
		} else {
			m.login.buttonFocus = (m.login.buttonFocus + 1) % 2
		}
	case "shift+tab", "up":
		if m.login.focusField == 1 {
			if m.login.buttonFocus > 0 {
				m.login.buttonFocus--
			} else {
				m.login.focusField = 0
				return *m, focusInput(&m.login.password)
			}
		}
	case "left":
		if m.login.focusField == 1 && m.login.buttonFocus > 0 {
			m.login.buttonFocus--
		}
	case "right":
		if m.login.focusField == 1 && m.login.buttonFocus < 1 {
			m.login.buttonFocus++
		}
	case "enter":
		if m.login.focusField == 0 {
			m.goToMain(m.login.password.Value())
			return *m, nil
		}
		if m.login.buttonFocus == 0 {
			m.goToMain(m.login.password.Value())
		} else {
			return *m, tea.Quit
		}
	default:
		if m.login.focusField == 0 {
			var cmd tea.Cmd
			m.login.password, cmd = m.login.password.Update(msg)
			m.login.strength = formatStrengthBar(m.login.password.Value())
			m.login.errorMsg = ""
			return *m, cmd
		}
	}
	return *m, nil
}

func (m *Model) goToMain(pwd string) {
	if pwd == "" {
		return
	}

	dbExisted := store.DBExists(m.dbPath)
	freshInstall := !dbExisted

	s, err := store.Open(m.dbPath, pwd)
	if err != nil {
		if freshInstall {
			store.RemoveDBFiles(m.dbPath)
		}
		m.login.errorMsg = loginErrorMessage(err, freshInstall)
		return
	}
	m.store = s

	isNewVault := !m.store.HasEntries() && !m.store.PinConfigExists()

	if isNewVault {
		_, level, _ := utils.PasswordStrength(pwd)
		if level < utils.StrengthGood {
			m.closeAndCleanupStore(!dbExisted)
			m.login.errorMsg = "Password is too weak."
			return
		}
		m.screen = screenPinSetup
		m.pin = newPinModel()
		return
	}

	pinCfg, _ := m.store.ReadPinConfig()
	if pinCfg != nil && pinCfg.Mode != "" {
		m.screen = screenPinVerify
		m.pin = newPinVerifyModel(pinCfg)
	} else {
		m.screen = screenPinSetup
		m.pin = newPinModel()
	}
}

func (m *Model) closeAndCleanupStore(removeDB bool) {
	if m.store != nil {
		m.store.Close()
		m.store = nil
	}
	if removeDB {
		store.RemoveDBFiles(m.dbPath)
	}
}

func loginErrorMessage(err error, freshInstall bool) string {
	if store.IsCGODisabled(err) {
		return "Database unavailable. Reinstall a CGO-enabled build."
	}
	if freshInstall {
		return "Could not create vault."
	}
	return "Wrong password."
}

func (m Model) viewLogin() string {
	title := " PassBook Login "
	submitLabel := "Login"
	if m.freshInstall {
		title = " PassBook Setup "
		submitLabel = "Create Vault"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Master Password: "))
	b.WriteString("\n")
	b.WriteString(m.login.password.View())
	b.WriteString("\n")
	if m.login.strength != "" {
		b.WriteString(m.login.strength)
		b.WriteString("\n")
	}
	if m.login.errorMsg != "" {
		b.WriteString(errorStyle.Render(m.login.errorMsg))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	submitFocused := m.login.focusField == 1 && m.login.buttonFocus == 0
	quitFocused := m.login.focusField == 1 && m.login.buttonFocus == 1
	b.WriteString(renderButton(submitLabel, submitFocused, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Quit", quitFocused, false))

	content := b.String()
	return centerModal(content, m.width, m.height, 55, 10, 80, 15, 0.5, 0.4)
}
