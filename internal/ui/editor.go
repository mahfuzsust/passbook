package ui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"passbook/internal/store"
)

type editorModel struct {
	entryType     EntryType
	editingID     int64
	folderID      int64
	title         textinput.Model
	folderIdx     int
	folderOptions []string
	username      textinput.Model
	password      textinput.Model
	link          textinput.Model
	totpSecret    textinput.Model
	cardNumber    textinput.Model
	expiry        textinput.Model
	cvv           textinput.Model
	notes         textarea.Model
	strength      string
	focusOrder    []editorFocusTarget
	focusPos      int
	focusTarget   editorFocusTarget
	btnFocus      int
	saveDisabled  bool
	pendingAttach []Attachment
	pendingPaths  map[string]string
	attachCursor  int
}

func newEditorModel(ent *Entry, entryID, folderID int64, s *store.Store) editorModel {
	e := editorModel{
		entryType:     EntryType(ent.Type),
		editingID:     entryID,
		folderID:      folderID,
		pendingAttach: append([]Attachment{}, ent.Attachments...),
		pendingPaths:  make(map[string]string),
	}
	e.title = newTextInput("Title", false)
	e.title.SetValue(ent.Title)
	e.username = newTextInput("Username", false)
	e.username.SetValue(ent.Username)
	e.password = newTextInput("Password", false)
	e.password.SetValue(ent.Password)
	e.link = newTextInput("Link", false)
	e.link.SetValue(ent.Link)
	e.totpSecret = newTextInput("TOTP Secret", false)
	e.totpSecret.SetValue(ent.TotpSecret)
	e.cardNumber = newTextInput("Card Number", false)
	e.cardNumber.SetValue(ent.CardNumber)
	e.expiry = newTextInput("Expiry (MM/YY)", false)
	e.expiry.SetValue(ent.Expiry)
	e.cvv = newTextInput("CVV", false)
	e.cvv.SetValue(ent.CVV)

	e.notes = textarea.New()
	e.notes.SetValue(ent.CustomText)
	e.notes.SetWidth(50)
	e.notes.SetHeight(4)

	e.folderOptions = []string{"— (root)"}
	if s != nil {
		e.folderOptions = append(e.folderOptions, listFolders(s)...)
	}
	if folderID != 0 && s != nil {
		if f, _ := s.GetFolder(folderID); f != nil {
			for i, name := range e.folderOptions {
				if name == f.Name {
					e.folderIdx = i
					break
				}
			}
		}
	}

	e.focusOrder = e.buildFocusOrder()
	e.focusPos = 0
	e.focusTarget = eftTitle
	e.title.Focus()
	return e
}

func (m *Model) updateEditorKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if isTabForward(msg) {
		cmd := m.editor.focusNext()
		m.editor.saveDisabled = m.editorTitleInvalid()
		return m.openFolderSelectIfFocused(cmd)
	}
	if isTabBackward(msg) {
		cmd := m.editor.focusPrev()
		m.editor.saveDisabled = m.editorTitleInvalid()
		return m.openFolderSelectIfFocused(cmd)
	}

	switch key {
	case "esc":
		m.overlay = overlayNone
		return *m, nil
	case "down":
		switch m.editor.focusTarget {
		case eftNotes:
			cmd := m.editor.updateFocused(msg)
			return *m, cmd
		default:
			cmd := m.editor.focusNext()
			return m.openFolderSelectIfFocused(cmd)
		}
	case "up":
		switch m.editor.focusTarget {
		case eftNotes:
			cmd := m.editor.updateFocused(msg)
			return *m, cmd
		default:
			cmd := m.editor.focusPrev()
			return m.openFolderSelectIfFocused(cmd)
		}
	case "ctrl+g":
		if m.editor.entryType == TypeLogin {
			m.overlay = overlayPassgen
			m.passgen = newPassgenModel()
			m.passgen.updatePreview()
		}
	case "ctrl+b":
		if m.editor.entryType == TypeFile {
			home, _ := os.UserHomeDir()
			m.overlay = overlayFileBrowser
			m.fileBrowser = newFileBrowserModel(home)
		}
	case "enter":
		switch m.editor.focusTarget {
		case eftButtons:
			if m.editor.btnFocus == 0 && !m.editor.saveDisabled {
				m.saveEntry()
			} else if m.editor.btnFocus == 1 {
				m.overlay = overlayNone
			}
		case eftNotes:
			cmd := m.editor.updateFocused(msg)
			return *m, cmd
		default:
			cmd := m.editor.focusNext()
			return m.openFolderSelectIfFocused(cmd)
		}
	case "left":
		if m.editor.focusTarget == eftButtons && m.editor.btnFocus > 0 {
			m.editor.btnFocus--
		}
	case "right":
		if m.editor.focusTarget == eftButtons && m.editor.btnFocus < 1 {
			m.editor.btnFocus++
		}
	default:
		cmd := m.editor.updateFocused(msg)
		m.editor.saveDisabled = m.editorTitleInvalid()
		return *m, cmd
	}
	m.editor.saveDisabled = m.editorTitleInvalid()
	return *m, nil
}

