package usecase

import (
<<<<<<< HEAD
	"context"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
=======
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	clipdto "github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/adapter/clip/dto"
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/logger"
)

type SearchService interface {
<<<<<<< HEAD
	SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.MediaWithScore, error)
	SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]*domain.MediaWithScore, error)
=======
	SearchByText(ctx context.Context, userID int64, text string, k int) ([]*domain.MediaWithScore, bool, error)
	SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]*domain.MediaWithScore, bool, error)
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}

type (
	Embedder interface {
		EmbedText(ctx context.Context, text string) ([]float32, error)
		EmbedImage(ctx context.Context, data []byte) ([]float32, error)
<<<<<<< HEAD
	}

	VectorIndex interface {
		Insert(ctx context.Context, userID, mediaId int64, vector []float32) error
		Search(ctx context.Context, userID int64, vector []float32, k int) ([]SearchResult, error)
		Delete(ctx context.Context, id int64) error
=======
		EmbedImageURL(ctx context.Context, url string) ([]float32, error)
		EmbedImageURLSlow(ctx context.Context, url string) (*clipdto.SlowEncodeResult, error)
	}

	VectorClient interface {
		SearchHybrid(ctx context.Context, namespace string, queryText string, imageVec []float32, textVec []float32, topK int) ([]SearchResult, bool, error)
		DeleteItems(ctx context.Context, userID int64, imageIDs []int64) error
		ClearNamespace(ctx context.Context, userID int64) error
		NukeNamespace(ctx context.Context, userID int64) error
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	}

	SearchResult struct {
		ID    int64
		Score float32
	}
)

type searchService struct {
<<<<<<< HEAD
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
=======
	mediaRepo    domain.MediaRepository
	embedder     Embedder
	vectorClient VectorClient
	store        ObjectStorage
	log          *logger.Logger
}

func NewSearchService(mediaRepo domain.MediaRepository, embedder Embedder, vectorClient VectorClient, store ObjectStorage, log *logger.Logger) SearchService {
	return &searchService{
		mediaRepo:    mediaRepo,
		embedder:     embedder,
		vectorClient: vectorClient,
		store:        store,
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
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	}

	// Fetch media info and attach score
	var out []*domain.MediaWithScore
	for _, r := range results { // r.ID, r.Score
		media, err := s.mediaRepo.Get(ctx, userID, r.ID)
		if err != nil {
			continue // skip missing media
		}
		out = append(out, &domain.MediaWithScore{
<<<<<<< HEAD
			Media: media,
			Score: r.Score,
		})
	}

	s.log.InfoContext(ctx, "search by text", "results", results)

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
=======
			Media:    media,
			Score:    r.Score,
			UsedQwen: usedQwen,
		})
	}

	s.log.InfoContext(ctx, "search by text", "results_count", len(results), "used_qwen", usedQwen)

	return out, usedQwen, nil
}

func (s *searchService) SearchByImage(ctx context.Context, userID int64, img []byte, k int) ([]*domain.MediaWithScore, bool, error) {
	// 1. Upload to temporary S3 location
	key := fmt.Sprintf("temp/search/%d/%d.jpg", userID, time.Now().UnixNano())
	_, err := s.store.Put(ctx, key, bytes.NewReader(img))
	if err != nil {
		return nil, false, fmt.Errorf("failed to upload search image to temp storage: %w", err)
	}
	defer s.store.Delete(context.Background(), key) // ensure cleanup async and safely

	// 2. Generate presigned URL
	url, err := s.store.GeneratePresignedURL(ctx, key, 15*time.Minute)
	if err != nil {
		return nil, false, fmt.Errorf("failed to generate presigned URL for search image: %w", err)
	}

	// 3. Embed using URL
	embedding, err := s.embedder.EmbedImageURL(ctx, url)
	if err != nil {
		return nil, false, err
	}

	namespace := fmt.Sprintf("user_%d", userID)

	results, usedQwen, err := s.vectorClient.SearchHybrid(ctx, namespace, "", embedding, nil, k)
	if err != nil {
		return nil, false, err
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
	}

	// Fetch media info and attach score
	var out []*domain.MediaWithScore
	for _, r := range results {
		media, err := s.mediaRepo.Get(ctx, userID, r.ID)
		if err != nil {
			continue
		}
		out = append(out, &domain.MediaWithScore{
<<<<<<< HEAD
			Media: media,
			Score: r.Score,
		})
	}

	s.log.InfoContext(ctx, "search by image", "results", results)

	return out, nil
=======
			Media:    media,
			Score:    r.Score,
			UsedQwen: usedQwen,
		})
	}

	s.log.InfoContext(ctx, "search by image", "results_count", len(results), "used_qwen", usedQwen)

	return out, usedQwen, nil
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
}
