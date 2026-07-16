package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGalleryPanelDimsFallsBackForSmallTerminal(t *testing.T) {
	if _, _, _, show := (galleryModel{width: 50, height: 24}).panelDims(); show {
		t.Error("panelDims shows a preview below minPreviewWidth")
	}
	if _, _, _, show := (galleryModel{width: 80, height: 10}).panelDims(); show {
		t.Error("panelDims shows a preview below minPreviewHeight")
	}

	leftW, rightW, panelH, show := (galleryModel{width: 80, height: 24}).panelDims()
	if !show {
		t.Fatal("panelDims hides the preview at 80x24")
	}
	if leftW+rightW != 80 {
		t.Errorf("leftW+rightW = %d, want 80", leftW+rightW)
	}
	if panelH != 23 {
		t.Errorf("panelH = %d, want 23", panelH)
	}
}

// TestGalleryPreviewLoadsAndFollowsSelection drives the gallery through
// a real WindowSizeMsg and a cursor move, draining commands the way the
// Bubble Tea runtime would (see runCmd), to check the preview actually
// loads content and follows the list's selection.
func TestGalleryPreviewLoadsAndFollowsSelection(t *testing.T) {
	dir := t.TempDir()
	saveTinyEntry(t, dir, "first")
	saveTinyEntry(t, dir, "second")

	gallery, err := newGallery(dir, defaultStyles())
	if err != nil {
		t.Fatalf("newGallery: %v", err)
	}
	m := model{gallery: gallery}
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(model)
	m = runCmd(t, m, cmd)

	firstPath := m.gallery.preview.path
	if firstPath == "" {
		t.Fatal("preview did not select the initial entry")
	}
	if got := m.gallery.preview.view(); !strings.Contains(got, "first") {
		t.Errorf("preview view = %q, want it to mention %q", got, "first")
	}

	m = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.gallery.preview.path == firstPath {
		t.Fatal("preview did not follow the selection change")
	}
	if got := m.gallery.preview.view(); !strings.Contains(got, "second") {
		t.Errorf("preview view = %q, want it to mention %q", got, "second")
	}
}