// openFolderSelectIfFocused opens the folder-picker overlay whenever focus
// has landed on the (non-editable) folder field, since it has no inline
// editing of its own.
func (m *Model) openFolderSelectIfFocused(cmd tea.Cmd) (Model, tea.Cmd) {
	if m.editor.focusTarget == eftFolder {
		m.overlay = overlayFolderSelect
		m.folderSelect = folderSelectModel{cursor: m.editor.folderIdx}
		return *m, nil
	}
	return *m, cmd
}

func (m *Model) editorTitleInvalid() bool {
	return m.editor.title.Value() == "" ||
		validateCardFieldValues(m.editor.cardNumber.Value(), m.editor.expiry.Value(), m.editor.cvv.Value(), m.editor.entryType) != nil
}

type folderSelectModel struct {
	cursor int
}

func (m *Model) updateFolderSelectKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m.overlay = overlayEditor
	case "up", "k":
		if m.folderSelect.cursor > 0 {
			m.folderSelect.cursor--
		}
	case "down", "j":
		if m.folderSelect.cursor < len(m.editor.folderOptions)-1 {
			m.folderSelect.cursor++
		}
	case "enter":
		m.editor.folderIdx = m.folderSelect.cursor
		m.overlay = overlayEditor
	}
	return *m, nil
}

func (m Model) viewFolderSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Select Folder "))
	b.WriteString("\n\n")
	for i, opt := range m.editor.folderOptions {
		line := opt
		if i == m.folderSelect.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("↑↓ navigate  Enter select  Esc cancel"))
	return centerModal(b.String(), m.width, m.height, 35, 10, 55, 20, 0.4, 0.4)
}

func (m *Model) saveEntry() {
	title := strings.TrimSpace(m.editor.title.Value())
	if title == "" {
		return
	}
	if err := validateCardFieldValues(m.editor.cardNumber.Value(), m.editor.expiry.Value(), m.editor.cvv.Value(), m.editor.entryType); err != nil {
		return
	}

	var priorPassword string
	var priorHistory []PasswordHistory
	if m.editor.editingID != 0 && m.main.currentEnt != nil {
		priorPassword = m.main.currentEnt.Password
		priorHistory = m.main.currentEnt.History
	}

	ent := &Entry{
		Type:        string(m.editor.entryType),
		Title:       title,
		CustomText:  m.editor.notes.Value(),
		History:     priorHistory,
		Attachments: m.editor.pendingAttach,
	}

	switch m.editor.entryType {
	case TypeLogin:
		collectLoginFields(ent, m.editor, priorPassword)
	case TypeCard:
		num, exp, cvv := collectCardFields(m.editor)
		ent.CardNumber = num
		ent.Expiry = exp
		ent.CVV = cvv
	}

	var folderID int64
	if m.editor.folderIdx > 0 && m.editor.folderIdx < len(m.editor.folderOptions) {
		f, _ := m.store.GetFolderByName(m.editor.folderOptions[m.editor.folderIdx])
		if f != nil {
			folderID = f.ID
		}
	}

	if m.editor.editingID != 0 {
		if m.store.EntryExistsInFolderExcluding(folderID, title, m.editor.editingID) {
			m.overlay = overlayError
			m.modals = newErrorModal("Title already exists in this folder. Please change the title.")
			return
		}
		m.commitSave(m.editor.editingID, folderID, ent)
	} else {
		if m.store.EntryExistsInFolder(folderID, title) {
			m.overlay = overlayError
			m.modals = newErrorModal("Title already exists in this folder. Please change the title.")
			return
		}
		m.commitSaveNew(folderID, ent)
	}
}

func (m *Model) commitSaveNew(folderID int64, ent *Entry) {
	entryID, err := m.store.SaveEntry(folderID, ent)
	if err != nil {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to save entry: " + err.Error())
		return
	}
	failed := m.saveAttachments(entryID)
	m.main.refreshTree(m.store, m.main.search.Value())
	m.main.tree.selectRef(nodeRef{IsFolder: false, ID: entryID})
	m.loadEntry(entryID)
	if len(failed) > 0 {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to attach: " + strings.Join(failed, ", "))
		return
	}
	m.overlay = overlayNone
}

func (m *Model) commitSave(entryID int64, folderID int64, ent *Entry) {
	failed := m.saveAttachments(entryID)
	if err := m.store.UpdateEntryFull(entryID, folderID, ent); err != nil {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to save entry: " + err.Error())
		return
	}
	m.main.refreshTree(m.store, m.main.search.Value())
	m.main.tree.selectRef(nodeRef{IsFolder: false, ID: entryID})
	m.loadEntry(entryID)
	if len(failed) > 0 {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to attach: " + strings.Join(failed, ", "))
		return
	}
	m.overlay = overlayNone
}

