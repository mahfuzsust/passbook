package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"passbook/internal/config"
	"passbook/internal/crypto"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func selectTreePath(path string) {
	if uiTreeView == nil {
		return
	}
	root := uiTreeView.GetRoot()
	if root == nil {
		return
	}

	var dfs func(n *tview.TreeNode) *tview.TreeNode
	dfs = func(n *tview.TreeNode) *tview.TreeNode {
		if n == nil {
			return nil
		}
		if ref := n.GetReference(); ref != nil {
			if s, ok := ref.(string); ok && s == path {
				return n
			}
		}
		for _, ch := range n.GetChildren() {
			if found := dfs(ch); found != nil {
				return found
			}
		}
		return nil
	}

	if node := dfs(root); node != nil {
		uiTreeView.SetCurrentNode(node)
		if uiApp != nil {
			uiApp.SetFocus(uiTreeView)
		}
	}
}

func entryTypeIcon(t string) string {
	switch EntryType(t) {
	case TypeLogin:
		return "🔐"
	case TypeCard:
		return "💳"
	case TypeNote:
		return "📝"
	case TypeFile:
		return "📎"
	default:
		return "📄"
	}
}

func readEntryType(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	dec, err := crypto.Decrypt(uiMasterKey, data)
	if err != nil {
		return ""
	}
	ent, err := unmarshalEntry(dec)
	if err != nil {
		return ""
	}
	return ent.Type
}

func listFolders() []string {
	basePath := config.ExpandPath(uiDataDir)
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}
	var folders []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(e.Name(), "_") {
			folders = append(folders, e.Name())
		}
	}
	sort.Strings(folders)
	return folders
}

func addItemNodes(parent *tview.TreeNode, dir string, filter string) int {
	files, _ := os.ReadDir(dir)
	count := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".pb") {
			continue
		}
		name := strings.TrimSuffix(f.Name(), ".pb")
		if filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
			continue
		}
		fullPath := filepath.Join(dir, f.Name())
		icon := entryTypeIcon(readEntryType(fullPath))
		child := tview.NewTreeNode(fmt.Sprintf("%s %s", icon, name)).
			SetReference(fullPath).
			SetSelectable(true)
		parent.AddChild(child)
		count++
	}
	return count
}

func refreshTree(filter string) {
	root := uiTreeView.GetRoot()
	root.ClearChildren()
	basePath := config.ExpandPath(uiDataDir)

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return
	}

	var folders []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		if e.IsDir() {
			folders = append(folders, e)
		}
	}

	for _, d := range folders {
		folderPath := filepath.Join(basePath, d.Name())
		folderNode := tview.NewTreeNode(fmt.Sprintf("📁 %s", d.Name())).
			SetReference(folderPath).
			SetColor(tcell.ColorSkyblue).
			SetSelectable(true).
			SetExpanded(true)

		count := addItemNodes(folderNode, folderPath, filter)
		if count > 0 || filter == "" {
			root.AddChild(folderNode)
		}
	}

	addItemNodes(root, basePath, filter)

	if uiCurrentPath == "" {
		uiRightPages.SetTitle(" Keybindings ")
		uiRightPages.SwitchToPage("empty")
	}
}

func loadEntry(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	decrypted, err := crypto.Decrypt(uiMasterKey, data)
	if err != nil {
		return
	}

	ent, err := unmarshalEntry(decrypted)
	if err == nil {
		uiCurrentEnt = ent
		uiCurrentPath = path
		uiShowSensitive = false
		updateViewPane()
		uiRightPages.SetTitle(" " + entryTypeIcon(ent.Type) + " " + ent.Title + " ")
		uiRightPages.SwitchToPage("content")
	}
}
