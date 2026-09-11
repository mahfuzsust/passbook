package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"passbook/internal/store"
)

type mainModel struct {
	search               textinput.Model
	searchFocused        bool
	tree                 treeState
	currentFolderID      int64
	currentEntryID       int64
	currentEnt           *Entry
	showSensitive        bool
	showSensitiveClearAt time.Time
	viewStatus           string
	viewStatusClearAt    time.Time
	totpCode             string
	totpBar              string
	showContent          bool
}

func newMainModel() mainModel {
	s := textinput.New()
	s.Placeholder = "Ctrl+F"
	s.Width = 30
	return mainModel{search: s, tree: treeState{}}
}

func (m *mainModel) refreshTree(s *store.Store, filter string) {
	m.tree.refreshTree(s, filter)
}

func (m *mainModel) updateTOTP() {
	if m.currentEnt == nil || EntryType(m.currentEnt.Type) != TypeLogin {
		m.totpCode = ""
		m.totpBar = ""
		return
	}
	secret := strings.ReplaceAll(m.currentEnt.TotpSecret, " ", "")
	if secret == "" {
		m.totpCode = ""
		m.totpBar = ""
		return
	}
	m.totpCode, m.totpBar = formatTOTPDisplay(secret)
}

func (m *Model) updateMainKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if m.main.searchFocused {
		switch key {
		case "enter", "esc":
			m.main.searchFocused = false
			m.main.search.Blur()
			return *m, nil
		default:
			var cmd tea.Cmd
			m.main.search, cmd = m.main.search.Update(msg)
			m.main.refreshTree(m.store, m.main.search.Value())
			return *m, cmd
		}
	}

	if handled, cmd := m.handleViewAction(key); handled {
		return *m, cmd
	}

	switch key {
	case "ctrl+a":
		m.overlay = overlayCreateMenu
		m.createMenu = newCreateMenuModel()
	case "ctrl+e":
		if m.main.currentFolderID != 0 {
			m.overlay = overlayFolderRename
			m.folder = newFolderRenameModel(m.store, m.main.currentFolderID)
		} else if m.main.currentEnt != nil && m.main.currentEntryID != 0 {
			return *m, m.openEditor(m.main.currentEnt)
		}
	case "ctrl+d":
		if m.main.currentFolderID != 0 {
			m.overlay = overlayFolderDelete
			m.folder = newFolderDeleteModel(m.store, m.main.currentFolderID)
		} else if m.main.currentEntryID != 0 {
			m.overlay = overlayDelete
			m.modals = newDeleteModal(m.main.currentEnt.Title)
		}
	case "ctrl+n":
		m.overlay = overlayFolderCreate
		m.folder = newFolderCreateModel()
	case "ctrl+f":
		m.main.searchFocused = true
		return *m, focusInput(&m.main.search)
	case "ctrl+y":
		m.showQuickCopy()
	case "ctrl+p":
		m.overlay = overlayChangePwd
		m.changePwd = newChangePwdModel()
	case "ctrl+q":
		return *m, tea.Quit
	case "esc":
		m.main.searchFocused = false
		m.main.search.Blur()
	case "up", "k":
		m.main.tree.moveUp()
		m.syncSelectionFromTree()
	case "down", "j":
		m.main.tree.moveDown()
		m.syncSelectionFromTree()
	case "enter":
		ref, isToggle := m.main.tree.toggleOrSelect()
		if isToggle && ref.IsFolder {
			m.main.currentFolderID = ref.ID
			m.main.currentEntryID = 0
			m.main.currentEnt = nil
			m.main.showContent = false
			m.main.refreshTree(m.store, m.main.search.Value())
		} else if !ref.IsFolder && ref.ID != 0 {
			m.loadEntry(ref.ID)
		}
	}
	return *m, nil
}

func (m *Model) syncSelectionFromTree() {
	ref := m.main.tree.currentRef()
	if ref.IsFolder {
		m.main.currentFolderID = ref.ID
		m.main.currentEntryID = 0
		m.main.currentEnt = nil
		m.main.showContent = false
	} else if ref.ID != 0 {
		m.loadEntry(ref.ID)
	}
}

func (m *Model) loadEntry(id int64) {
	ent, err := m.store.LoadEntry(id)
	if err != nil {
		return
	}
	m.main.currentEnt = ent
	m.main.currentEntryID = id
	m.main.currentFolderID = 0
	m.main.showSensitive = false
	m.main.showSensitiveClearAt = time.Time{}
	m.main.showContent = true
	m.main.updateTOTP()
}

func (m Model) viewMain() string {
	leftH := m.height - 2
	searchLine := "Search: "
	if m.main.searchFocused {
		searchLine += m.main.search.View()
	} else {
		q := m.main.search.Value()
		if q == "" {
			searchLine += dimStyle.Render("Ctrl+F")
		} else {
			searchLine += q
		}
	}

	treeH := leftH - 3
	left := searchLine + "\n\n" + m.main.tree.render(treeH)

	right := m.viewDetailPane()
	return splitView(left, right, m.width, m.height, 0.30, 24, 40)
}

func (m Model) viewDetailPane() string {
	if !m.main.showContent || m.main.currentEnt == nil {
		return renderKeybindings()
	}
	return renderEntryView(m)
}

func renderKeybindings() string {
	bindings := [][2]string{
		{"Ctrl+A", "Create new item"},
		{"Ctrl+E", "Edit item / rename folder"},
		{"Ctrl+D", "Delete item / folder"},
		{"Ctrl+N", "Create new folder"},
		{"Ctrl+F", "Search vault"},
		{"Ctrl+Y", "Quick copy to clipboard"},
		{"u/c/l/t", "Copy username/password/link/TOTP"},
		{"Ctrl+P", "Change master password"},
		{"Ctrl+Q", "Quit"},
		{"Enter", "Open item / toggle folder"},
		{"Esc", "Focus tree view"},
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Keybindings "))
	b.WriteString("\n\n")
	for _, bind := range bindings {
		b.WriteString(skyStyle.Render(bind[0]))
		b.WriteString("  ")
		b.WriteString(bind[1])
		b.WriteString("\n")
	}
	return b.String()
}

func (m *Model) notifyCopied(item string) {
	m.main.viewStatus = successStyle.Render("✓ " + item + " copied!")
	m.main.viewStatusClearAt = time.Now().Add(3 * time.Second)
}
