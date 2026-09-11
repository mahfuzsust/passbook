package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileEditorModel(t *testing.T) {
	ent := NewEntry(TypeFile)
	e := newEditorModel(ent, 0, 0, nil)
	if e.entryType != TypeFile {
		t.Fatalf("expected file entry type")
	}
	if e.pendingPaths == nil {
		t.Fatalf("expected pending paths map")
	}
}

func TestFileEditorFocusOrderHasNoDropzone(t *testing.T) {
	ent := NewEntry(TypeFile)
	e := newEditorModel(ent, 0, 0, nil)
	for _, target := range e.focusOrder {
		if target != eftTitle && target != eftFolder && target != eftNotes && target != eftButtons {
			t.Fatalf("unexpected focus target %v in file entry (only ctrl+b file select is supported)", target)
		}
	}
}

func TestFileEditorFieldsMentionOnlyCtrlB(t *testing.T) {
	m := Model{editor: newEditorModel(NewEntry(TypeFile), 0, 0, nil)}
	got := m.renderFileEditorFields()
	if !strings.Contains(got, "Ctrl+B") {
		t.Fatalf("expected a Ctrl+B hint, got %q", got)
	}
	if strings.Contains(got, "Attach file path") {
		t.Fatalf("did not expect a dropzone input, got %q", got)
	}
}


func TestFileBrowserListsFilesAndDirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	fb := newFileBrowserModel(dir)
	if fb.err != "" {
		t.Fatalf("did not expect an error, got %q", fb.err)
	}
	var sawFile, sawDir bool
	for _, it := range fb.items {
		if it.name == "a.txt" && !it.isDir {
			sawFile = true
		}
		if it.name == "sub" && it.isDir {
			sawDir = true
		}
	}
	if !sawFile || !sawDir {
		t.Fatalf("expected to see file and directory entries, got %+v", fb.items)
	}
}

func TestFileBrowserHasParentEntry(t *testing.T) {
	dir := t.TempDir()
	fb := newFileBrowserModel(dir)
	if len(fb.items) == 0 || fb.items[0].name != ".." {
		t.Fatalf("expected first entry to be '..' for parent navigation, got %+v", fb.items)
	}
}

func TestFileBrowserSurfacesReadError(t *testing.T) {
	fb := newFileBrowserModel(filepath.Join(t.TempDir(), "does-not-exist"))
	if fb.err == "" {
		t.Fatalf("expected an error to be surfaced for an unreadable directory")
	}
}

func TestUpdateFileBrowserSelectsFileAndAttaches(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(fpath, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	ent := NewEntry(TypeFile)
	m := &Model{}
	m.editor = newEditorModel(ent, 0, 0, nil)
	m.overlay = overlayFileBrowser
	m.fileBrowser = newFileBrowserModel(dir)

	// cursor 0 is the ".." entry; move to the file.
	nm, _ := m.updateFileBrowser("down")
	*m = nm
	nm, _ = m.updateFileBrowser("enter")
	*m = nm

	if len(m.editor.pendingAttach) != 1 {
		t.Fatalf("expected 1 pending attachment, got %d", len(m.editor.pendingAttach))
	}
	if m.overlay != overlayEditor {
		t.Fatalf("expected editor overlay after picking a file, got %v", m.overlay)
	}
}

func TestUpdateFileBrowserDescendsIntoDirAndUpdatesRootPath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	m := &Model{}
	m.overlay = overlayFileBrowser
	m.fileBrowser = newFileBrowserModel(dir)

	// find the "sub" entry and select it
	for i, it := range m.fileBrowser.items {
		if it.name == "sub" {
			m.fileBrowser.cursor = i
		}
	}
	nm, _ := m.updateFileBrowser("enter")
	*m = nm
	if m.fileBrowser.rootPath != sub {
		t.Fatalf("expected rootPath to update to %q, got %q", sub, m.fileBrowser.rootPath)
	}
}

func TestUpdateFileBrowserEscReturnsToEditor(t *testing.T) {
	m := &Model{overlay: overlayFileBrowser}
	nm, _ := m.updateFileBrowser("esc")
	if nm.overlay != overlayEditor {
		t.Fatalf("expected esc to return to editor overlay")
	}
}

func TestSaveAttachmentsSurfacesReadFailure(t *testing.T) {
	ent := NewEntry(TypeFile)
	m := &Model{}
	m.editor = newEditorModel(ent, 0, 0, nil)
	m.editor.pendingPaths["missing-id"] = filepath.Join(t.TempDir(), "does-not-exist.bin")

	failed := m.saveAttachments(1)
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed attachment, got %d: %+v", len(failed), failed)
	}
}

