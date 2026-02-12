package main

import (
	"github.com/rivo/tview"
	"time"
)

// --- Global Application State ---

var (
	// Core
	app          = tview.NewApplication()
	pages        = tview.NewPages()
	masterKey    []byte
	dataDir      = "~/.passbook/data"
	lastActivity time.Time

	// Current Selection State
	currentPath string
	currentEnt  Entry
	editingEnt  Entry

	// Temporary State (for Editor/Saver)
	pendingAttachments []Attachment
	pendingFilePaths   map[string]string // Maps Attachment.ID -> Local File Path
	pendingSaveData    []byte
	pendingPath        string
	lastGeneratedPass  string

	// UI Components - Login
	loginForm *tview.Form

	// UI Components - Main Layout
	searchField *tview.InputField
	treeView    *tview.TreeView
	rightPages  *tview.Pages
	viewFlex    *tview.Flex

	// UI Components - Viewer
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

	// UI Components - Editor
	editorForm       *tview.Form
	editorLayout     *tview.Flex
	attachFlex       *tview.Flex
	attachList       *tview.List
	createList       *tview.List
	fileBrowser      *tview.TreeView
	fileBrowserModal *tview.Flex

	// UI Components - Modals & Tools
	settingsForm   *tview.Form
	deleteModal    *tview.Modal
	collisionModal *tview.Modal
	errorModal     *tview.Modal
	historyList    *tview.List
	passGenForm    *tview.Form
	passGenLayout  *tview.Flex
	passGenPreview *tview.TextView
)
