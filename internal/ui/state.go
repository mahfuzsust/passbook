package ui

import (
	"passbook/internal/config"
	"passbook/internal/crypto"

	"github.com/rivo/tview"
)

var (
	uiApp   = tview.NewApplication()
	uiPages = tview.NewPages()

	uiCfg     config.AppConfig
	uiDataDir string
	uiKDF     crypto.KDFParams

	uiMasterKey []byte

	uiCurrentPath string
	uiCurrentEnt  *Entry
	uiEditingEnt  *Entry

	uiPendingAttachments []*Attachment
	uiPendingFilePaths   map[string]string
	uiPendingSaveData    []byte
	uiPendingPath        string
	uiLastGeneratedPass  string

	uiLoginForm  *tview.Form
	uiLoginModal tview.Primitive

	uiSearchField *tview.InputField
	uiTreeView    *tview.TreeView
	uiRightPages  *tview.Pages
	uiViewFlex    *tview.Flex

	uiViewTitle      *tview.TextView
	uiViewSubtitle   *tview.TextView
	uiViewPassword   *tview.TextView
	uiViewDetails    *tview.TextView
	uiViewTOTP       *tview.TextView
	uiViewTOTPBar    *tview.TextView
	uiViewCustom     *tview.TextView
	uiViewStatus     *tview.TextView
	uiAttachmentList *tview.List
	uiShowSensitive  bool

	uiEditorForm       *tview.Form
	uiEditorLayout     *tview.Flex
	uiAttachFlex       *tview.Flex
	uiAttachList       *tview.List
	uiCreateList       *tview.List
	uiFileBrowser      *tview.TreeView
	uiFileBrowserModal tview.Primitive

	uiDeleteModal    *tview.Modal
	uiCollisionModal *tview.Modal
	uiErrorModal     *tview.Modal
	uiHistoryList    *tview.List
	uiPassGenForm    *tview.Form
	uiPassGenLayout  *tview.Flex
	uiPassGenPreview *tview.TextView
)
