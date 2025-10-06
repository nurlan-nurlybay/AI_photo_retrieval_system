package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/worker"
)

type Seaweed interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

type WorkRepo struct {
	DB      *pgxpool.Pool
	Seaweed Seaweed
}

func NewWorkRepo(db *pgxpool.Pool, seaweed Seaweed) worker.Repo {
	return &WorkRepo{DB: db, Seaweed: seaweed}
}

// SeaweedFS fetch
func (r *WorkRepo) LoadMediaBytes(ctx context.Context, mediaID int64) ([]byte, string, error) {
	return nil, "", nil
}

func extFromMime(m string) string {
	m = strings.ToLower(m)
	switch m {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/heic":
		return ".heic"
	case "image/heif":
		return ".heif"
	case "image/tiff":
		return ".tiff"
	default:
		return ""
	}
}

// Embeddings table ops
func (r *WorkRepo) UpsertEmbedding(ctx context.Context, mediaID int64, model string, vecBytes []byte) error {

	return nil
}

func (r *WorkRepo) MarkPending(ctx context.Context, mediaID int64) error {
	return nil
}

func (r *WorkRepo) MarkInIndex(ctx context.Context, mediaID int64) error {
	return nil
}

func (r *WorkRepo) MarkFailed(ctx context.Context, mediaID int64, msg string) error {
	return nil
}

// Retry helpers

func (r *WorkRepo) ListUnindexed(ctx context.Context, limit int) ([]int64, error) {
	return nil, nil
}

func (r *WorkRepo) GetEmbeddingBytes(ctx context.Context, mediaID int64) ([]byte, error) {
	return nil, nil
}
