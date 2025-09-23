package usecase

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedImage(ctx context.Context, data []byte) ([]float32, error)
}

type SearchResult struct {
	ID    string
	Score float64
}

type VectorIndex interface {
	Insert(ctx context.Context, id string, vector []float32) error
	Search(ctx context.Context, vector []float32, k int) ([]SearchResult, error)
	Delete(ctx context.Context, id string) error
}

type MediaRepo interface {
	FindByIDs(ctx context.Context, deviceID string, ids []string) ([]domain.Media, error)
}

// TODO: remove if not needed
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
}