// saveAttachments persists any newly-added attachments and returns the file
// names of any that could not be read or written, so the caller can surface
// the failure to the user instead of silently dropping the attachment.
func (m *Model) saveAttachments(entryID int64) []string {
	var failed []string
	for id, localPath := range m.editor.pendingPaths {
		data, err := os.ReadFile(localPath)
		if err != nil {
			failed = append(failed, filepath.Base(localPath))
			continue
		}
		var fileName string
		var size int64
		for _, att := range m.editor.pendingAttach {
			if att.ID == id {
				fileName = att.FileName
				size = att.Size
				break
			}
		}
		if fileName == "" {
			fileName = filepath.Base(localPath)
			size = int64(len(data))
		}
		if err := m.store.WriteAttachment(id, entryID, fileName, size, data); err != nil {
			failed = append(failed, fileName)
		}
	}
	return failed
}

func (m Model) viewEditor() string {
	title := " Add Entry "
	if m.editor.editingID != 0 {
		title = " Edit Entry "
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Title:"))
	b.WriteString(" ")
	b.WriteString(m.editor.title.View())
	b.WriteString("\n")

	folderLabel := m.editor.folderOptions[m.editor.folderIdx]
	b.WriteString(labelStyle.Render("Folder:"))
	b.WriteString(" ")
	if m.editor.focusTarget == eftFolder {
		b.WriteString(focusedBorderStyle.Render(" " + folderLabel + " ▾ "))
	} else {
		b.WriteString(folderLabel + " ▾")
	}
	b.WriteString("\n")

	switch m.editor.entryType {
	case TypeLogin:
		b.WriteString(m.renderLoginEditorFields())
	case TypeCard:
		b.WriteString(m.renderCardEditorFields())
	case TypeFile:
		b.WriteString(m.renderFileEditorFields())
	}

	b.WriteString(labelStyle.Render("Notes:"))
	b.WriteString("\n")
	b.WriteString(m.editor.notes.View())
	b.WriteString("\n\n")

	saveLabel := "Save"
	onButtons := m.editor.focusTarget == eftButtons
	saveFocused := onButtons && m.editor.btnFocus == 0
	cancelFocused := onButtons && m.editor.btnFocus == 1
	if m.editor.saveDisabled {
		saveLabel = dimStyle.Render("Save (disabled)")
	} else {
		b.WriteString(renderButton(saveLabel, saveFocused, true))
	}
	b.WriteString("  ")
	b.WriteString(renderButton("Cancel", cancelFocused, false))

	return centerModal(b.String(), m.width, m.height, 60, 25, 120, 50, 0.8, 0.85)
}

func (m Model) renderLoginEditorFields() string {
	var b strings.Builder
	b.WriteString(labelStyle.Render("Username:"))
	b.WriteString(" ")
	b.WriteString(m.editor.username.View())
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Password:"))
	b.WriteString(" ")
	b.WriteString(m.editor.password.View())
	b.WriteString(dimStyle.Render(" [Ctrl+G] generate"))
	b.WriteString("\n")
	if m.editor.strength != "" {
		b.WriteString(m.editor.strength)
		b.WriteString("\n")
	}
	b.WriteString(labelStyle.Render("Link:"))
	b.WriteString(" ")
	b.WriteString(m.editor.link.View())
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("TOTP Secret:"))
	b.WriteString(" ")
	b.WriteString(m.editor.totpSecret.View())
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderCardEditorFields() string {
	var b strings.Builder
	b.WriteString(labelStyle.Render("Card Number:"))
	b.WriteString(" ")
	b.WriteString(m.editor.cardNumber.View())
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Expiry:"))
	b.WriteString(" ")
	b.WriteString(m.editor.expiry.View())
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("CVV:"))
	b.WriteString(" ")
	b.WriteString(m.editor.cvv.View())
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderFileEditorFields() string {
	var b strings.Builder
	b.WriteString(dimStyle.Render("[Ctrl+B] Browse Filesystem"))
	b.WriteString("\n")
	for _, att := range m.editor.pendingAttach {
		b.WriteString("📎 " + att.FileName)
		if _, isNew := m.editor.pendingPaths[att.ID]; isNew {
			b.WriteString(successStyle.Render(" (New)"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func collectLoginFields(ent *Entry, e editorModel, priorPassword string) {
	ent.Username = e.username.Value()
	ent.Password = e.password.Value()
	ent.Link = e.link.Value()
	ent.TotpSecret = e.totpSecret.Value()
	if priorPassword != "" && priorPassword != ent.Password {
		ent.History = append(ent.History, PasswordHistory{
			Password: priorPassword,
			Date:     time.Now().Format("2006-01-02 15:04"),
		})
	}
}
