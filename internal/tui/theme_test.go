package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderPanelDimensions(t *testing.T) {
	st := defaultStyles()
	panel := renderPanel("library", "line one\nline two", 24, 6, st)
	lines := strings.Split(panel, "\n")
	if len(lines) != 6 {
		t.Fatalf("panel has %d lines, want 6", len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w != 24 {
			t.Errorf("line %d width = %d, want 24 (%q)", i, w, line)
		}
	}
	if !strings.HasPrefix(lines[0], borderTL) {
		t.Errorf("top-left corner missing: %q", lines[0])
	}
	if !strings.HasSuffix(lines[len(lines)-1], borderBR) {
		t.Errorf("bottom-right corner missing: %q", lines[len(lines)-1])
	}
	if !strings.Contains(lines[0], "library") {
		t.Errorf("title not spliced into top border: %q", lines[0])
	}
}

func TestRenderPanelClampsOverflow(t *testing.T) {
	st := defaultStyles()
	long := strings.Repeat("x", 100)
	panel := renderPanel("t", long, 10, 3, st)
	for _, line := range strings.Split(panel, "\n") {
		if w := lipgloss.Width(line); w != 10 {
			t.Errorf("overflow line width = %d, want 10 (%q)", w, line)
		}
	}
}

func TestRenderPanelLongTitleTruncates(t *testing.T) {
	st := defaultStyles()
	panel := renderPanel(strings.Repeat("z", 50), "body", 12, 3, st)
	top := strings.Split(panel, "\n")[0]
	if w := lipgloss.Width(top); w != 12 {
		t.Errorf("top border width = %d, want 12 (%q)", w, top)
	}
}

func TestRenderFooterWidth(t *testing.T) {
	st := defaultStyles()
	footer := renderFooter("enter play · q quit", 40, st.help, st)
	if w := lipgloss.Width(footer); w != 40 {
		t.Errorf("footer width = %d, want 40 (%q)", w, footer)
	}
	if !strings.HasPrefix(footer, borderBL) {
		t.Errorf("footer missing bottom-left corner: %q", footer)
	}
	if !strings.HasSuffix(footer, borderBR) {
		t.Errorf("footer missing bottom-right corner: %q", footer)
	}
}

func TestRenderFooterTruncatesLongContent(t *testing.T) {
	st := defaultStyles()
	footer := renderFooter(strings.Repeat("word ", 40), 20, st.help, st)
	if w := lipgloss.Width(footer); w != 20 {
		t.Errorf("footer width = %d, want 20 (%q)", w, footer)
	}
}
