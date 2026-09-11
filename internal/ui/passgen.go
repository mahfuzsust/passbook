package ui

import (
	"strconv"
	"strings"

	"passbook/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// passgenStops is the number of focusable stops in the generator dialog:
// 0=length, 1=upper, 2=lower, 3=special, 4=Refresh button, 5=Use button.
const passgenStops = 6

type passgenModel struct {
	length   textinput.Model
	upper    bool
	lower    bool
	special  bool
	preview  string
	strength string
	cursor   int
}

func newPassgenModel() passgenModel {
	l := textinput.New()
	l.SetValue("28")
	l.CharLimit = 3
	l.Width = 5
	l.Focus()
	return passgenModel{length: l, upper: true, lower: true, special: true, cursor: 0}
}

func (p *passgenModel) updatePreview() {
	l, _ := strconv.Atoi(p.length.Value())
	if l <= 0 {
		l = 28
	}
	pass := utils.GeneratePassword(l, p.upper, p.lower, p.special)
	p.preview = pass
	p.strength = formatStrengthBar(pass)
}

func (m *Model) updatePassgenKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if isTabForward(msg) {
		m.passgen.cursor = (m.passgen.cursor + 1) % passgenStops
		return *m, m.passgen.applyCursorFocus()
	}
	if isTabBackward(msg) {
		m.passgen.cursor--
		if m.passgen.cursor < 0 {
			m.passgen.cursor = passgenStops - 1
		}
		return *m, m.passgen.applyCursorFocus()
	}

	switch key {
	case "esc":
		m.overlay = overlayEditor
	case "enter":
		switch m.passgen.cursor {
		case 4:
			m.passgen.updatePreview()
		case 5:
			m.editor.password.SetValue(m.passgen.preview)
			m.editor.strength = formatStrengthBar(m.passgen.preview)
			m.overlay = overlayEditor
		}
	case "down":
		m.passgen.cursor = (m.passgen.cursor + 1) % passgenStops
		return *m, m.passgen.applyCursorFocus()
	case "up":
		m.passgen.cursor--
		if m.passgen.cursor < 0 {
			m.passgen.cursor = passgenStops - 1
		}
		return *m, m.passgen.applyCursorFocus()
	case " ":
		switch m.passgen.cursor {
		case 1:
			m.passgen.upper = !m.passgen.upper
		case 2:
			m.passgen.lower = !m.passgen.lower
		case 3:
			m.passgen.special = !m.passgen.special
		}
		m.passgen.updatePreview()
	case "left":
		if m.passgen.cursor == 5 {
			m.passgen.cursor = 4
		}
	case "right":
		if m.passgen.cursor == 4 {
			m.passgen.cursor = 5
		}
	case "r":
		m.passgen.updatePreview()
	default:
		if m.passgen.cursor == 0 {
			m.passgen.length, _ = m.passgen.length.Update(msg)
			m.passgen.updatePreview()
		}
	}
	return *m, nil
}

func (m Model) viewPassgen() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Generator "))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Generated:"))
	b.WriteString("\n")
	b.WriteString(successStyle.Render(m.passgen.preview))
	b.WriteString("\n")
	if m.passgen.strength != "" {
		b.WriteString(m.passgen.strength)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Length:"))
	b.WriteString(" ")
	b.WriteString(m.passgen.length.View())
	b.WriteString("\n")
	check := func(label string, on, focused bool) string {
		box := "[ ]"
		if on {
			box = "[x]"
		}
		line := box + " " + label
		if focused {
			line = focusedBorderStyle.Render(line)
		}
		return line
	}
	b.WriteString(check("A-Z", m.passgen.upper, m.passgen.cursor == 1))
	b.WriteString("  ")
	b.WriteString(check("a-z", m.passgen.lower, m.passgen.cursor == 2))
	b.WriteString("  ")
	b.WriteString(check("Special", m.passgen.special, m.passgen.cursor == 3))
	b.WriteString("\n\n")
	b.WriteString(renderButton("Refresh", m.passgen.cursor == 4, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Use", m.passgen.cursor == 5, false))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("\u2191\u2193/Tab navigate  Space toggle  r regenerate  Enter select  Esc cancel"))
	return centerModal(b.String(), m.width, m.height, 45, 20, 70, 30, 0.6, 0.6)
}

func (p *passgenModel) applyCursorFocus() tea.Cmd {
	p.length.Blur()
	if p.cursor == 0 {
		p.length.Focus()
		return textinput.Blink
	}
	return nil
}
