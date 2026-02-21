package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	uiFileBrowser      *tview.TreeView
	uiFileBrowserModal tview.Primitive
)

// addFileFields adds file-specific form fields to the editor.
func addFileFields(_ *Entry) {
	uiEditorLayout.AddItem(uiAttachFlex, 0, 0, false)

	uiEditorForm.AddButton("Browse Filesystem", func() {
		home, _ := os.UserHomeDir()
		openFileBrowser(home)
	})

	dropZone := tview.NewTextArea().
		SetLabel("Drag File Here").
		SetPlaceholder("Click here, then drop/paste a file path, then press Enter to attach").
		SetSize(5, 40)

	dropZone.SetBorder(true)
	dropZone.SetTitle(" Dropzone ")
	dropZone.SetTitleColor(tcell.ColorYellow)
	dropZone.SetBackgroundColor(tcell.ColorBlack)

	resetDropZone := func() {
		dropZone.SetText("", true)
		dropZone.SetLabel("Drag File Here")
		dropZone.SetPlaceholder("Click here, then drop/paste a file path, then press Enter to attach")
		dropZone.SetBorder(true)
		dropZone.SetTitle(" Dropzone ")
		dropZone.SetTitleColor(tcell.ColorYellow)
		dropZone.SetBackgroundColor(tcell.ColorBlack)
	}

	attachFromDropZone := func() {
		rawPath := dropZone.GetText()
		cleanPath := strings.Trim(rawPath, "\"' \n\r\t")
		if cleanPath == "" {
			return
		}

		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			cleanPath = strings.ReplaceAll(cleanPath, "\\ ", " ")
		}

		if strings.Contains(cleanPath, "\n") || strings.Contains(cleanPath, "\r") {
			return
		}

		fi, err := os.Stat(cleanPath)
		if err != nil || fi.IsDir() {
			return
		}

		id := fmt.Sprintf("%d", time.Now().UnixNano())
		att := &Attachment{Id: id, FileName: filepath.Base(cleanPath), Size: fi.Size()}
		uiPendingAttachments = append(uiPendingAttachments, att)
		uiPendingFilePaths[id] = cleanPath

		resetDropZone()
		refreshAttachmentList(TypeFile)
		uiApp.SetFocus(uiAttachList)
	}

	dropZone.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			attachFromDropZone()
			return nil
		}
		return event
	})

	uiEditorForm.AddFormItem(dropZone)

	resetDropZone()
	refreshAttachmentList(TypeFile)
}

// collectFileFields reads file form values into the entry (no-op beyond shared).
func collectFileFields(_ *Entry) {}

// renderFileView renders the file-type view pane content.
// File entries have no type-specific rows; attachments are rendered by the shared section.
func renderFileView() {}

// setupFileBrowser sets up the file browser modal for picking files.
func setupFileBrowser() {
	uiFileBrowser = tview.NewTreeView()
	uiFileBrowser.SetBorder(true).SetTitle(" Select File (Enter to Pick, Esc Cancel) ")
	uiFileBrowser.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			uiPages.SwitchToPage("editor")
		}
		return event
	})

	uiFileBrowserModal = newResponsiveModal(uiFileBrowser, 50, 20, 100, 40, 0.7, 0.75)
	uiPages.AddPage("filebrowser", uiFileBrowserModal, true, false)
}

// openFileBrowser opens the file browser at the given path.
func openFileBrowser(path string) {
	rootDir, _ := filepath.Abs(path)
	rootNode := tview.NewTreeNode(rootDir).SetColor(tcell.ColorYellow).SetReference(rootDir)
	uiFileBrowser.SetRoot(rootNode).SetCurrentNode(rootNode)
	addNodes(rootNode, rootDir)

	uiFileBrowser.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			return
		}
		path := ref.(string)
		fi, err := os.Stat(path)
		if err != nil {
			return
		}

		if fi.IsDir() {
			if len(node.GetChildren()) == 0 {
				addNodes(node, path)
			}
			node.SetExpanded(!node.IsExpanded())
		} else {
			id := fmt.Sprintf("%d", time.Now().UnixNano())
			att := &Attachment{Id: id, FileName: filepath.Base(path), Size: fi.Size()}
			uiPendingAttachments = append(uiPendingAttachments, att)
			uiPendingFilePaths[id] = path
			refreshAttachmentList(TypeFile)
			uiPages.SwitchToPage("editor")
		}
	})
	uiPages.SwitchToPage("filebrowser")
}

// addNodes adds directory entries as children of the given tree node.
func addNodes(target *tview.TreeNode, path string) {
	files, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		node := tview.NewTreeNode(f.Name()).SetReference(filepath.Join(path, f.Name()))
		if f.IsDir() {
			node.SetColor(tcell.ColorSkyblue)
		}
		target.AddChild(node)
	}
}

// refreshAttachmentList rebuilds the attachment list in the editor.
func refreshAttachmentList(t EntryType) {
	uiAttachList.Clear()
	size := 0

	if t == TypeFile || len(uiPendingAttachments) > 0 {
		if len(uiPendingAttachments) > 0 {
			size = 6
			for i, att := range uiPendingAttachments {
				label := att.FileName
				if _, isNew := uiPendingFilePaths[att.Id]; isNew {
					label += " [green](New)[-]"
				}
				idx := i
				uiAttachList.AddItem(label, "Press Enter to Remove", 0, func() {
					uiPendingAttachments = append(uiPendingAttachments[:idx], uiPendingAttachments[idx+1:]...)
					refreshAttachmentList(t)
				})
			}
		}
	}
	uiEditorLayout.ResizeItem(uiAttachFlex, size, 0)
}
