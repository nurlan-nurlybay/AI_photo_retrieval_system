package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
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
func (r *WorkRepo) LoadMediaBytes(ctx context.Context, mediaID int64) ([]byte, error) {
	var url string

	// Look up the URL and MIME type for the given media ID
	err := r.DB.QueryRow(ctx, `
		SELECT url  
		FROM media 
		WHERE id = $1
	`, mediaID).Scan(&url)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("media %d not found", mediaID)
		}
		return nil, fmt.Errorf("failed to query media: %w", err)
	}

	// Remove URL part
	publicBase := "http://localhost:8080/uploads/"
	key := strings.TrimPrefix(url, publicBase)

	data, err := r.Seaweed.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch media from storage: %w", err)
	}

	return data, nil
}

// Embeddings table ops
func (r *WorkRepo) UpsertEmbedding(ctx context.Context, mediaID int64, model string, vecBytes []byte) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO embeddings (media_id, model, vec_bytes, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', NOW(), NOW())
		ON CONFLICT (media_id) DO UPDATE
		SET 
			model = EXCLUDED.model,
			vec_bytes = EXCLUDED.vec_bytes,
			status = 'pending',
			last_error = NULL,
			updated_at = NOW()
	`, mediaID, model, vecBytes)

	if err != nil {
		return fmt.Errorf("failed to upsert embedding for media %d: %w", mediaID, err)
	}
	return nil
}

func (r *WorkRepo) MarkPending(ctx context.Context, mediaID int64) error {
	_, err := r.DB.Exec(ctx, `
		UPDATE embeddings
		SET status = 'pending',
		    last_error = NULL,
		    updated_at = NOW()
		WHERE media_id = $1
	`, mediaID)
	if err != nil {
		return fmt.Errorf("failed to mark embedding %d as pending: %w", mediaID, err)
	}
	return nil
}

func (r *WorkRepo) MarkInIndex(ctx context.Context, mediaID int64) error {
	_, err := r.DB.Exec(ctx, `
		UPDATE embeddings
		SET status = 'in_index',
		    last_error = NULL,
		    updated_at = NOW()
		WHERE media_id = $1
	`, mediaID)
	if err != nil {
		return fmt.Errorf("failed to mark embedding %d as in_index: %w", mediaID, err)
	}
	return nil
}

func (r *WorkRepo) MarkFailed(ctx context.Context, mediaID int64, msg string) error {
	_, err := r.DB.Exec(ctx, `
		UPDATE embeddings
		SET status = 'failed',
		    last_error = $2,
		    updated_at = NOW()
		WHERE media_id = $1
	`, mediaID, msg)
	if err != nil {
		return fmt.Errorf("failed to mark embedding %d as failed: %w", mediaID, err)
	}
	return nil
}

// Retry helpers
func (r *WorkRepo) ListUnindexed(ctx context.Context, limit int) ([]int64, error) {
	return nil, nil
}

func (r *WorkRepo) GetEmbeddingBytes(ctx context.Context, mediaID int64) ([]byte, error) {
	return nil, nil
}
