package usecase

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float64, error)
	EmbedImage(ctx context.Context, data []byte, filename string) ([]float64, error)
}

type SearchResult struct {
	ID    int64
	Score float64
}

type VectorIndex interface {
	Insert(ctx context.Context, id int64, vector []float64) error
	Search(ctx context.Context, vector []float64, k int) ([]SearchResult, error)
	Delete(ctx context.Context, id int64) error
}

type MediaRepo interface {
	FindByIDs(ctx context.Context, deviceID string, ids []int64) ([]domain.Media, error)
}

// TODO: remove if not needed
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
}
