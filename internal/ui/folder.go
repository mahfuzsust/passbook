package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"passbook/internal/store"
)

type folderModel struct {
	mode       string // create, rename, delete
	nameInput  textinput.Model
	folderID   int64
	deleteText string
	btnFocus   int
}

func newFolderCreateModel() folderModel {
	f := folderModel{mode: "create"}
	f.nameInput = newTextInput("Folder Name", false)
	f.nameInput.Focus()
	return f
}

func newFolderRenameModel(s *store.Store, folderID int64) folderModel {
	f := folderModel{mode: "rename", folderID: folderID}
	f.nameInput = newTextInput("Folder Name", false)
	if folder, _ := s.GetFolder(folderID); folder != nil {
		f.nameInput.SetValue(folder.Name)
	}
	f.nameInput.Focus()
	return f
}

func newFolderDeleteModel(s *store.Store, folderID int64) folderModel {
	f := folderModel{mode: "delete", folderID: folderID}
	folder, _ := s.GetFolder(folderID)
	if folder == nil {
		return f
	}
	count := s.CountEntriesInFolder(folderID)
	if count > 0 {
		f.deleteText = fmt.Sprintf(
			"Folder \"%s\" contains %d item(s).\nAll items inside will be permanently deleted.\n\nAre you sure?",
			folder.Name, count)
	} else {
		f.deleteText = fmt.Sprintf("Delete empty folder \"%s\"?", folder.Name)
	}
	return f
}

func isValidFolderName(name string) bool {
	return name != "" &&
		!strings.ContainsAny(name, `<>:"/\|?*`) &&
		name != "." && name != ".." &&
		!strings.HasPrefix(name, ".") &&
		!strings.HasPrefix(name, "_")
}

func (f *folderModel) update(msg tea.Msg) (folderModel, tea.Cmd) {
	f.nameInput, _ = f.nameInput.Update(msg)
	return *f, nil
}

func (m *Model) updateFolderDelete(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "left":
		if m.folder.btnFocus > 0 {
			m.folder.btnFocus--
		}
	case "right", "tab":
		if m.folder.btnFocus < 1 {
			m.folder.btnFocus++
		}
	case "enter":
		if m.folder.btnFocus == 0 {
			m.doFolderDelete()
		}
		m.overlay = overlayNone
	}
	return *m, nil
}

func (m *Model) updateFolderKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "enter":
		if m.folder.mode == "create" {
			m.doFolderCreate()
		} else if m.folder.mode == "rename" {
			m.doFolderRename()
		}
	default:
		m.folder.nameInput, _ = m.folder.nameInput.Update(msg)
	}
	return *m, nil
}

func (m *Model) doFolderCreate() {
	name := strings.TrimSpace(m.folder.nameInput.Value())
	if !isValidFolderName(name) {
		return
	}
	if _, err := m.store.CreateFolder(name); err != nil {
		return
	}
	m.main.refreshTree(m.store, m.main.search.Value())
	m.overlay = overlayNone
}

func (m *Model) doFolderRename() {
	name := strings.TrimSpace(m.folder.nameInput.Value())
	if !isValidFolderName(name) {
		return
	}
	if err := m.store.RenameFolder(m.folder.folderID, name); err != nil {
		return
	}
	m.main.refreshTree(m.store, m.main.search.Value())
	m.overlay = overlayNone
}

func (m *Model) doFolderDelete() {
	if m.folder.folderID == 0 {
		return
	}
	_ = m.store.DeleteFolder(m.folder.folderID)
	m.main.currentFolderID = 0
	m.main.currentEntryID = 0
	m.main.currentEnt = nil
	m.main.showContent = false
	m.main.refreshTree(m.store, m.main.search.Value())
}

func (m Model) viewFolderCreate() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" New Folder "))
	b.WriteString("\n\n")
	b.WriteString(m.folder.nameInput.View())
	b.WriteString("\n\n")
	b.WriteString(renderButton("Create", true, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Cancel", false, false))
	return centerModal(b.String(), m.width, m.height, 45, 9, 65, 13, 0.45, 0.3)
}

func (m Model) viewFolderRename() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Rename Folder "))
	b.WriteString("\n\n")
	b.WriteString(m.folder.nameInput.View())
	b.WriteString("\n\n")
	b.WriteString(renderButton("Rename", true, false))
	b.WriteString("  ")
	b.WriteString(renderButton("Cancel", false, false))
	return centerModal(b.String(), m.width, m.height, 45, 9, 65, 13, 0.45, 0.3)
}

func (m Model) viewFolderDelete() string {
	var b strings.Builder
	b.WriteString(m.folder.deleteText)
	b.WriteString("\n\n")
	delFocused := m.folder.btnFocus == 0
	cancelFocused := m.folder.btnFocus == 1
	b.WriteString(renderButton("Delete", delFocused, true))
	b.WriteString("  ")
	b.WriteString(renderButton("Cancel", cancelFocused, false))
	return centerModal(b.String(), m.width, m.height, 50, 8, 70, 12, 0.5, 0.35)
}
