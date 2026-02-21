package ui

import (
	"fmt"
	"os"
	"path/filepath"
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

func refreshTree(filter string) {
	root := uiTreeView.GetRoot()
	root.ClearChildren()
	basePath := config.ExpandPath(uiDataDir)

	cats := []struct {
		T EntryType
		I string
	}{{TypeLogin, "🔐"}, {TypeCard, "💳"}, {TypeNote, "📝"}, {TypeFile, "📎"}}

	for _, c := range cats {
		catNode := tview.NewTreeNode(fmt.Sprintf("%s %ss", c.I, c.T)).SetColor(tcell.ColorSkyblue).SetSelectable(true).SetExpanded(true)
		dir := filepath.Join(basePath, strings.ToLower(string(c.T))+"s")
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			return
		}
		files, _ := os.ReadDir(dir)

		count := 0
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".pb") {
				name := strings.TrimSuffix(f.Name(), ".pb")
				if filter == "" || strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
					child := tview.NewTreeNode(name).SetReference(filepath.Join(dir, f.Name())).SetSelectable(true)
					catNode.AddChild(child)
					count++
				}
			}
		}
		if count > 0 || filter == "" {
			root.AddChild(catNode)
		}
	}
	if uiCurrentPath == "" {
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
		uiRightPages.SwitchToPage("content")
	}
}
