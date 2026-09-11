package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fileBrowserModel struct {
	rootPath string
	items    []fileBrowserItem
	cursor   int
	err      string
}

type fileBrowserItem struct {
	name  string
	path  string
	isDir bool
}

func newFileBrowserModel(path string) fileBrowserModel {
	var fb fileBrowserModel
	fb.refresh(path)
	return fb
}

func (fb *fileBrowserModel) refresh(path string) {
	fb.items = nil
	fb.err = ""
	fb.rootPath = path
	if parent := filepath.Dir(path); parent != path {
		fb.items = append(fb.items, fileBrowserItem{name: "..", path: parent, isDir: true})
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		fb.err = err.Error()
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fb.items = append(fb.items, fileBrowserItem{
			name:  e.Name(),
			path:  filepath.Join(path, e.Name()),
			isDir: e.IsDir(),
		})
	}
}

func (m *Model) updateFileBrowser(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m.overlay = overlayEditor
	case "up", "k":
		if m.fileBrowser.cursor > 0 {
			m.fileBrowser.cursor--
		}
	case "down", "j":
		if m.fileBrowser.cursor < len(m.fileBrowser.items)-1 {
			m.fileBrowser.cursor++
		}
	case "enter":
		if m.fileBrowser.cursor < len(m.fileBrowser.items) {
			item := m.fileBrowser.items[m.fileBrowser.cursor]
			if item.isDir {
				m.fileBrowser.refresh(item.path)
				m.fileBrowser.cursor = 0
			} else {
				id := fmt.Sprintf("%d", time.Now().UnixNano())
				fi, err := os.Stat(item.path)
				if err != nil {
					return *m, nil
				}
				att := Attachment{ID: id, FileName: filepath.Base(item.path), Size: fi.Size()}
				m.editor.pendingAttach = append(m.editor.pendingAttach, att)
				m.editor.pendingPaths[id] = item.path
				m.overlay = overlayEditor
			}
		}
	}
	return *m, nil
}

func (m Model) viewFileBrowser() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Select File "))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(m.fileBrowser.rootPath))
	b.WriteString("\n\n")
	if m.fileBrowser.err != "" {
		b.WriteString(errorStyle.Render("⚠ " + m.fileBrowser.err))
		b.WriteString("\n")
	} else if len(m.fileBrowser.items) == 0 {
		b.WriteString(dimStyle.Render("(empty directory)"))
		b.WriteString("\n")
	}
	for i, item := range m.fileBrowser.items {
		prefix := "📄 "
		if item.isDir {
			prefix = "📁 "
		}
		line := prefix + item.name
		if i == m.fileBrowser.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Enter to pick/expand, Esc to cancel"))
	return centerModal(b.String(), m.width, m.height, 50, 20, 100, 40, 0.7, 0.75)
}
