// Package export rasterizes rendered ASCII animations back into GIF
// files, drawing each character cell with a monospace bitmap font.
package export

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/jhayashi1/ascii-tui/internal/frames"
)

// Cell geometry of basicfont.Face7x13: 7px advance, 13px line height,
// baseline 11px below the top of the cell.
const (
	cellW  = 7
	cellH  = 13
	ascent = 11
)

var (
	background = color.RGBA{A: 255}
	// defaultFg colors characters in animations rendered without ANSI
	// colors, matching a typical terminal's light-gray text.
	defaultFg = color.RGBA{R: 235, G: 235, B: 235, A: 255}
)

// cell is one character of a parsed frame with its foreground color.
type cell struct {
	ch rune
	fg color.RGBA
}

// PixelSize returns the dimensions of the GIF this package produces
// for the animation.
func PixelSize(anim *frames.Animation) (w, h int) {
	return anim.Width * cellW, anim.Height * cellH
}

// GIF encodes the animation as an infinitely looping GIF, one image per
// frame at cellW x cellH pixels per character on a black background.
func GIF(w io.Writer, anim *frames.Animation) error {
	cols, rows := anim.Width, anim.Height
	if cols <= 0 || rows <= 0 {
		return errors.New("animation has no dimensions")
	}
	out := gif.GIF{
		Image: make([]*image.Paletted, len(anim.Frames)),
		Delay: make([]int, len(anim.Frames)),
	}
	for i, frame := range anim.Frames {
		out.Image[i] = rasterize(parseFrame(frame), cols, rows)
		out.Delay[i] = max(1, int((anim.Delays[i]+5*time.Millisecond)/(10*time.Millisecond)))
	}
	if err := gif.EncodeAll(w, &out); err != nil {
		return fmt.Errorf("encoding gif: %w", err)
	}
	return nil
}

// Save writes the animation as <name>.gif into dir, appending a numeric
// suffix rather than overwriting an existing file. It returns the path.
func Save(dir, name string, anim *frames.Animation) (string, error) {
	path := filepath.Join(dir, name+".gif")
	for n := 2; ; n++ {
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("checking export path: %w", err)
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d.gif", name, n))
	}
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("creating gif file: %w", err)
	}
	if err := GIF(f, anim); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("writing gif file: %w", err)
	}
	return path, nil
}

// parseFrame decodes one frame string into rows of colored cells,
// interpreting the 24-bit foreground escapes emitted by the engine.
func parseFrame(frame string) [][]cell {
	lines := strings.Split(frame, "\n")
	rows := make([][]cell, len(lines))
	for y, line := range lines {
		fg := defaultFg
		cells := make([]cell, 0, len(line))
		for i := 0; i < len(line); {
			if line[i] == 0x1b {
				m := strings.IndexByte(line[i:], 'm')
				if m < 0 {
					break
				}
				fg = escapeColor(line[i : i+m+1])
				i += m + 1
				continue
			}
			r, size := utf8.DecodeRuneInString(line[i:])
			cells = append(cells, cell{ch: r, fg: fg})
			i += size
		}
		rows[y] = cells
	}
	return rows
}

// escapeColor maps an ANSI escape sequence to the foreground color it
// selects: a 24-bit color for "\x1b[38;2;R;G;Bm", the default for reset
// and anything unrecognized.
func escapeColor(seq string) color.RGBA {
	var r, g, b int
	if n, err := fmt.Sscanf(seq, "\x1b[38;2;%d;%d;%dm", &r, &g, &b); err == nil && n == 3 {
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}
	return defaultFg
}

// rasterize draws the parsed cells onto a paletted image sized for the
// animation's full cols x rows grid, coalescing same-color runs into
// single DrawString calls.
func rasterize(rows [][]cell, cols, rowCount int) *image.Paletted {
	pal := framePalette(rows)
	img := image.NewPaletted(image.Rect(0, 0, cols*cellW, rowCount*cellH), pal)
	d := font.Drawer{Dst: img, Face: basicfont.Face7x13}
	for y := 0; y < min(len(rows), rowCount); y++ {
		line := rows[y]
		for x := 0; x < min(len(line), cols); {
			if line[x].ch == ' ' {
				x++
				continue
			}
			fg := line[x].fg
			var run strings.Builder
			start := x
			for x < min(len(line), cols) && line[x].ch != ' ' && line[x].fg == fg {
				run.WriteRune(line[x].ch)
				x++
			}
			d.Src = image.NewUniform(pal.Convert(fg))
			d.Dot = fixed.P(start*cellW, y*cellH+ascent)
			d.DrawString(run.String())
		}
	}
	return img
}

// framePalette builds the frame's GIF palette: the background first,
// then the cell colors by decreasing frequency. Past GIF's 256-color
// limit the remaining colors are left to map onto their nearest kept
// neighbor via Palette.Convert.
func framePalette(rows [][]cell) color.Palette {
	counts := make(map[color.RGBA]int)
	for _, line := range rows {
		for _, c := range line {
			if c.ch != ' ' {
				counts[c.fg]++
			}
		}
	}
	colors := make([]color.RGBA, 0, len(counts))
	for c := range counts {
		if c != background {
			colors = append(colors, c)
		}
	}
	slices.SortFunc(colors, func(a, b color.RGBA) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return rgbKey(a) - rgbKey(b)
	})
	pal := color.Palette{background}
	for _, c := range colors {
		if len(pal) == 256 {
			break
		}
		pal = append(pal, c)
	}
	return pal
}

// rgbKey packs a color into a comparable int for deterministic ordering.
func rgbKey(c color.RGBA) int {
	return int(c.R)<<16 | int(c.G)<<8 | int(c.B)
}
