package metadata

import (
	"os"
	"testing"
	"time"
)

func TestExifExtractor_Extract_WithExif(t *testing.T) {
	img, err := os.ReadFile("testdata/exif.jpg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	ex := NewExifExtractor()
	meta, err := ex.Extract(img)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	t.Logf("EXIF results:\n"+
		"  Orientation: %d\n"+
		"  DateTimeOriginal: %v\n"+
		"  Make: %q\n"+
		"  Model: %q\n"+
		"  Software: %q\n",
		meta.Orientation,
		meta.DateTimeOriginal,
		meta.CameraMake,
		meta.CameraModel,
		meta.Software,
	)

	if meta.Orientation == 0 {
		t.Errorf("expected non-zero Orientation, got %d", meta.Orientation)
	}
	if meta.DateTimeOriginal == nil {
		t.Error("expected DateTimeOriginal to be parsed, got nil")
	} else if meta.DateTimeOriginal.IsZero() {
		t.Error("DateTimeOriginal is zero")
	}
	if meta.CameraMake == "" {
		t.Error("expected CameraMake, got empty string")
	}
	if meta.CameraModel == "" {
		t.Error("expected CameraModel, got empty string")
	}
}

func TestExifExtractor_Extract_NoExif(t *testing.T) {
	img, err := os.ReadFile("testdata/no_exif.jpg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	ex := NewExifExtractor()
	meta, err := ex.Extract(img)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if meta.Orientation != 0 {
		t.Errorf("expected Orientation 0, got %d", meta.Orientation)
	}
	if meta.DateTimeOriginal != nil {
		t.Error("expected DateTimeOriginal to be nil for no EXIF image")
	}
}

func TestParseEXIFTime(t *testing.T) {
	input := "2024:05:30 13:45:21"
	got, err := parseEXIFTime(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2024, 5, 30, 13, 45, 21, 0, time.Local)
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}
