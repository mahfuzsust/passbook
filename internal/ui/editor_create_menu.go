package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type createMenuModel struct {
	cursor int
}

func newCreateMenuModel() createMenuModel {
	return createMenuModel{}
}

var createMenuItems = []struct {
	label, desc string
	key         string
	type_       EntryType
	isFolder    bool
}{
	{"Login", "Password & 2FA", "l", TypeLogin, false},
	{"Card", "Credit/Debit Details", "c", TypeCard, false},
	{"Note", "Secure Text", "n", TypeNote, false},
	{"File", "Encrypted Attachments", "f", TypeFile, false},
	{"Folder", "Organize entries", "d", "", true},
}

func (m *Model) selectCreateMenuItem(i int) tea.Cmd {
	item := createMenuItems[i]
	if item.isFolder {
		m.overlay = overlayFolderCreate
		m.folder = newFolderCreateModel()
		return nil
	}
	return m.newEntry(item.type_)
}

func (m *Model) updateCreateMenu(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "up", "k":
		if m.createMenu.cursor > 0 {
			m.createMenu.cursor--
		}
	case "down", "j":
		if m.createMenu.cursor < len(createMenuItems)-1 {
			m.createMenu.cursor++
		}
	case "enter":
		return *m, m.selectCreateMenuItem(m.createMenu.cursor)
	default:
		for i, item := range createMenuItems {
			if key == item.key {
				return *m, m.selectCreateMenuItem(i)
			}
		}
	}
	return *m, nil
}

func (m Model) viewCreateMenu() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create New "))
	b.WriteString("\n\n")
	for i, item := range createMenuItems {
		line := item.label + " — " + item.desc
		if i == m.createMenu.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return centerModal(b.String(), m.width, m.height, 30, 14, 50, 20, 0.4, 0.5)
}
