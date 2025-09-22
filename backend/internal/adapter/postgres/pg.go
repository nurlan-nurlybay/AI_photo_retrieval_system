package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

var ErrNoMedia = errors.New("no media found")

const findByIDsQuery = `
	SELECT id, device_id, url, thumb_url, created_at, deleted
	FROM media
	WHERE device_id = $1
	  AND id = ANY($2)
`

type MediaRepository struct {
	pool *pgxpool.Pool
}

func NewMediaRepository(ctx context.Context, dsn string) (*MediaRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &MediaRepository{pool: pool}, nil
}

func (r *MediaRepository) FindByIDs(ctx context.Context, deviceID string, ids []string) ([]domain.Media, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx, findByIDsQuery, deviceID, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("query FindByIDs: %w", err)
	}
	defer rows.Close()

	medias, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Media, error) {
		var m domain.Media
		if err := row.Scan(&m.ID, &m.DeviceID, &m.URL, &m.ThumbURL, &m.CreatedAt, &m.Deleted); err != nil {
			return domain.Media{}, err
		}
		return m, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan FindByIDs: %w", err)
	}
	if len(medias) == 0 {
		return nil, ErrNoMedia
	}
	return medias, nil
}
