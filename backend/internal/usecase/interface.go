package usecase

import (
	"context"
)

type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float64, error)
	EmbedImage(ctx context.Context, data []byte, filename string) ([]float64, error)
}

type VectorIndex interface {
	Insert(ctx context.Context, id int64, vector []float64) error
	Search(ctx context.Context, vector []float64, k int) ([]SearchResult, error)
	Delete(ctx context.Context, id int64) error
}

type SearchResult struct {
	ID    int64
	Score float64
}

