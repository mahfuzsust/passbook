package ui

import (
	"path/filepath"

	"passbook/internal/config"
	"passbook/internal/store"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiApp   = tview.NewApplication()
	uiPages = tview.NewPages()

	uiCfg     config.AppConfig
	uiDataDir string
	uiDBPath  string
	uiStore   *store.Store
)

func NewApp(c config.AppConfig) (*AppHandle, error) {
	uiCfg = c
	uiDataDir = config.ExpandPath(uiCfg.DataDir)
	uiDBPath = filepath.Join(uiDataDir, "passbook.db")

	setupUI()
	uiPages.SwitchToPage("login")
	return &AppHandle{}, nil
}

type AppHandle struct{}

func (a *AppHandle) Run() error {
	defer a.cleanup()
	return uiApp.SetRoot(uiPages, true).EnableMouse(true).Run()
}

func (a *AppHandle) cleanup() {
	if uiStore != nil {
		uiStore.Close()
	}
}

func (a *AppHandle) QueueUpdateDraw(f func()) {
	uiApp.QueueUpdateDraw(f)
}

func (a *AppHandle) DrawTOTP() { drawTOTP() }

func setupUI() {
	tview.Styles.ContrastBackgroundColor = colorUnfocusedBg
	tview.Styles.TitleColor = tcell.ColorLightSkyBlue

	setupLogin()
	setupPin()
	setupMainLayout()
	setupModals()
	setupQuickCopy()
	setupEditor()
	setupChangePassword()
	setupFolderCreate()
	setupFolderRename()
	setupFolderDelete()
}
