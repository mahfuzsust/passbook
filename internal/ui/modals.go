package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type modalModel struct {
	mode       string // delete, history, error
	text       string
	history    []PasswordHistory
	btnFocus   int
}

func newDeleteModal(title string) modalModel {
	return modalModel{mode: "delete", text: "Delete " + title + "?"}
}

func newHistoryModal(history []PasswordHistory) modalModel {
	return modalModel{mode: "history", history: history}
}

func newErrorModal(text string) modalModel {
	return modalModel{mode: "error", text: text}
}

func (m *Model) updateModal(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "left":
		if m.modals.btnFocus > 0 {
			m.modals.btnFocus--
		}
	case "right", "tab":
		if m.modals.btnFocus < 1 {
			m.modals.btnFocus++
		}
	case "enter":
		if m.overlay == overlayDelete {
			if m.modals.btnFocus == 0 {
				deleteEntry(m)
			}
			m.overlay = overlayNone
		} else if m.overlay == overlayFolderDelete {
			if m.modals.btnFocus == 0 {
				m.doFolderDelete()
			}
			m.overlay = overlayNone
		}
	}
	return *m, nil
}

func (m Model) viewDeleteModal() string {
	var b strings.Builder
	b.WriteString(m.modals.text)
	b.WriteString("\n\n")
	b.WriteString(renderButton("Delete", m.modals.btnFocus == 0, true))
	b.WriteString("  ")
	b.WriteString(renderButton("Cancel", m.modals.btnFocus == 1, false))
	return centerModal(b.String(), m.width, m.height, 40, 7, 60, 10, 0.4, 0.3)
}

func (m Model) viewHistory() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" History "))
	b.WriteString("\n\n")
	for i := len(m.modals.history) - 1; i >= 0; i-- {
		h := m.modals.history[i]
		b.WriteString(h.Password)
		b.WriteString(dimStyle.Render(" — " + h.Date))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Esc to close"))
	return centerModal(b.String(), m.width, m.height, 50, 15, 80, 25, 0.6, 0.65)
}

func (m Model) viewErrorModal() string {
	var b strings.Builder
	b.WriteString(errorStyle.Render(m.modals.text))
	b.WriteString("\n\n")
	b.WriteString(renderButton("OK", true, false))
	return centerModal(b.String(), m.width, m.height, 45, 7, 70, 10, 0.45, 0.3)
}
