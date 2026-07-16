package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// fixtureGifDir builds a directory with gifs at two levels plus files
// the completer should ignore.
func fixtureGifDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "clips"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"party.gif", "cat.GIF", "notes.txt", ".hidden.gif", "clips/nested.gif"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func suggestionNames(p pathInput) []string {
	names := make([]string, len(p.suggestions))
	for i, s := range p.suggestions {
		names[i] = s.name
	}
	return names
}

func setPickerValue(p *pathInput, value string) {
	p.input.SetValue(value)
	p.refresh()
}

func TestPathInputListsGifsRecursively(t *testing.T) {
	dir := fixtureGifDir(t)
	p := newPathInput(defaultStyles())
	setPickerValue(&p, dir+string(os.PathSeparator))

	want := []string{"cat.GIF", "party.gif", filepath.Join("clips", "nested.gif")}
	got := suggestionNames(p)
	if len(got) != len(want) {
		t.Fatalf("suggestions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("suggestion[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPathInputFuzzyFilters(t *testing.T) {
	dir := fixtureGifDir(t)
	p := newPathInput(defaultStyles())
	setPickerValue(&p, filepath.Join(dir, "pty"))

	got := suggestionNames(p)
	if len(got) != 1 || got[0] != "party.gif" {
		t.Fatalf("fuzzy suggestions for \"pty\" = %v, want [party.gif]", got)
	}
	if len(p.suggestions[0].matches) == 0 {
		t.Error("fuzzy match has no highlighted indexes")
	}
}

func TestPathInputFuzzyMatchesNestedGifs(t *testing.T) {
	dir := fixtureGifDir(t)
	p := newPathInput(defaultStyles())
	setPickerValue(&p, filepath.Join(dir, "nested"))

	got := suggestionNames(p)
	if want := filepath.Join("clips", "nested.gif"); len(got) != 1 || got[0] != want {
		t.Fatalf("fuzzy suggestions for \"nested\" = %v, want [%s]", got, want)
	}
}

func TestPathInputShowsHiddenWhenAskedFor(t *testing.T) {
	dir := fixtureGifDir(t)
	p := newPathInput(defaultStyles())
	setPickerValue(&p, filepath.Join(dir, ".h"))

	got := suggestionNames(p)
	if len(got) != 1 || got[0] != ".hidden.gif" {
		t.Fatalf("suggestions for \".h\" = %v, want [.hidden.gif]", got)
	}
}

func TestPathInputCompleteFillsNestedPath(t *testing.T) {
	dir := fixtureGifDir(t)
	p := newPathInput(defaultStyles())
	setPickerValue(&p, filepath.Join(dir, "cl"))

	p.complete()
	want := filepath.Join(dir, "clips", "nested.gif")
	if got := p.input.Value(); got != want {
		t.Fatalf("value after complete = %q, want %q", got, want)
	}
}

func TestPathInputAcceptGif(t *testing.T) {
	dir := fixtureGifDir(t)
	p := newPathInput(defaultStyles())
	setPickerValue(&p, filepath.Join(dir, "party"))

	path, ok := p.accept()
	if !ok {
		t.Fatal("accept on gif suggestion returned ok=false")
	}
	if want := filepath.Join(dir, "party.gif"); path != want {
		t.Errorf("accepted path = %q, want %q", path, want)
	}
}

func TestPathInputTildeCompletion(t *testing.T) {
	home := fixtureGifDir(t)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	p := newPathInput(defaultStyles())
	setPickerValue(&p, "~")

	got := suggestionNames(p)
	if len(got) != 3 {
		t.Fatalf("suggestions under ~ = %v, want 3 entries", got)
	}

	p.complete()
	want := "~" + string(os.PathSeparator) + "cat.GIF"
	if got := p.input.Value(); got != want {
		t.Errorf("value after completing under ~ = %q, want %q", got, want)
	}
}

func TestPathInputAcceptExpandsTypedTilde(t *testing.T) {
	home := t.TempDir() // empty: no suggestions, the literal text is used
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	p := newPathInput(defaultStyles())
	setPickerValue(&p, "~/party.gif")

	path, ok := p.accept()
	if !ok {
		t.Fatal("accept on typed path returned ok=false")
	}
	if want := filepath.Join(home, "party.gif"); path != want {
		t.Errorf("accepted path = %q, want %q", path, want)
	}
}

func TestPathInputSelectionWraps(t *testing.T) {
	dir := fixtureGifDir(t)
	p := newPathInput(defaultStyles())
	setPickerValue(&p, dir+string(os.PathSeparator))

	p.moveSelection(-1)
	if p.sel != len(p.suggestions)-1 {
		t.Errorf("sel after up from top = %d, want %d", p.sel, len(p.suggestions)-1)
	}
	p.moveSelection(1)
	if p.sel != 0 {
		t.Errorf("sel after wrap down = %d, want 0", p.sel)
	}
}

func TestPathInputDepthCap(t *testing.T) {
	dir := t.TempDir()
	deep := dir
	for range maxWalkDepth + 1 {
		deep = filepath.Join(deep, "d")
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "far.gif"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	p := newPathInput(defaultStyles())
	setPickerValue(&p, dir+string(os.PathSeparator))
	if got := suggestionNames(p); len(got) != 0 {
		t.Errorf("gif beyond depth cap suggested: %v", got)
	}
}
