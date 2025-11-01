package usecase

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type SearchService interface {
	SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.MediaWithScore, error)
	SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]*domain.MediaWithScore, error)
}

type (
	Embedder interface {
		EmbedText(ctx context.Context, text string) ([]float32, error)
		EmbedImage(ctx context.Context, data []byte) ([]float32, error)
	}

	VectorIndex interface {
		Insert(ctx context.Context, userID, mediaId int64, vector []float32) error
		Search(ctx context.Context, userID int64, vector []float32, k int) ([]SearchResult, error)
		Delete(ctx context.Context, id int64) error
	}

	SearchResult struct {
		ID    int64
		Score float32
	}
)

type searchService struct {
	mediaRepo   domain.MediaRepository
	embedder    Embedder
	vectorIndex VectorIndex
	log         *logger.Logger
}

func NewSearchService(mediaRepo domain.MediaRepository, embedder Embedder, vectorIndex VectorIndex, log *logger.Logger) SearchService {
	return &searchService{
		mediaRepo:   mediaRepo,
		embedder:    embedder,
		vectorIndex: vectorIndex,
		log:         log,
	}
}

func (s *searchService) SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.MediaWithScore, error) {
	embedding, err := s.embedder.EmbedText(ctx, text)
	if err != nil {
		return nil, err
	}

	results, err := s.vectorIndex.Search(ctx, userID, embedding, k)
	if err != nil {
		return nil, err
	}

	// Fetch media info and attach score
	var out []*domain.MediaWithScore
	for _, r := range results { // r.ID, r.Score
		media, err := s.mediaRepo.Get(ctx, userID, r.ID)
		if err != nil {
			continue // skip missing media
		}
		out = append(out, &domain.MediaWithScore{
			Media: media,
			Score: r.Score,
		})
	}

	s.log.InfoContext(ctx, "search by text", "result:", results)

	return out, nil
}

func (s *searchService) SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]*domain.MediaWithScore, error) {
	embedding, err := s.embedder.EmbedImage(ctx, img)
	if err != nil {
		return nil, err
	}

	results, err := s.vectorIndex.Search(ctx, userID, embedding, k)
	if err != nil {
		return nil, err
	}

	// Fetch media info and attach score
	var out []*domain.MediaWithScore
	for _, r := range results {
		media, err := s.mediaRepo.Get(ctx, userID, r.ID)
		if err != nil {
			continue
		}
		out = append(out, &domain.MediaWithScore{
			Media: media,
			Score: r.Score,
		})
	}

	s.log.InfoContext(ctx, "search by image", "result:", results)

	return out, nil
}
