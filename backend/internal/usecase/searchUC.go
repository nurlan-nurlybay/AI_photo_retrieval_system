package usecase

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type SearchService interface {
	SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.Media, error)
	SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]domain.Media, error)
}

type (
	Embedder interface {
		EmbedText(ctx context.Context, text string) ([]float64, error)
		EmbedImage(ctx context.Context, data []byte) ([]float64, error)
	}

	VectorIndex interface {
		Insert(ctx context.Context, id int64, vector []float64) error
		Search(ctx context.Context, vector []float64, k int) ([]SearchResult, error)
		Delete(ctx context.Context, id int64) error
	}

	SearchResult struct {
		ID    int64
		Score float64
	}
)

type searchService struct {
	embedder    Embedder
	vectorIndex VectorIndex
	mediaRepo   domain.MediaRepository
}

func NewSearchService(mediaRepo domain.MediaRepository, embedder Embedder, vectorIndex VectorIndex) SearchService {
	return &searchService{
		embedder:    embedder,
		vectorIndex: vectorIndex,
		mediaRepo:   mediaRepo,
	}
}

func (s *searchService) SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.Media, error) {
	embedding, err := s.embedder.EmbedText(ctx, text)
	if err != nil {
		return nil, err
	}

	results, err := s.vectorIndex.Search(ctx, embedding, k)
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, r := range results {
		ids = append(ids, r.ID)
	}

	var result []*domain.Media
	for _, mediaID := range ids {
		media, err := s.mediaRepo.Get(ctx, userID, mediaID)
		if err != nil {
			return nil, err
		}
		result = append(result, media)

	}

	return result, nil
}

func (s *searchService) SearchByImage(ctx context.Context, deviceID int64, img []byte, k int) ([]domain.Media, error) {
	return nil, domain.ErrSearchFailed
}
