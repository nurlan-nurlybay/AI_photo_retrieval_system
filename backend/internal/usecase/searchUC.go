package usecase

import (
	"context"
	"fmt"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type SearchService interface {
	SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.MediaWithScore, bool, error)
	SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]*domain.MediaWithScore, bool, error)
}

type (
	Embedder interface {
		EmbedText(ctx context.Context, text string) ([]float32, error)
		EmbedImage(ctx context.Context, data []byte) ([]float32, error)
	}

	VectorClient interface {
		SearchHybrid(ctx context.Context, namespace string, queryText string, imageVec []float32, textVec []float32, topK int) ([]SearchResult, bool, error)
		DeleteImage(ctx context.Context, namespace string, imageID int64) error
	}

	SearchResult struct {
		ID    int64
		Score float32
	}
)

type searchService struct {
	mediaRepo    domain.MediaRepository
	embedder     Embedder
	vectorClient VectorClient
	log          *logger.Logger
}

func NewSearchService(mediaRepo domain.MediaRepository, embedder Embedder, vectorClient VectorClient, log *logger.Logger) SearchService {
	return &searchService{
		mediaRepo:    mediaRepo,
		embedder:     embedder,
		vectorClient: vectorClient,
		log:          log,
	}
}

func (s *searchService) SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.MediaWithScore, bool, error) {
	embedding, err := s.embedder.EmbedText(ctx, text)
	if err != nil {
		return nil, false, err
	}

	namespace := fmt.Sprintf("user_%d", userID)

	// SigLIP text vectors live in the same embedding space as image vectors,
	// so pass the text embedding as imageVec for baseline search.
	results, usedQwen, err := s.vectorClient.SearchHybrid(ctx, namespace, text, embedding, nil, k)
	if err != nil {
		return nil, false, err
	}

	// Fetch media info and attach score
	var out []*domain.MediaWithScore
	for _, r := range results { // r.ID, r.Score
		media, err := s.mediaRepo.Get(ctx, userID, r.ID)
		if err != nil {
			continue // skip missing media
		}
		out = append(out, &domain.MediaWithScore{
			Media:    media,
			Score:    r.Score,
			UsedQwen: usedQwen,
		})
	}

	s.log.InfoContext(ctx, "search by text", "results_count", len(results), "used_qwen", usedQwen)

	return out, usedQwen, nil
}

func (s *searchService) SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]*domain.MediaWithScore, bool, error) {
	embedding, err := s.embedder.EmbedImage(ctx, img)
	if err != nil {
		return nil, false, err
	}

	namespace := fmt.Sprintf("user_%d", userID)

	results, usedQwen, err := s.vectorClient.SearchHybrid(ctx, namespace, "", embedding, nil, k)
	if err != nil {
		return nil, false, err
	}

	// Fetch media info and attach score
	var out []*domain.MediaWithScore
	for _, r := range results {
		media, err := s.mediaRepo.Get(ctx, userID, r.ID)
		if err != nil {
			continue
		}
		out = append(out, &domain.MediaWithScore{
			Media:    media,
			Score:    r.Score,
			UsedQwen: usedQwen,
		})
	}

	s.log.InfoContext(ctx, "search by image", "results_count", len(results), "used_qwen", usedQwen)

	return out, usedQwen, nil
}
