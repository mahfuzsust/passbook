package ui

import (
	"os"
	"path/filepath"

	"passbook/internal/config"
	"passbook/internal/crypto"
	"passbook/internal/pb"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/protobuf/proto"
)

var (
	uiApp   = tview.NewApplication()
	uiPages = tview.NewPages()

	uiCfg     config.AppConfig
	uiDataDir string

	uiMasterKey []byte
)

func NewApp(c config.AppConfig) (*AppHandle, error) {
	uiCfg = c
	uiDataDir = config.ExpandPath(uiCfg.DataDir)

	if err := os.MkdirAll(uiDataDir, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(getAttachmentDir(), 0700); err != nil {
		return nil, err
	}
	setupUI()
	uiPages.SwitchToPage("login")
	return &AppHandle{}, nil
}

type AppHandle struct{}

func getAttachmentDir() string {
	return filepath.Join(uiDataDir, "_attachments")
}

func unmarshalEntry(data []byte) (*pb.Entry, error) {
	e := &pb.Entry{}
	err := proto.Unmarshal(data, e)
	return e, err
}

func (a *AppHandle) Run() error {
	defer a.wipeKeys()
	return uiApp.SetRoot(uiPages, true).EnableMouse(true).Run()
}

func (a *AppHandle) wipeKeys() {
	crypto.WipeBytes(uiMasterKey)
	uiMasterKey = nil
	wipeTempMasterKey()
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
	setupEditor()
	setupChangePassword()
	setupFolderCreate()
	setupFolderRename()
	setupFolderDelete()
}
