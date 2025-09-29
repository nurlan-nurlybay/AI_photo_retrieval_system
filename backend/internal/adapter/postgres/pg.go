package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

var ErrNoMedia = errors.New("no media found")

const findByIDsQuery = `
WITH ids AS (
  SELECT unnest($2::bigint[]) AS id, generate_series(1, array_length($2::bigint[],1)) AS ord
)
SELECT m.id, m.device_id, m.url, m.thumb_url, m.created_at, m.deleted
FROM media m
JOIN ids i ON m.id = i.id
WHERE m.device_id = $1
  AND m.deleted = false
ORDER BY i.ord;
`

const insertSQL = `INSERT INTO media (url, label) VALUES ($1, $2) RETURNING id`

type MediaRepository struct {
	pool *pgxpool.Pool
}

func NewMediaRepository(ctx context.Context, dsn string) (*MediaRepository, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	// Sensible pool defaults; tune for your env.
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	// Optional: prepare insert (pgx has statement cache anyway)
	// _, err = pool.Prepare(ctx, "media_insert", insertSQL)
	// if err != nil { pool.Close(); return nil, fmt.Errorf("prepare: %w", err) }

	return &MediaRepository{pool: pool}, nil
}

func (r *MediaRepository) Close() { r.pool.Close() }

func (r *MediaRepository) FindByIDs(ctx context.Context, deviceID string, ids []int64) ([]domain.Media, error) {
	if len(ids) == 0 {
		return []domain.Media{}, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(cctx, findByIDsQuery, deviceID, ids)
	if err != nil {
		return nil, fmt.Errorf("query FindByIDs: %w", err)
	}
	defer rows.Close()

	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Media, error) {
		var m domain.Media
		var thumb sql.NullString
		if err := row.Scan(&m.ID, &m.DeviceID, &m.URL, &thumb, &m.CreatedAt, &m.Deleted); err != nil {
			return domain.Media{}, err
		}
		if thumb.Valid {
			m.ThumbURL = thumb.String
		} else {
			m.ThumbURL = ""
		}
		return m, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan FindByIDs: %w", err)
	}
	if len(items) == 0 {
		return nil, ErrNoMedia
	}
	return items, nil
}

func (r *MediaRepository) InsertMediaMetadata(ctx context.Context, url, label string) (int64, error) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var id int64
	// If you prepared: r.pool.QueryRow(cctx, "media_insert", url, label).Scan(&id)
	if err := r.pool.QueryRow(cctx, insertSQL, url, label).Scan(&id); err != nil {
		return 0, wrapPgErr(err)
	}
	return id, nil
}

func wrapPgErr(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505": // unique_violation
			return fmt.Errorf("conflict: %s (%s)", pg.Message, pg.ConstraintName)
		case "23503": // foreign_key_violation
			return fmt.Errorf("foreign key: %s", pg.Message)
		case "23514": // check_violation
			return fmt.Errorf("check violation: %s", pg.Message)
		case "22001": // string_data_right_truncation
			return fmt.Errorf("value too long: %s", pg.Message)
		}
		return fmt.Errorf("pg %s: %s", pg.Code, pg.Message)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("no rows returned: %w", err)
	}
	return err
}
