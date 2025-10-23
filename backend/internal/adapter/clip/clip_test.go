package clip_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
)

func TestEmbedTextIntegration(t *testing.T) {
	client, _ := clip.NewClient(
		context.Background(),
		config.Clip{Host: "localhost", Port: 8005},
		&http.Client{},
	)

	t.Log("sending request to Python CLIP service...")

	vec, err := client.EmbedText(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("[error] request failed: %v", err)
	}

	t.Logf("[debug] got vector slice length: %d", len(vec))
	if len(vec) != 512 {
		t.Fatalf("[error] expected vector length 512, got %d", len(vec))
	}

	t.Logf("[success] received valid vector, first 5 elements: %v", vec[:5])
}

func TestEmbedImage_WithLocalFile(t *testing.T) {
	// --- Step 1: Mock CLIP API server ---
	mockVec := make([]float32, 512)
	respBody := map[string][][]float32{"vectors": {mockVec}}
	respJSON, _ := json.Marshal(respBody)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		file, _, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("expected 'files' form field: %v", err)
		}
		defer file.Close()
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(file)
		if buf.Len() == 0 {
			t.Fatalf("uploaded file empty")
		}

		// Respond with vector
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respJSON)
	}))
	defer srv.Close()

	// --- Step 2: Initialize client ---
	client, _ := clip.NewClient(
		context.Background(),
		config.Clip{Host: "localhost", Port: 8005},
		&http.Client{},
	)

	// --- Step 3: Read actual test image ---
	imgPath := filepath.Join("test", "pic1.jpg")
	t.Logf("file path: %s", imgPath)
	data, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatalf("failed to read image %s: %v", imgPath, err)
	}

	// --- Step 4: Run EmbedImage ---
	vec, err := client.EmbedImage(context.Background(), data)
	if err != nil {
		t.Fatalf("EmbedImage failed: %v", err)
	}

	if len(vec) != 512 {
		t.Errorf("expected 512-d vector, got %d", len(vec))
	}
}
