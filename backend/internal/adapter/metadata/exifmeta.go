package metadata

import (
	"bytes"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/usecase"
	"github.com/rwcarlsen/goexif/exif"
)

type ExifExtractor struct{}

func NewExifExtractor() *ExifExtractor { return &ExifExtractor{} }

func (e *ExifExtractor) Extract(buf []byte) (usecase.ExtractedMetadata, error) {
	out := usecase.ExtractedMetadata{}

	x, err := exif.Decode(bytes.NewReader(buf))
	if err != nil {
		// No EXIF. Not an error for us; return zero meta
		return out, nil
	}

	// DateTimeOriginal
	if t, err := x.DateTime(); err == nil && !t.IsZero() {
		tt := t
		out.DateTimeOriginal = &tt
	} else if tag, err := x.Get(exif.DateTimeOriginal); err == nil && tag != nil {
		if s, err := tag.StringVal(); err == nil {
			if tt, perr := parseEXIFTime(s); perr == nil {
				out.DateTimeOriginal = &tt
			}
		}
	}

	// Orientation (1..8). Default 1 if missing
	if tag, err := x.Get(exif.Orientation); err == nil && tag != nil {
		if ori, err := tag.Int(0); err == nil {
			out.Orientation = ori
		}
	}

	// Make / Model / Software
	if tag, err := x.Get(exif.Make); err == nil && tag != nil {
		out.CameraMake, _ = tag.StringVal()
	}
	if tag, err := x.Get(exif.Model); err == nil && tag != nil {
		out.CameraModel, _ = tag.StringVal()
	}
	if tag, err := x.Get(exif.Software); err == nil && tag != nil {
		out.Software, _ = tag.StringVal()
	}

	return out, nil
}

func parseEXIFTime(s string) (time.Time, error) {
	// "2006:01:02 15:04:05"
	layout := "2006:01:02 15:04:05"
	t, err := time.ParseInLocation(layout, s, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
