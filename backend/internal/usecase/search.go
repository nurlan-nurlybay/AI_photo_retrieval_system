package usecase

import (
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type SearchUsecase struct {
	embedder    Embedder
	vectorIndex VectorIndex
	mediaRepo   MediaRepo
}

func NewSearchUsecase(embedder Embedder, vectorIndex VectorIndex, mediaRepo MediaRepo) *SearchUsecase {
	return &SearchUsecase{
		embedder:    embedder,
		vectorIndex: vectorIndex,
		mediaRepo:   mediaRepo,
	}
}

func (s *SearchUsecase) SearchByText(ctx context.Context, deviceID, text string, k int) ([]domain.Media, error) {
	embedding, err := s.embedder.EmbedText(ctx, text)
	if err != nil {
		return nil, err
	}

	results, err := s.vectorIndex.Search(ctx, embedding, k)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, r := range results {
		ids = append(ids, r.ID)
	}

	medias, err := s.mediaRepo.FindByIDs(ctx, deviceID, ids)
	if err != nil {
		return nil, err
	}

	var result []domain.Media
	for _, media := range medias {
		if media.IsActive() {
			result = append(result, media)
		}
	}

	return result, nil
}

func (s *SearchUsecase) SearchByImage(ctx context.Context, deviceID string, img []byte, k int) ([]domain.Media, error) {
	return nil, domain.ErrSearchFailed
}
