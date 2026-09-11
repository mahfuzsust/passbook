package ui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) updateOverlayKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	switch m.overlay {
	case overlayCreateMenu:
		return m.updateCreateMenu(key)
	case overlayEditor:
		return m.updateEditorKey(msg)
	case overlayPassgen:
		return m.updatePassgenKey(msg)
	case overlayFileBrowser:
		return m.updateFileBrowser(key)
	case overlayFolderSelect:
		return m.updateFolderSelectKey(key)
	case overlayQuickCopy:
		return m.updateQuickCopy(key)
	case overlayChangePwd:
		return m.updateChangePwdKey(msg)
	case overlayFolderCreate, overlayFolderRename:
		return m.updateFolderKey(msg)
	case overlayFolderDelete:
		return m.updateFolderDelete(key)
	case overlayDelete:
		return m.updateModal(key)
	case overlayHistory:
		if key == "esc" || key == "enter" {
			m.overlay = overlayNone
		}
	case overlayError:
		if key == "enter" || key == "esc" {
			m.overlay = overlayNone
		}
	}
	return *m, nil
}

func (m Model) renderOverlay() string {
	switch m.overlay {
	case overlayCreateMenu:
		return m.viewCreateMenu()
	case overlayEditor:
		return m.viewEditor()
	case overlayPassgen:
		return m.viewPassgen()
	case overlayFileBrowser:
		return m.viewFileBrowser()
	case overlayFolderSelect:
		return m.viewFolderSelect()
	case overlayQuickCopy:
		return m.viewQuickCopy()
	case overlayChangePwd:
		return m.viewChangePwd()
	case overlayFolderCreate:
		return m.viewFolderCreate()
	case overlayFolderRename:
		return m.viewFolderRename()
	case overlayFolderDelete:
		return m.viewFolderDelete()
	case overlayDelete:
		return m.viewDeleteModal()
	case overlayHistory:
		return m.viewHistory()
	case overlayError:
		return m.viewErrorModal()
	}
	return ""
}

func (m *Model) dismissOverlay() {
	m.overlay = overlayNone
}

func (m *Model) openEditor(ent *Entry) tea.Cmd {
	m.overlay = overlayEditor
	folderID := m.main.currentFolderID
	if m.main.currentEntryID != 0 {
		// editing an existing entry: default to its own folder, not the
		// currently browsed tree folder (which loadEntry resets to 0).
		folderID = ent.FolderID
	}
	m.editor = newEditorModel(ent, m.main.currentEntryID, folderID, m.store)
	return m.editor.initFocus()
}

func (m *Model) newEntry(t EntryType) tea.Cmd {
	m.main.currentEntryID = 0
	ent := NewEntry(t)
	return m.openEditor(ent)
}
