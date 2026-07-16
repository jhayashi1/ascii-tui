package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// theme is a flat set of named colors. Any value that lipgloss.Color
// accepts works (ANSI index, hex, name), so config can override them.
type theme struct {
	Accent string
	Border string
	Text   string
	Dim    string
	Error  string
}

// defaultTheme preserves the app's original pink-on-grey look.
func defaultTheme() theme {
	return theme{
		Accent: "212",
		Border: "240",
		Text:   "252",
		Dim:    "241",
		Error:  "203",
	}
}

// styles is the bundle of lipgloss styles derived once from a theme and
// plumbed into every sub-model, replacing package-level style vars.
type styles struct {
	theme      theme
	border     lipgloss.Style
	panelTitle lipgloss.Style
	help       lipgloss.Style
	status     lipgloss.Style
	selected   lipgloss.Style
	text       lipgloss.Style
	dim        lipgloss.Style
	accent     lipgloss.Style
	warning    lipgloss.Style
	prompt     lipgloss.Style
}

func newStyles(t theme) styles {
	return styles{
		theme:      t,
		border:     lipgloss.NewStyle().Foreground(lipgloss.Color(t.Border)),
		panelTitle: lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true),
		help:       lipgloss.NewStyle().Foreground(lipgloss.Color(t.Dim)),
		status:     lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)),
		selected:   lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true),
		text:       lipgloss.NewStyle().Foreground(lipgloss.Color(t.Text)),
		dim:        lipgloss.NewStyle().Foreground(lipgloss.Color(t.Dim)),
		accent:     lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)),
		warning:    lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Bold(true),
		prompt:     lipgloss.NewStyle().Padding(0, 2),
	}
}

// defaultStyles is a convenience for construction sites (and tests) that
// have no theme yet.
func defaultStyles() styles { return newStyles(defaultTheme()) }

// Rounded border runes; lipgloss v1.1.0 has no border-title API, so
// panels are assembled by hand with ANSI-aware width math.
const (
	borderTL = "╭"
	borderTR = "╮"
	borderBL = "╰"
	borderBR = "╯"
	borderH  = "─"
	borderV  = "│"
)

// renderPanel draws content inside a rounded-border box of the given
// outer width and height, splicing title into the top border line. The
// interior is clipped and padded so the result is exactly width x height.
func renderPanel(title, content string, width, height int, st styles) string {
	if width < 2 {
		width = 2
	}
	if height < 2 {
		height = 2
	}
	innerW := width - 2
	innerH := height - 2

	var b strings.Builder
	b.WriteString(st.topBorder(title, innerW))
	b.WriteByte('\n')

	lines := strings.Split(content, "\n")
	left := st.border.Render(borderV)
	right := st.border.Render(borderV)
	for i := 0; i < innerH; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		b.WriteString(left)
		b.WriteString(fitLine(line, innerW))
		b.WriteString(right)
		b.WriteByte('\n')
	}
	b.WriteString(st.border.Render(borderBL + strings.Repeat(borderH, innerW) + borderBR))
	return b.String()
}

// topBorder builds "╭─ title ───╮" clamped to innerW dashes between the
// corners, coloring the title with the accent style.
func (st styles) topBorder(title string, innerW int) string {
	if innerW < 0 {
		innerW = 0
	}
	if title == "" {
		return st.border.Render(borderTL + strings.Repeat(borderH, innerW) + borderTR)
	}
	const lead = 1
	label := " " + title + " "
	avail := innerW - lead
	if avail < 0 {
		avail = 0
	}
	label = truncateLabel(label, avail)
	trail := innerW - lead - lipgloss.Width(label)
	if trail < 0 {
		trail = 0
	}
	return st.border.Render(borderTL+strings.Repeat(borderH, lead)) +
		st.panelTitle.Render(label) +
		st.border.Render(strings.Repeat(borderH, trail)+borderTR)
}

// renderFooter draws the bottom help bar "╰ content ────╯" spanning the
// given width, coloring content with the help (dim) style.
func renderFooter(content string, width int, st styles) string {
	if width < 4 {
		width = 4
	}
	const prefix = borderBL + " "
	fill := width - 2 - lipgloss.Width(content) - 1 - 1
	if fill < 0 {
		content = truncateLabel(content, width-2-1-1)
		fill = width - 2 - lipgloss.Width(content) - 1 - 1
		if fill < 0 {
			fill = 0
		}
	}
	return st.border.Render(prefix) +
		st.help.Render(content) +
		st.border.Render(" "+strings.Repeat(borderH, fill)+borderBR)
}

// fitLine pads or truncates a (possibly ANSI-styled) line to exactly
// width display columns.
func fitLine(line string, width int) string {
	w := lipgloss.Width(line)
	if w == width {
		return line
	}
	if w < width {
		return line + strings.Repeat(" ", width-w)
	}
	return truncateLabel(line, width)
}

// truncateLabel trims a string to at most width display columns. It is
// ANSI-aware via lipgloss.Width but assumes styling does not span the
// cut point; panel titles and footers are plain text, so this holds.
func truncateLabel(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	for i := range runes {
		if lipgloss.Width(string(runes[:i+1])) > width {
			return string(runes[:i])
		}
	}
	return s
}
