package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"passbook/internal/store"
	"passbook/internal/utils"
)

type changePwdFocus int

const (
	cpfCurrent  changePwdFocus = 0
	cpfNew      changePwdFocus = 1
	cpfConfirm  changePwdFocus = 2
	cpfButtons  changePwdFocus = 3
)

type changePwdModel struct {
	current  textinput.Model
	newPwd   textinput.Model
	confirm  textinput.Model
	strength string
	status   string
	focus    changePwdFocus
	btnFocus int
}

func newChangePwdModel() changePwdModel {
	c := changePwdModel{focus: cpfCurrent}
	c.current = newTextInput("Current Password", true)
	c.newPwd = newTextInput("New Password", true)
	c.confirm = newTextInput("Confirm Password", true)
	c.current.Focus()
	return c
}

func (c *changePwdModel) blurAll() {
	c.current.Blur()
	c.newPwd.Blur()
	c.confirm.Blur()
}

func (c *changePwdModel) applyFocus() tea.Cmd {
	c.blurAll()
	switch c.focus {
	case cpfCurrent:
		c.current.Focus()
		return textinput.Blink
	case cpfNew:
		c.newPwd.Focus()
		return textinput.Blink
	case cpfConfirm:
		c.confirm.Focus()
		return textinput.Blink
	}
	return nil
}

func (c *changePwdModel) focusNext() tea.Cmd {
	if c.focus < cpfButtons {
		c.focus++
	} else {
		c.focus = cpfCurrent
	}
	return c.applyFocus()
}

func (c *changePwdModel) focusPrev() tea.Cmd {
	if c.focus > cpfCurrent {
		c.focus--
	} else {
		c.focus = cpfButtons
	}
	return c.applyFocus()
}

func (c *changePwdModel) updateFocused(msg tea.Msg) tea.Cmd {
	switch c.focus {
	case cpfCurrent:
		c.current, _ = c.current.Update(msg)
		return textinput.Blink
	case cpfNew:
		c.newPwd, _ = c.newPwd.Update(msg)
		c.strength = formatStrengthBar(c.newPwd.Value())
		return textinput.Blink
	case cpfConfirm:
		c.confirm, _ = c.confirm.Update(msg)
		return textinput.Blink
	}
	return nil
}

func (m *Model) updateChangePwdKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if isTabForward(msg) {
		return *m, m.changePwd.focusNext()
	}
	if isTabBackward(msg) {
		return *m, m.changePwd.focusPrev()
	}

	switch key {
	case "esc":
		m.overlay = overlayNone
	case "down":
		return *m, m.changePwd.focusNext()
	case "up":
		return *m, m.changePwd.focusPrev()
	case "enter":
		if m.changePwd.focus == cpfButtons {
			if m.changePwd.btnFocus == 0 {
				m.doChangePassword()
			} else {
				m.overlay = overlayNone
			}
		} else {
			return *m, m.changePwd.focusNext()
		}
	case "left":
		if m.changePwd.focus == cpfButtons && m.changePwd.btnFocus > 0 {
			m.changePwd.btnFocus--
		}
	case "right":
		if m.changePwd.focus == cpfButtons && m.changePwd.btnFocus < 1 {
			m.changePwd.btnFocus++
		}
	default:
		return *m, m.changePwd.updateFocused(msg)
	}
	return *m, nil
}

func (m *Model) doChangePassword() {
	currentPwd := m.changePwd.current.Value()
	newPwd := m.changePwd.newPwd.Value()
	confirmPwd := m.changePwd.confirm.Value()

	if currentPwd == "" || newPwd == "" || confirmPwd == "" {
		m.changePwd.status = "All fields are required."
		return
	}
	if newPwd != confirmPwd {
		m.changePwd.status = "New passwords do not match."
		return
	}
	if err := store.VerifyKey(m.dbPath, currentPwd); err != nil {
		m.changePwd.status = "Current password is incorrect."
		return
	}
	if currentPwd == newPwd {
		m.changePwd.status = "New password must be different from current."
		return
	}
	_, level, _ := utils.PasswordStrength(newPwd)
	if level < utils.StrengthGood {
		m.changePwd.status = "New password is too weak."
		return
	}
	if err := m.store.Rekey(newPwd); err != nil {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to change encryption key.")
		return
	}
	m.overlay = overlayNone
}

func (m Model) viewChangePwd() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Change Master Password "))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Current:"))
	b.WriteString(" ")
	b.WriteString(m.changePwd.current.View())
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("New:"))
	b.WriteString(" ")
	b.WriteString(m.changePwd.newPwd.View())
	b.WriteString("\n")
	if m.changePwd.strength != "" {
		b.WriteString(m.changePwd.strength)
		b.WriteString("\n")
	}
	b.WriteString(labelStyle.Render("Confirm:"))
	b.WriteString(" ")
	b.WriteString(m.changePwd.confirm.View())
	b.WriteString("\n")
	if m.changePwd.status != "" {
		b.WriteString(errorStyle.Render(m.changePwd.status))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	onButtons := m.changePwd.focus == cpfButtons
	b.WriteString(renderButton("Change", onButtons && m.changePwd.btnFocus == 0, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Cancel", onButtons && m.changePwd.btnFocus == 1, false))
	return centerModal(b.String(), m.width, m.height, 50, 15, 80, 19, 0.5, 0.4)
}
