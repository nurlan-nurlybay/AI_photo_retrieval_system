package usecase

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedImage(ctx context.Context, data []byte) ([]float32, error)
}

type VectorIndex interface {
	Search(ctx context.Context, deviceID string, embedding []float32, k int) ([]string, error)
	Insert(ctx context.Context, deviceID, id string, embedding []float32) error
	Delete(ctx context.Context, deviceID, id string) error
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
