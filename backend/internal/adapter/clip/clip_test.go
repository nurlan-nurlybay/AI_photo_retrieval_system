package clip_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip"
)

func TestEmbedTextIntegration(t *testing.T) {
	client, _ := clip.NewClient(
		context.Background(),
		config.Clip{Host: "localhost", Port: 8003},
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
