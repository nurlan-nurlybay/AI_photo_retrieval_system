package imageproc

import (
	"os"
	"testing"

	"github.com/h2non/bimg"
)

func TestVipsProcessor_Process(t *testing.T) {
	// load a sample image
	data, err := os.ReadFile("testdata/sample.jpg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	proc := NewVipsProcessor(256, 80)

	oriented, thumb, w, h, err := proc.Process(data)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if len(oriented) == 0 {
		t.Error("oriented output is empty")
	}
	if len(thumb) == 0 {
		t.Error("thumb output is empty")
	}
	if w == 0 || h == 0 {
		t.Errorf("unexpected dimensions: w=%d h=%d", w, h)
	}

	// optionally verify that the thumb really is 256x256
	size, err := bimg.NewImage(thumb).Size()
	if err != nil {
		t.Fatalf("cannot read thumb size: %v", err)
	}
	if size.Width != 256 || size.Height != 256 {
		t.Errorf("expected 256x256 thumb, got %dx%d", size.Width, size.Height)
	}

	// dump results for manual inspection (optional)
	_ = os.WriteFile("testdata/out-thumb.jpg", thumb, 0o644)
	_ = os.WriteFile("testdata/out-oriented.jpg", oriented, 0o644)
}
