package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/worker"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/pkg/db"
)

const (
	EmbTableName = "embeddings"

	EmbMediaIDCol  = "media_id"
	EmbUserIDCol   = "user_id"
	EmbModelCol    = "model"
	EmbVecBytesCol = "vec_bytes"
	EmbStatusCol   = "status"
	EmbLastErrCol  = "last_error"
	EmbCreatedCol  = "created_at"
	EmbUpdatedCol  = "updated_at"
)

var _ worker.EmbeddingsRepo = (*EmbeddingsRepo)(nil)

type EmbeddingsRepo struct {
	db db.Client
}

func NewEmbeddingsRepo(db db.Client) *EmbeddingsRepo {
	return &EmbeddingsRepo{db: db}
}

func (r *EmbeddingsRepo) UpsertEmbedding(ctx context.Context, emb *domain.Embedding) error {
	query, args, err := squirrel.
		Insert(EmbTableName).
		Columns(
			EmbMediaIDCol, EmbUserIDCol, EmbModelCol, EmbVecBytesCol,
			EmbStatusCol, EmbLastErrCol, EmbCreatedCol, EmbUpdatedCol,
		).
		Values(
			emb.MediaID, emb.UserID, emb.Model, emb.VecBytes,
			emb.Status, emb.LastError, emb.CreatedAt, emb.UpdatedAt,
		).
		Suffix(`ON CONFLICT (media_id, model) DO UPDATE SET 
			vec_bytes = EXCLUDED.vec_bytes, 
			status = EXCLUDED.status, 
			last_error = EXCLUDED.last_error, 
			updated_at = EXCLUDED.updated_at`).
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

// GetEmbedding fetches the row for (userID, mediaID, model). With the
// composite PK (media_id, model) the same media may have multiple embedding
// rows (e.g. "siglip" + "qwen"), so the model must be specified.
func (r *EmbeddingsRepo) GetEmbedding(ctx context.Context, userID, mediaID int64, model string) (*domain.Embedding, error) {
	query, args, err := squirrel.
		Select(
			EmbMediaIDCol, EmbUserIDCol, EmbModelCol, EmbVecBytesCol,
			EmbStatusCol, EmbLastErrCol, EmbCreatedCol, EmbUpdatedCol,
		).
		From(EmbTableName).
		Where(squirrel.Eq{EmbMediaIDCol: mediaID, EmbUserIDCol: userID, EmbModelCol: model}).
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
		&emb.MediaID, &emb.UserID, &emb.Model, &emb.VecBytes,
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

func (r *EmbeddingsRepo) DeleteEmbedding(ctx context.Context, userID, mediaID int64) error {
	query, args, err := squirrel.
		Delete(EmbTableName).
		Where(squirrel.Eq{EmbMediaIDCol: mediaID, EmbUserIDCol: userID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build emb delete: %w", err)
	}

	q := db.Query{
		Name:     "Emb.Delete",
		QueryRaw: query,
	}

	// TODO: check tag for RowsAffected()
	_, err = r.db.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("exec emb delete: %w", err)
	}

	return nil
}

func (r *EmbeddingsRepo) MarkPending(ctx context.Context, userID, mediaID int64, model string) error {
	return r.updateStatus(ctx, userID, mediaID, model, "pending", "")
}

func (r *EmbeddingsRepo) MarkInIndex(ctx context.Context, userID, mediaID int64, model string) error {
	return r.updateStatus(ctx, userID, mediaID, model, "in_index", "")
}

func (r *EmbeddingsRepo) MarkFailed(ctx context.Context, userID, mediaID int64, model, msg string) error {
	return r.updateStatus(ctx, userID, mediaID, model, "failed", msg)
}

// updateStatus mutates exactly one row — the (media_id, user_id, model)
// triple that matches the composite PK. Without the model clause this would
// clobber every model's row for a given media (e.g. failing the qwen run
// would also mark siglip's 'in_index' row as 'failed').
func (r *EmbeddingsRepo) updateStatus(ctx context.Context, userID, mediaID int64, model, status, msg string) error {
	query, args, err := squirrel.
		Update(EmbTableName).
		Set(EmbStatusCol, status).
		Set(EmbLastErrCol, msg).
		Set(EmbUpdatedCol, time.Now().UTC()).
		Where(squirrel.Eq{EmbMediaIDCol: mediaID, EmbUserIDCol: userID, EmbModelCol: model}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build emb status update: %w", err)
	}

	q := db.Query{
		Name:     "Emb.UpdateStatus",
		QueryRaw: query,
	}

	// TODO: check tag for RowsAffected()
	if _, err := r.db.DB().ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("exec emb status update: %w", err)
	}
	return nil
}

// staleProcessingInterval defines how long a 'processing' row can sit before
// the retry sweep treats it as abandoned (worker crashed mid-batch). Kept
// well above any realistic single-batch embed latency so legitimate in-flight
// rows are never stolen.
const staleProcessingInterval = "5 minutes"

// ListUnindexed returns rows that need replay into the vector service:
//   - status IN ('pending','failed') — normal retry targets
//   - status = 'processing' AND updated_at older than staleProcessingInterval
//     — orphaned by a crashed worker
//
// userID=0 yields a global sweep across all users.
func (r *EmbeddingsRepo) ListUnindexed(ctx context.Context, userID int64, limit int) ([]worker.MediaRef, error) {
	statusClause := squirrel.Or{
		squirrel.Eq{EmbStatusCol: []string{"pending", "failed"}},
		squirrel.And{
			squirrel.Eq{EmbStatusCol: "processing"},
			squirrel.Expr(fmt.Sprintf("%s < NOW() - INTERVAL '%s'", EmbUpdatedCol, staleProcessingInterval)),
		},
	}

	qb := squirrel.
		Select(EmbMediaIDCol, EmbUserIDCol, EmbModelCol).
		From(EmbTableName).
		Where(statusClause).
		OrderBy(fmt.Sprintf("%s DESC", EmbUpdatedCol)).
		Limit(uint64(limit)).
		PlaceholderFormat(squirrel.Dollar)

	if userID > 0 {
		qb = qb.Where(squirrel.Eq{EmbUserIDCol: userID})
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list unindexed: %w", err)
	}

	q := db.Query{
		Name:     "Emb.ListUnindexed",
		QueryRaw: query,
	}

	rows, err := r.db.DB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query list unindexed: %w", err)
	}
	defer rows.Close()

	var refs []worker.MediaRef
	for rows.Next() {
		var ref worker.MediaRef
		if err := rows.Scan(&ref.MediaID, &ref.UserID, &ref.Model); err != nil {
			return nil, fmt.Errorf("scan unindexed ref: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

// ListUnembedded returns media that exist in the media table but have no
// corresponding row in the embeddings table — i.e. uploads that were never embedded.
func (r *EmbeddingsRepo) ListUnembedded(ctx context.Context, limit int) ([]worker.MediaRef, error) {
	raw := fmt.Sprintf(
		`SELECT m.id, m.user_id FROM %s m LEFT JOIN %s e ON m.id = e.media_id WHERE e.media_id IS NULL ORDER BY m.id LIMIT $1`,
		"media", EmbTableName,
	)

	q := db.Query{
		Name:     "Emb.ListUnembedded",
		QueryRaw: raw,
	}

	rows, err := r.db.DB().QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("query list unembedded: %w", err)
	}
	defer rows.Close()

	var refs []worker.MediaRef
	for rows.Next() {
		var ref worker.MediaRef
		if err := rows.Scan(&ref.MediaID, &ref.UserID); err != nil {
			return nil, fmt.Errorf("scan unembedded ref: %w", err)
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// ClaimForProcessing atomically claims the (media_id, model) row. Returns
// true when this worker now owns the row, false when another worker owns it
// or it's already indexed.
//
// Reclaimable states:
//   - row does not exist yet                      → INSERT
//   - status IN ('pending','failed')              → UPDATE to 'processing'
//   - status = 'processing' AND updated_at is older
//     than staleProcessingInterval (crashed worker) → UPDATE to 'processing'
//
// Blocked states:
//   - status = 'in_index'                                         → noop
//   - status = 'processing' with a recent updated_at (live owner) → noop
func (r *EmbeddingsRepo) ClaimForProcessing(ctx context.Context, userID, mediaID int64, model string) (bool, error) {
	now := time.Now().UTC()
	raw := fmt.Sprintf(`INSERT INTO embeddings (media_id, user_id, model, status, last_error, created_at, updated_at)
			VALUES ($1, $2, $3, 'processing', '', $4, $5)
			ON CONFLICT (media_id, model) DO UPDATE
			SET status = 'processing', updated_at = EXCLUDED.updated_at
			WHERE embeddings.status IN ('pending', 'failed')
			   OR (embeddings.status = 'processing' AND embeddings.updated_at < NOW() - INTERVAL '%s')`,
		staleProcessingInterval)

	q := db.Query{
		Name:     "Emb.ClaimForProcessing",
		QueryRaw: raw,
	}

	tag, err := r.db.DB().ExecContext(ctx, q, mediaID, userID, model, now, now)
	if err != nil {
		return false, fmt.Errorf("claim for processing: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *EmbeddingsRepo) GetEmbeddingBytes(ctx context.Context, userID, mediaID int64, model string) ([]byte, error) {
	emb, err := r.GetEmbedding(ctx, userID, mediaID, model)
	if err != nil {
		return nil, err
	}
	if emb == nil {
		return nil, fmt.Errorf("embedding not found for user=%d media=%d model=%s", userID, mediaID, model)
	}

	return emb.VecBytes, nil
}
