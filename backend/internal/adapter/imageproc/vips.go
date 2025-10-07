package imageproc

import (
	"github.com/h2non/bimg"
)

type VipsProcessor struct {
	ThumbSize int // e.g. 512
	Quality   int // e.g. 85
}

func NewVipsProcessor(thumbSize, quality int) *VipsProcessor {
	if thumbSize <= 0 {
		thumbSize = 512
	}
	if quality <= 0 {
		quality = 85
	}
	return &VipsProcessor{ThumbSize: thumbSize, Quality: quality}
}

// Process auto-orients, returns oriented bytes in original format,
// plus a JPEG thumb. Width/height describe the oriented original.
func (p *VipsProcessor) Process(original []byte) (oriented []byte, thumb []byte, width int, height int, err error) {
	img := bimg.NewImage(original)

	// Auto-rotate by EXIF (might silently fail)
	oriented, err = img.AutoRotate()
	if err != nil {
		return nil, nil, 0, 0, err
	}

	// Oriented dimensions
	size, err := bimg.NewImage(oriented).Size()
	if err != nil {
		return nil, nil, 0, 0, err
	}

	// Square thumbnail, center-cropped
	thumb, err = bimg.NewImage(oriented).Process(bimg.Options{
		Width:   p.ThumbSize,
		Height:  p.ThumbSize,
		Crop:    true,
		Gravity: bimg.GravityCentre,
		Type:    bimg.JPEG,
		Quality: p.Quality,
	})
	if err != nil {
		return nil, nil, 0, 0, err
	}

	return oriented, thumb, size.Width, size.Height, nil
}
