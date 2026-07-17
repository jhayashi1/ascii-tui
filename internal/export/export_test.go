package export

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jhayashi1/ascii-tui/internal/frames"
)

// cellHasColor reports whether any pixel of the character cell at
// (col, row) decodes to the given color.
func cellHasColor(img *image.Paletted, col, row int, want color.RGBA) bool {
	for y := row * cellH; y < (row+1)*cellH; y++ {
		for x := col * cellW; x < (col+1)*cellW; x++ {
			if img.At(x, y) == want {
				return true
			}
		}
	}
	return false
}

func TestGIFRoundtrip(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	anim := &frames.Animation{
		Width:  3,
		Height: 2,
		Frames: []string{
			"\x1b[38;2;255;0;0m@#\x1b[0m.\n a ",
			"...\n...",
		},
		Delays: []time.Duration{100 * time.Millisecond, 4 * time.Millisecond},
	}

	var buf bytes.Buffer
	if err := GIF(&buf, anim); err != nil {
		t.Fatalf("GIF: %v", err)
	}
	decoded, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}

	if len(decoded.Image) != 2 {
		t.Fatalf("frame count = %d, want 2", len(decoded.Image))
	}
	if got, want := decoded.Image[0].Bounds().Max, image.Pt(3*cellW, 2*cellH); got != want {
		t.Errorf("frame size = %v, want %v", got, want)
	}
	if decoded.Delay[0] != 10 {
		t.Errorf("delay 0 = %d, want 10 (100ms in centiseconds)", decoded.Delay[0])
	}
	if decoded.Delay[1] != 1 {
		t.Errorf("delay 1 = %d, want 1 (clamped minimum)", decoded.Delay[1])
	}
	if decoded.LoopCount != 0 {
		t.Errorf("loop count = %d, want 0 (loop forever)", decoded.LoopCount)
	}

	first := decoded.Image[0]
	if !cellHasColor(first, 0, 0, red) {
		t.Error("cell (0,0) has no red pixel for the colored '@'")
	}
	if !cellHasColor(first, 2, 0, defaultFg) {
		t.Error("cell (2,0) after reset has no default-colored pixel")
	}
	if cellHasColor(first, 0, 1, defaultFg) || cellHasColor(first, 0, 1, red) {
		t.Error("space cell (0,1) has glyph pixels, want background only")
	}
	if !cellHasColor(first, 1, 1, defaultFg) {
		t.Error("uncolored 'a' cell has no default-colored pixel")
	}
}

func TestGIFQuantizesPast256Colors(t *testing.T) {
	// 300 distinct colors on one row, with green repeated enough that
	// quantization must keep it.
	green := color.RGBA{G: 255, A: 255}
	var b strings.Builder
	for i := range 300 {
		fmt.Fprintf(&b, "\x1b[38;2;%d;%d;200m@", i%256, i/256)
	}
	for range 20 {
		b.WriteString("\x1b[38;2;0;255;0m@")
	}
	anim := &frames.Animation{
		Width:  320,
		Height: 1,
		Frames: []string{b.String()},
		Delays: []time.Duration{50 * time.Millisecond},
	}

	var buf bytes.Buffer
	if err := GIF(&buf, anim); err != nil {
		t.Fatalf("GIF: %v", err)
	}
	decoded, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	pal := decoded.Image[0].Palette
	if len(pal) > 256 {
		t.Fatalf("palette size = %d, want <= 256", len(pal))
	}
	found := false
	for _, c := range pal {
		if c == color.Color(green) {
			found = true
			break
		}
	}
	if !found {
		t.Error("frequent green missing from quantized palette")
	}
}

func TestGIFRejectsMissingDimensions(t *testing.T) {
	anim := &frames.Animation{Frames: []string{"x"}, Delays: []time.Duration{time.Millisecond}}
	if err := GIF(&bytes.Buffer{}, anim); err == nil {
		t.Error("want error for animation without dimensions, got nil")
	}
}

func TestSaveAppendsSuffixInsteadOfOverwriting(t *testing.T) {
	dir := t.TempDir()
	anim := &frames.Animation{
		Width:  1,
		Height: 1,
		Frames: []string{"x"},
		Delays: []time.Duration{50 * time.Millisecond},
	}

	first, err := Save(dir, "duck", anim)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if want := filepath.Join(dir, "duck.gif"); first != want {
		t.Errorf("first path = %q, want %q", first, want)
	}

	second, err := Save(dir, "duck", anim)
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if want := filepath.Join(dir, "duck-2.gif"); second != want {
		t.Errorf("second path = %q, want %q", second, want)
	}

	for _, path := range []string{first, second} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if _, err := gif.DecodeAll(bytes.NewReader(data)); err != nil {
			t.Errorf("%s is not a valid gif: %v", path, err)
		}
	}
}
