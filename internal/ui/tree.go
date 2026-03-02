package ui

import (
	"fmt"
	"strings"

	"passbook/internal/store"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type nodeRef struct {
	IsFolder bool
	ID       int64
}

func selectTreeNode(ref nodeRef) {
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
		if r := n.GetReference(); r != nil {
			if nr, ok := r.(nodeRef); ok && nr == ref {
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

func listFolders() []string {
	folders, err := uiStore.ListFolders()
	if err != nil {
		return nil
	}
	names := make([]string, len(folders))
	for i, f := range folders {
		names[i] = f.Name
	}
	return names
}

func listFolderInfos() []store.FolderInfo {
	folders, err := uiStore.ListFolders()
	if err != nil {
		return nil
	}
	return folders
}

func addItemNodes(parent *tview.TreeNode, folderID int64, filter string) int {
	entries, err := uiStore.ListEntries(folderID)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if filter != "" && !strings.Contains(strings.ToLower(e.Title), strings.ToLower(filter)) {
			continue
		}
		icon := entryTypeIcon(e.EntryType)
		child := tview.NewTreeNode(fmt.Sprintf("%s %s", icon, e.Title)).
			SetReference(nodeRef{IsFolder: false, ID: e.ID}).
			SetSelectable(true)
		parent.AddChild(child)
		count++
	}
	return count
}

func refreshTree(filter string) {
	root := uiTreeView.GetRoot()
	root.ClearChildren()

	folders := listFolderInfos()

	for _, f := range folders {
		folderNode := tview.NewTreeNode(fmt.Sprintf("📁 %s", f.Name)).
			SetReference(nodeRef{IsFolder: true, ID: f.ID}).
			SetColor(tcell.ColorSkyblue).
			SetSelectable(true).
			SetExpanded(true)

		count := addItemNodes(folderNode, f.ID, filter)
		if count > 0 || filter == "" {
			root.AddChild(folderNode)
		}
	}

	addItemNodes(root, 0, filter)

	if uiCurrentEntryID == 0 {
		uiRightPages.SetTitle(" Keybindings ")
		uiRightPages.SwitchToPage("empty")
	}
}

func loadEntry(id int64) {
	ent, err := uiStore.LoadEntry(id)
	if err != nil {
		return
	}

	uiCurrentEnt = ent
	uiCurrentEntryID = id
	uiShowSensitive = false
	updateViewPane()
	uiRightPages.SetTitle(" " + entryTypeIcon(ent.Type) + " " + ent.Title + " ")
	uiRightPages.SwitchToPage("content")
}
