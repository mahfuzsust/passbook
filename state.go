package main

import (
	"time"

	"github.com/rivo/tview"
)

var (
	app          = tview.NewApplication()
	pages        = tview.NewPages()
	masterKey    []byte
	dataDir      = "~/.passbook/data"
	lastActivity time.Time

	currentPath string
	currentEnt  Entry
	editingEnt  Entry

	pendingAttachments []Attachment
	pendingFilePaths   map[string]string
	pendingSaveData    []byte
	pendingPath        string
	lastGeneratedPass  string

	loginForm *tview.Form

	searchField *tview.InputField
	treeView    *tview.TreeView
	rightPages  *tview.Pages
	viewFlex    *tview.Flex

	viewTitle      *tview.TextView
	viewSubtitle   *tview.TextView
	viewPassword   *tview.TextView
	viewDetails    *tview.TextView
	viewTOTP       *tview.TextView
	viewTOTPBar    *tview.TextView
	viewCustom     *tview.TextView
	viewStatus     *tview.TextView
	attachmentList *tview.List
	showSensitive  bool

	editorForm       *tview.Form
	editorLayout     *tview.Flex
	attachFlex       *tview.Flex
	attachList       *tview.List
	createList       *tview.List
	fileBrowser      *tview.TreeView
	fileBrowserModal *tview.Flex

	settingsForm   *tview.Form
	deleteModal    *tview.Modal
	collisionModal *tview.Modal
	errorModal     *tview.Modal
	historyList    *tview.List
	passGenForm    *tview.Form
	passGenLayout  *tview.Flex
	passGenPreview *tview.TextView
)
