package ui

import (
	"fmt"
	"strings"

	"passbook/internal/store"
)

type nodeRef struct {
	IsFolder bool
	ID       int64
}

type treeItem struct {
	ref      nodeRef
	label    string
	depth    int
	expanded bool
}

type treeState struct {
	items      []treeItem
	flatCursor int
	filter     string
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

func (t *treeState) refreshTree(s *store.Store, filter string) {
	t.filter = filter
	t.items = nil

	folders, _ := s.ListFolders()
	for _, f := range folders {
		folderItem := treeItem{
			ref:      nodeRef{IsFolder: true, ID: f.ID},
			label:    fmt.Sprintf("📁 %s", f.Name),
			depth:    0,
			expanded: t.isFolderExpanded(f.ID),
		}
		entries := t.collectEntries(s, f.ID, filter, 1)
		if len(entries) > 0 || filter == "" {
			t.items = append(t.items, folderItem)
			if folderItem.expanded {
				t.items = append(t.items, entries...)
			}
		}
	}

	t.items = append(t.items, t.collectEntries(s, 0, filter, 0)...)

	if t.flatCursor >= len(t.items) {
		t.flatCursor = 0
	}
}

func (t *treeState) isFolderExpanded(id int64) bool {
	for _, item := range t.items {
		if item.ref.IsFolder && item.ref.ID == id {
			return item.expanded
		}
	}
	return true
}

func (t *treeState) collectEntries(s *store.Store, folderID int64, filter string, depth int) []treeItem {
	entries, err := s.ListEntries(folderID)
	if err != nil {
		return nil
	}
	var items []treeItem
	for _, e := range entries {
		if filter != "" && !strings.Contains(strings.ToLower(e.Title), strings.ToLower(filter)) {
			continue
		}
		icon := entryTypeIcon(e.EntryType)
		items = append(items, treeItem{
			ref:   nodeRef{IsFolder: false, ID: e.ID},
			label: fmt.Sprintf("%s %s", icon, e.Title),
			depth: depth,
		})
	}
	return items
}

func (t *treeState) render(height int) string {
	var lines []string
	lines = append(lines, titleStyle.Render(" Vault "))
	for i, item := range t.items {
		prefix := strings.Repeat("  ", item.depth)
		expand := " "
		if item.ref.IsFolder {
			if item.expanded {
				expand = "▼"
			} else {
				expand = "▶"
			}
		}
		line := prefix + expand + " " + item.label
		if i == t.flatCursor {
			line = selectedStyle.Render(line)
		} else if item.ref.IsFolder {
			line = skyStyle.Render(line)
		}
		lines = append(lines, line)
	}
	for len(lines) < height-1 {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func (t *treeState) moveUp() {
	if t.flatCursor > 0 {
		t.flatCursor--
	}
}

func (t *treeState) moveDown() {
	if t.flatCursor < len(t.items)-1 {
		t.flatCursor++
	}
}

func (t *treeState) toggleOrSelect() (nodeRef, bool) {
	if t.flatCursor >= len(t.items) {
		return nodeRef{}, false
	}
	item := t.items[t.flatCursor]
	if item.ref.IsFolder {
		t.items[t.flatCursor].expanded = !t.items[t.flatCursor].expanded
		return item.ref, true
	}
	return item.ref, false
}

func (t *treeState) currentRef() nodeRef {
	if t.flatCursor >= len(t.items) {
		return nodeRef{}
	}
	return t.items[t.flatCursor].ref
}

func (t *treeState) selectRef(ref nodeRef) {
	for i, item := range t.items {
		if item.ref == ref {
			t.flatCursor = i
			return
		}
	}
}

func listFolders(s *store.Store) []string {
	folders, err := s.ListFolders()
	if err != nil {
		return nil
	}
	names := make([]string, len(folders))
	for i, f := range folders {
		names[i] = f.Name
	}
	return names
}
