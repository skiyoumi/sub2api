package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	imagePreviewMaxEdge   = 1024
	imagePreviewMaxPixels = 32 * 1024 * 1024
)

// Bound decoded pixel memory across concurrent image tasks. Original bytes are
// stored independently; unsupported or oversized images simply have no preview.
var imagePreviewSlots = make(chan struct{}, 2)

func buildImagePreview(ctx context.Context, data []byte) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > imagePreviewMaxPixels/cfg.Height {
		return nil, errors.New("image exceeds preview pixel limit")
	}
	select {
	case imagePreviewSlots <- struct{}{}:
		defer func() { <-imagePreviewSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	w, h := cfg.Width, cfg.Height
	if w >= h && w > imagePreviewMaxEdge {
		h = max(1, h*imagePreviewMaxEdge/w)
		w = imagePreviewMaxEdge
	} else if h > imagePreviewMaxEdge {
		w = max(1, w*imagePreviewMaxEdge/h)
		h = imagePreviewMaxEdge
	}
	preview := image.NewRGBA(image.Rect(0, 0, w, h))
	// JPEG has no alpha channel; composite transparent source pixels onto white.
	draw.Draw(preview, preview.Bounds(), image.White, image.Point{}, draw.Src)
	xdraw.ApproxBiLinear.Scale(preview, preview.Bounds(), source, source.Bounds(), draw.Over, nil)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, preview, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}
