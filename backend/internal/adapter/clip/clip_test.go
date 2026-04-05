package clip_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
}
