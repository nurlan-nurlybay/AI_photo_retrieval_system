package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db"
)

const (
	EmbTableName = "embeddings"

	EmbMediaIDCol  = "media_id"
	EmbModelCol    = "model"
	EmbVecBytesCol = "vec_bytes"
	EmbStatusCol   = "status"
	EmbLastErrCol  = "last_error"
	EmbCreatedCol  = "created_at"
	EmbUpdatedCol  = "updated_at"
)

type EmbeddingsRepo struct {
	db db.Client
}

func NewEmbeddingsRepo(db db.Client) *EmbeddingsRepo {
	return &EmbeddingsRepo{db: db}
}

func (r *EmbeddingsRepo) Create(ctx context.Context, emb *domain.Embedding) error {
	query, args, err := squirrel.
		Insert(EmbTableName).
		Columns(
			EmbMediaIDCol, EmbModelCol, EmbVecBytesCol,
			EmbStatusCol, EmbLastErrCol, EmbCreatedCol, EmbUpdatedCol,
		).
		Values(
			emb.MediaID, emb.Model, emb.VecBytes,
			emb.Status, emb.LastError, emb.CreatedAt, emb.UpdatedAt,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build emb insert: %w", err)
	}

	q := db.Query{
		Name:     "Emb.Create",
		QueryRaw: query,
	}

	if _, err := r.db.DB().ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("exec emb insert: %w", err)
	}

	return nil
}

func (r *EmbeddingsRepo) Get(ctx context.Context, mediaID int64) (*domain.Embedding, error) {
	query, args, err := squirrel.
		Select(
			EmbMediaIDCol, EmbModelCol, EmbVecBytesCol,
			EmbStatusCol, EmbLastErrCol, EmbCreatedCol, EmbUpdatedCol,
		).
		From(EmbTableName).
		Where(squirrel.Eq{EmbMediaIDCol: mediaID}).
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build emb select: %w", err)
	}

	q := db.Query{
		Name:     "Emb.Get",
		QueryRaw: query,
	}

	row := r.db.DB().QueryRowContext(ctx, q, args...)

	var emb domain.Embedding
	err = row.Scan(
		&emb.MediaID, &emb.Model, &emb.VecBytes,
		&emb.Status, &emb.LastError, &emb.CreatedAt, &emb.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan emb: %w", err)
	}

	return &emb, nil
}

func (r *EmbeddingsRepo) Delete(ctx context.Context, mediaID int64) error {
	query, args, err := squirrel.
		Delete(EmbTableName).
		Where(squirrel.Eq{EmbMediaIDCol: mediaID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build emb delete: %w", err)
	}

	q := db.Query{
		Name:     "Emb.Delete",
		QueryRaw: query,
	}

	_, err = r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("exec emb delete: %w", err)
	}

	return nil
}

func (r *EmbeddingsRepo) MarkPending(ctx context.Context, mediaID int64) error {
	return r.updateStatus(ctx, mediaID, "pending", "")
}

func (r *EmbeddingsRepo) MarkInIndex(ctx context.Context, mediaID int64) error {
	return r.updateStatus(ctx, mediaID, "in_index", "")
}

func (r *EmbeddingsRepo) MarkFailed(ctx context.Context, mediaID int64, msg string) error {
	return r.updateStatus(ctx, mediaID, "failed", msg)
}

func (r *EmbeddingsRepo) updateStatus(ctx context.Context, mediaID int64, status, msg string) error {
	query, args, err := squirrel.
		Update(EmbTableName).
		Set(EmbStatusCol, status).
		Set(EmbLastErrCol, msg).
		Set(EmbUpdatedCol, time.Now().UTC()).
		Where(squirrel.Eq{EmbMediaIDCol: mediaID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build emb status update: %w", err)
	}

	q := db.Query{
		Name:     "Emb.UpdateStatus",
		QueryRaw: query,
	}

	if _, err := r.db.DB().ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("exec emb status update: %w", err)
	}
	return nil
}
