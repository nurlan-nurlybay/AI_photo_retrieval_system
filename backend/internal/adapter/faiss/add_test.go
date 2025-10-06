package faiss_test

import (
	"context"
	"testing"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/faiss"
)

func makeVec(val float64) []float64 {
	v := make([]float64, 512) // CLIP embedding size
	for i := range v {
		v[i] = val
	}
	return v
}

func TestInsert_RealService(t *testing.T) {
	client := faiss.NewClientWithBaseURL("http://localhost:8002", 5*time.Second)

	// 512-d float64 vector
	vec := makeVec(0.0001)

	t.Logf("sending vector of length %d", len(vec))

	err := client.Insert(context.Background(), 1234, vec)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	t.Log("Insert succeeded with real FAISS service")
}
