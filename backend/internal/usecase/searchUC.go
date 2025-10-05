package usecase

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type SearchUsecase struct {
	embedder    Embedder
	vectorIndex VectorIndex
	mediaRepo   domain.MediaRepository
}

func NewSearchUsecase(embedder Embedder, vectorIndex VectorIndex, mediaRepo domain.MediaRepository) *SearchUsecase {
	return &SearchUsecase{
		embedder:    embedder,
		vectorIndex: vectorIndex,
		mediaRepo:   mediaRepo,
	}
}

func (s *SearchUsecase) SearchByText(ctx context.Context, deviceID, text string, k int) ([]*domain.Media, error) {
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
	for _, id := range ids {
		media, err := s.mediaRepo.Get(ctx, domain.UserID(deviceID), domain.MediaID(id))
		if err != nil {
			return nil, err
		}
		result = append(result, media)

	}

	return result, nil
}

func (s *SearchUsecase) SearchByImage(ctx context.Context, deviceID string, img []byte, k int) ([]domain.Media, error) {
	return nil, domain.ErrSearchFailed
}
