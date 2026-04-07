package clip_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
<<<<<<< HEAD
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
=======
	"testing"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
	clipdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockServer(t *testing.T, vecDim int) (*httptest.Server, *clip.Client) {
	t.Helper()
	mockVec := make([]float32, vecDim)
	for i := range mockVec {
		mockVec[i] = float32(i) * 0.001
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"ok": true})

		case "/v1/encode/text/":
			var req clipdto.TextRequest
			json.NewDecoder(r.Body).Decode(&req)
			resp := clipdto.VectorResponse{Vectors: make([][]float32, len(req.Texts))}
			for i := range req.Texts {
				resp.Vectors[i] = mockVec
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case "/v1/encode/image/fast/":
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			file, _, err := r.FormFile("files")
			if err != nil {
				t.Fatalf("expected 'files' form field: %v", err)
			}
			defer file.Close()
			buf := new(bytes.Buffer)
			buf.ReadFrom(file)
			if buf.Len() == 0 {
				t.Fatal("uploaded file was empty")
			}
			resp := clipdto.VectorResponse{Vectors: [][]float32{mockVec}}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	client := clip.NewClientFromURL(srv.URL, srv.Client())
	return srv, client
}

func TestEmbedText(t *testing.T) {
	srv, client := setupMockServer(t, 1152)
	defer srv.Close()

	vec, err := client.EmbedText(context.Background(), "a cat on a couch")
	require.NoError(t, err)
	assert.Len(t, vec, 1152)
}

func TestEmbedImage_Fast(t *testing.T) {
	srv, client := setupMockServer(t, 1152)
	defer srv.Close()

	fakeJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // JPEG magic bytes + padding
	vec, err := client.EmbedImage(context.Background(), fakeJPEG)
	require.NoError(t, err)
	assert.Len(t, vec, 1152)
}

func TestEmbedText_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("model not loaded"))
	}))
	defer srv.Close()

	client := clip.NewClientFromURL(srv.URL, srv.Client())
	_, err := client.EmbedText(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}
