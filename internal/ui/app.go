package ui

import (
	"path/filepath"

	"passbook/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type AppHandle struct {
	program *tea.Program
	model   *Model
}

func NewApp(c config.AppConfig) (*AppHandle, error) {
	dataDir := config.ExpandPath(c.DataDir)
	dbPath := filepath.Join(dataDir, "passbook.db")
	m := newModel(c, dataDir, dbPath)
	p := tea.NewProgram(&m, tea.WithAltScreen())
	return &AppHandle{program: p, model: &m}, nil
}

func (a *AppHandle) Run() error {
	defer a.cleanup()
	_, err := a.program.Run()
	return err
}

func (a *AppHandle) cleanup() {
	if a.model != nil && a.model.store != nil {
		a.model.store.Close()
	}
}

func (a *AppHandle) QueueUpdateDraw(f func()) {
	a.program.Send(redrawMsg{})
	if f != nil {
		f()
	}
}

func (a *AppHandle) DrawTOTP() {
	a.program.Send(tickMsg{})
}
