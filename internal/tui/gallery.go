package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jhayashi1/ascii-tui/internal/library"
)

type entryItem struct{ library.Entry }

func (e entryItem) Title() string       { return e.Name }
func (e entryItem) Description() string { return e.Path }
func (e entryItem) FilterValue() string { return e.Name }

// inputMode says what the gallery is collecting text for: a gif path
// (through the completing picker) or a new entry name (plain input).
type inputMode int

const (
	inputNone inputMode = iota
	inputAddGIF
	inputRename
)

type galleryModel struct {
	dir        string
	list       list.Model
	picker     pathInput
	input      textinput.Model
	mode       inputMode
	renamePath string
	status     string
	st         styles
	keys       galleryKeyMap
	width      int
	height     int
}

func newGallery(dir string, st styles) (galleryModel, error) {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "ascii-tui library"
	l.SetShowHelp(false)
	l.SetStatusBarItemName("animation", "animations")
	l.DisableQuitKeybindings()

	g := galleryModel{dir: dir, list: l, picker: newPathInput(st), input: textinput.New(), st: st, keys: newGalleryKeyMap()}
	if err := g.reload(); err != nil {
		return g, err
	}
	return g, nil
}

func (g *galleryModel) reload() error {
	entries, err := library.List(g.dir)
	if err != nil {
		return err
	}
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = entryItem{e}
	}
	g.list.SetItems(items)
	return nil
}

func (g *galleryModel) entries() []library.Entry {
	items := g.list.Items()
	entries := make([]library.Entry, len(items))
	for i, item := range items {
		entries[i] = item.(entryItem).Entry
	}
	return entries
}

func (g *galleryModel) setSize(width, height int) {
	g.width, g.height = width, height
	g.picker.setWidth(max(10, width-8))
	g.input.Width = max(10, width-8)
	g.layout()
}

// layout sizes the list, reserving room for the completion rows while
// the path prompt is open so the view height stays constant.
func (g *galleryModel) layout() {
	reserved := 3
	if g.mode == inputAddGIF {
		reserved += maxVisibleSuggestions
	}
	g.list.SetSize(g.width, max(0, g.height-reserved))
}

func (g *galleryModel) stopTyping() {
	g.mode = inputNone
	g.picker.blur()
	g.input.Blur()
	g.layout()
}

func (g galleryModel) update(msg tea.Msg) (galleryModel, tea.Cmd) {
	if g.mode != inputNone {
		return g.updateTyping(msg)
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok && g.list.FilterState() != list.Filtering {
		switch {
		case key.Matches(keyMsg, g.keys.Quit):
			return g, tea.Quit
		case key.Matches(keyMsg, g.keys.Play):
			// GlobalIndex maps the selection back into the full item
			// slice even when a filter is applied.
			if index := g.list.GlobalIndex(); index >= 0 && index < len(g.list.Items()) {
				entries := g.entries()
				return g, func() tea.Msg { return playEntryMsg{entries: entries, index: index} }
			}
		case key.Matches(keyMsg, g.keys.Add):
			g.mode = inputAddGIF
			g.status = ""
			g.layout()
			return g, g.picker.focus()
		case key.Matches(keyMsg, g.keys.Rename):
			if item, ok := g.list.SelectedItem().(entryItem); ok {
				g.mode = inputRename
				g.renamePath = item.Path
				g.status = ""
				g.input.SetValue(item.Name)
				g.input.CursorEnd()
				g.layout()
				return g, g.input.Focus()
			}
			return g, nil
		case key.Matches(keyMsg, g.keys.Delete):
			if item, ok := g.list.SelectedItem().(entryItem); ok {
				if err := os.Remove(item.Path); err != nil {
					g.status = fmt.Sprintf("delete failed: %v", err)
				} else if err := g.reload(); err != nil {
					g.status = err.Error()
				}
			}
			return g, nil
		}
	}

	var cmd tea.Cmd
	g.list, cmd = g.list.Update(msg)
	return g, cmd
}

func (g galleryModel) updateTyping(msg tea.Msg) (galleryModel, tea.Cmd) {
	if g.mode == inputRename {
		return g.updateRenaming(msg)
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			path, ok := g.picker.accept()
			if !ok {
				return g, nil
			}
			g.stopTyping()
			return g, func() tea.Msg { return startRenderMsg{gifPath: path} }
		case "tab":
			g.picker.complete()
			return g, nil
		case "down", "ctrl+n":
			g.picker.moveSelection(1)
			return g, nil
		case "up", "ctrl+p", "shift+tab":
			g.picker.moveSelection(-1)
			return g, nil
		case "esc":
			g.stopTyping()
			return g, nil
		}
	}

	var cmd tea.Cmd
	g.picker, cmd = g.picker.update(msg)
	return g, cmd
}

func (g galleryModel) updateRenaming(msg tea.Msg) (galleryModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			name := strings.TrimSpace(g.input.Value())
			g.stopTyping()
			if name == "" {
				return g, nil
			}
			return g.commitRename(name)
		case "esc":
			g.stopTyping()
			return g, nil
		}
	}

	var cmd tea.Cmd
	g.input, cmd = g.input.Update(msg)
	return g, cmd
}

// commitRename renames the selected entry and keeps it selected, since
// entries are listed by name and the rename can reorder them.
func (g galleryModel) commitRename(newName string) (galleryModel, tea.Cmd) {
	newPath, err := library.Rename(g.renamePath, newName)
	if err != nil {
		g.status = fmt.Sprintf("rename failed: %v", err)
		return g, nil
	}
	if err := g.reload(); err != nil {
		g.status = err.Error()
		return g, nil
	}
	for i, item := range g.list.Items() {
		if item.(entryItem).Path == newPath {
			g.list.Select(i)
			break
		}
	}
	return g, nil
}

func (g galleryModel) view() string {
	var b strings.Builder
	b.WriteString(g.list.View())
	b.WriteByte('\n')
	switch g.mode {
	case inputAddGIF:
		b.WriteString(g.picker.view())
		b.WriteString(g.st.help.Render("[enter] render  [tab] complete  [↑/↓] select  [esc] cancel"))
		return b.String()
	case inputRename:
		b.WriteString(g.st.prompt.Render("rename to: "+g.input.View()) + "\n")
		b.WriteString(g.st.help.Render("[enter] rename  [esc] cancel"))
		return b.String()
	}
	if g.status != "" {
		b.WriteString(g.st.status.Render(g.status) + "\n")
	}
	b.WriteString(g.st.help.Render("[enter] play  [a] add gif  [r] rename  [d] delete  [/] filter  [q] quit"))
	return b.String()
}
