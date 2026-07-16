package engine

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"strings"
	"testing"
)

// tinyPreviewGIF encodes a 2-frame 8x4 checkerboard GIF; RenderPreview
// only looks at the first frame.
func tinyPreviewGIF(t *testing.T) []byte {
	t.Helper()
	palette := []color.Color{color.Black, color.White}
	g := &gif.GIF{}
	for i := 0; i < 2; i++ {
		img := image.NewPaletted(image.Rect(0, 0, 8, 4), palette)
		for y := 0; y < 4; y++ {
			for x := 0; x < 8; x++ {
				img.SetColorIndex(x, y, uint8((x+y+i)%2))
			}
		}
		g.Image = append(g.Image, img)
		g.Delay = append(g.Delay, 5)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatalf("encoding gif: %v", err)
	}
	return buf.Bytes()
}

func TestRenderPreviewProducesGrid(t *testing.T) {
	data := tinyPreviewGIF(t)
	text, cols, rows, err := RenderPreview(bytes.NewReader(data), Options{MaxWidth: 20, MaxHeight: 10})
	if err != nil {
		t.Fatalf("RenderPreview: %v", err)
	}
	if cols <= 0 || rows <= 0 {
		t.Fatalf("grid = %dx%d, want positive dimensions", cols, rows)
	}
	if got := strings.Count(text, "\n") + 1; got != rows {
		t.Errorf("text has %d lines, want %d", got, rows)
	}
}

func TestRenderPreviewFiltersBackground(t *testing.T) {
	data := solidBGGif(t)
	text, _, _, err := RenderPreview(bytes.NewReader(data), Options{Width: 20, Height: 20, FilterBackground: true})
	if err != nil {
		t.Fatalf("RenderPreview: %v", err)
	}
	if got := strings.TrimSpace(strings.Split(text, "\n")[0]); got != "" {
		t.Errorf("top row = %q, want blank background", got)
	}
}

func TestRenderPreviewRejectsBadData(t *testing.T) {
	if _, _, _, err := RenderPreview(strings.NewReader("not a gif"), Options{}); err == nil {
		t.Fatal("RenderPreview succeeded on invalid data, want error")
	}
}
