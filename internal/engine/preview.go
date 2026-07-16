package engine

import (
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"io"
)

// RenderPreview decodes only the first frame of a GIF and converts it to
// a single ASCII frame. Unlike Render, which converts every frame, this
// is cheap enough to call on every gallery selection change.
func RenderPreview(r io.Reader, opts Options) (text string, cols, rows int, err error) {
	src, err := gif.Decode(r)
	if err != nil {
		return "", 0, 0, fmt.Errorf("decoding gif: %w", err)
	}

	bounds := src.Bounds()
	frame := image.NewRGBA(bounds)
	draw.Draw(frame, bounds, src, bounds.Min, draw.Src)

	cols, rows = targetGrid(bounds.Dx(), bounds.Dy(), opts)
	ramp := []rune(opts.ramp())

	if opts.FilterBackground {
		if bg, ok := detectBackground(frame); ok {
			masked := image.NewRGBA(bounds)
			maskBackground(masked, frame, bg)
			frame = masked
		}
	}

	return frameToASCII(frame, cols, rows, opts.Colored, ramp), cols, rows, nil
}
