// Package ui implements The Elm Architecture (Model / Update / View):
//   - Model: the single Model struct below holds all application state.
//   - Update: Model.Update(msg) is the only place state transitions happen,
//     dispatching by message/screen/overlay to the update* handlers in this
//     package (e.g. updateMainKey, updateEditorKey, updatePinKey). Handlers
//     return the new Model plus an optional tea.Cmd for effects that must
//     run outside Update (timers, quitting, deferred messages).
//   - View: Model.View() and the view* functions render the current Model
//     to a string with no side effects; they all take Model by value so
//     they cannot mutate state.
package ui

import (
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"passbook/internal/config"
	"passbook/internal/store"
)

type screen int

const (
	screenLogin screen = iota
	screenPinSetup
	screenPinCreate
	screenTotpSetup
	screenPinVerify
	screenMain
)

type overlay int

const (
	overlayNone overlay = iota
	overlayCreateMenu
	overlayEditor
	overlayPassgen
	overlayFileBrowser
	overlayFolderSelect
	overlayQuickCopy
	overlayChangePwd
	overlayFolderCreate
	overlayFolderRename
	overlayFolderDelete
	overlayDelete
	overlayHistory
	overlayError
)

type tickMsg struct{}
type statusClearMsg struct{}
type clipboardClearMsg struct{ text string }
type redrawMsg struct{}

type Model struct {
	cfg     config.AppConfig
	dataDir string
	dbPath  string
	store   *store.Store

	width  int
	height int

	screen  screen
	overlay overlay

	freshInstall bool

	login  loginModel
	pin    pinModel
	main   mainModel
	editor editorModel

	createMenu   createMenuModel
	passgen      passgenModel
	fileBrowser  fileBrowserModel
	folderSelect folderSelectModel
	quickCopy    quickCopyModel
	changePwd    changePwdModel
	folder       folderModel
	modals       modalModel
}

func newModel(c config.AppConfig, dataDir, dbPath string) Model {
	fresh := !store.DBExists(dbPath)
	m := Model{
		cfg:          c,
		dataDir:      dataDir,
		dbPath:       dbPath,
		screen:       screenLogin,
		freshInstall: fresh,
	}
	m.login = newLoginModel(fresh)
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.login.password.Width = min(40, msg.Width-20)
		return m, nil

	case tea.KeyMsg:
		if m.overlay != overlayNone {
			return m.updateOverlayKey(msg)
		}
		switch m.screen {
		case screenLogin:
			return m.updateLoginKey(msg)
		case screenPinSetup, screenPinCreate, screenTotpSetup, screenPinVerify:
			return m.updatePinKey(msg)
		case screenMain:
			return m.updateMainKey(msg)
		}

	case tickMsg:
		if m.screen == screenMain && m.overlay == overlayNone {
			m.main.updateTOTP()
			if !m.main.viewStatusClearAt.IsZero() && time.Now().After(m.main.viewStatusClearAt) {
				m.main.viewStatus = ""
				m.main.viewStatusClearAt = time.Time{}
			}
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })

	case statusClearMsg:
		m.main.viewStatus = ""
		return m, nil

	case clipboardClearMsg:
		return m.handleClipboardClear(msg.text)

	case redrawMsg:
		return m, nil
	}

	if m.overlay == overlayEditor {
		cmd := m.editor.updateFocused(msg)
		return m, cmd
	}
	if m.overlay == overlayChangePwd {
		cmd := m.changePwd.updateFocused(msg)
		return m, cmd
	}
	if m.screen == screenLogin {
		var cmd tea.Cmd
		m.login.password, cmd = m.login.password.Update(msg)
		return m, cmd
	}
	if m.screen == screenPinCreate || m.screen == screenTotpSetup || m.screen == screenPinVerify {
		var cmd tea.Cmd
		m.pin, cmd = m.pin.updateInputs(msg)
		return m, cmd
	}
	if m.overlay == overlayFolderCreate || m.overlay == overlayFolderRename {
		var cmd tea.Cmd
		m.folder, cmd = m.folder.update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.overlay != overlayNone {
		base := m.renderBase()
		overlay := m.renderOverlay()
		return overlayOn(base, overlay, m.width, m.height)
	}

	switch m.screen {
	case screenLogin:
		return m.viewLogin()
	case screenPinSetup:
		return m.viewPinSetup()
	case screenPinCreate:
		return m.viewPinCreate()
	case screenTotpSetup:
		return m.viewTotpSetup()
	case screenPinVerify:
		return m.viewPinVerify()
	case screenMain:
		return m.viewMain()
	}
	return ""
}

func (m *Model) handleClipboardClear(text string) (Model, tea.Cmd) {
	curr, _ := clipboard.ReadAll()
	if curr == text {
		_ = clipboard.WriteAll("")
		m.main.viewStatus = successStyle.Render("Clipboard cleared")
		return *m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return statusClearMsg{} })
	}
	return *m, nil
}

func (m Model) renderBase() string {
	if m.screen == screenMain {
		return m.viewMain()
	}
	return ""
}

func overlayOn(base, overlay string, w, h int) string {
	if base == "" {
		return overlay
	}
	// Dim base behind modal by rendering overlay centered on full screen
	return overlay
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func newTextInput(placeholder string, password bool) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256
	ti.Width = 40
	if password {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '*'
	}
	return ti
}

func focusInput(ti *textinput.Model) tea.Cmd {
	ti.Focus()
	return textinput.Blink
}
