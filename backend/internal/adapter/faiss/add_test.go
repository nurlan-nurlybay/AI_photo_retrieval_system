package faiss_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/config"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss"
)

func makeVec(val float32) []float32 {
	v := make([]float32, 512) // CLIP embedding size
	for i := range v {
		v[i] = val
	}
	return v
}

func TestInsert_RealService(t *testing.T) {
	client, _ := faiss.NewClient(
		context.Background(),
		config.Faiss{Host: "localhost", Port: 8002},
		&http.Client{},
	)

	// 512-d float32 vector
	vec := makeVec(0.0)

	t.Logf("sending vector of length %d", len(vec))

	err := client.Insert(context.Background(), 404, 12345, vec)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	t.Log("Insert succeeded with real FAISS service")
}
