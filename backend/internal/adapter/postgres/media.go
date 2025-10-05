package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nurlan-nurlybay/AI_photo_retrieval_system/internal/domain"
)

type MediaPG struct {
	db               *pgxpool.Pool
	table            string
	uniqUserChecksum string // name of UNIQUE(user_id, checksum) constraint for error mapping
}

func NewMediaPG(db *pgxpool.Pool) *MediaPG {
	return &MediaPG{
		db:               db,
		table:            "media",
		uniqUserChecksum: "uq_media_user_checksum",
	}
}

var _ domain.MediaRepository = (*MediaPG)(nil)

// Create inserts a new media row.
func (r *MediaPG) Create(ctx context.Context, m *domain.Media) error {
	const q = `
		INSERT INTO media (
			id, user_id, url, thumb_url, mime_type, size_bytes, checksum,
			created_at, taken_at,
			meta_width, meta_height, meta_file_format,
			meta_camera_make, meta_camera_model, meta_software, meta_orientation
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`

	_, err := r.db.Exec(ctx, q,
		m.ID, m.UserID, m.URL, m.ThumbURL, m.MimeType, m.SizeBytes, m.Checksum,
		m.CreatedAt,
		m.Metadata.Width, m.Metadata.Height, m.Metadata.FileFormat,
		m.Metadata.CameraMake, m.Metadata.CameraModel, m.Metadata.Software, m.Metadata.Orientation,
	)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			// map unique(user_id, checksum) to already exists
			if pgErr.ConstraintName == r.uniqUserChecksum {
				return &domain.DomainError{Code: domain.ErrMediaAlreadyExists.Code, Msg: domain.ErrMediaAlreadyExists.Msg}
			}
		}
		return err
	}
	return nil
}

// Delete removes by user + id for multi-tenant safety.
func (r *MediaPG) Delete(ctx context.Context, userID domain.UserID, id domain.MediaID) error {
	q := `DELETE FROM ` + r.table + ` WHERE user_id = $1 AND id = $2`
	ct, err := r.db.Exec(ctx, q, userID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return &domain.DomainError{Code: domain.ErrMediaNotFound.Code, Msg: "media not found"}
	}
	return nil
}

// Get fetches a single media by user + id.
func (r *MediaPG) Get(ctx context.Context, userID domain.UserID, id domain.MediaID) (*domain.Media, error) {
	q := `
		SELECT
			id, user_id, url, thumb_url, mime_type, size_bytes, checksum,
			created_at, taken_at,
			meta_width, meta_height, meta_file_format,
			meta_camera_make, meta_camera_model, meta_software, meta_orientation
		FROM ` + r.table + `
		WHERE user_id = $1 AND id = $2
		LIMIT 1
	`
	row := r.db.QueryRow(ctx, q, userID, id)
	m, err := scanMedia(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.DomainError{Code: domain.ErrMediaNotFound.Code, Msg: "media not found"}
		}
		return nil, err
	}
	return m, nil
}

// GetByChecksum fetches by user + checksum for exact-byte dedup.
func (r *MediaPG) GetByChecksum(ctx context.Context, userID domain.UserID, checksum string) (*domain.Media, error) {
	q := `
		SELECT
			id, user_id, url, thumb_url, mime_type, size_bytes, checksum,
			created_at, taken_at,
			meta_width, meta_height, meta_file_format,
			meta_camera_make, meta_camera_model, meta_software, meta_orientation
		FROM ` + r.table + `
		WHERE user_id = $1 AND checksum = $2
		LIMIT 1
	`
	row := r.db.QueryRow(ctx, q, userID, checksum)
	m, err := scanMedia(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.DomainError{Code: domain.ErrMediaNotFound.Code, Msg: "media not found"}
		}
		return nil, err
	}
	return m, nil
}

// List returns paginated results and the total count.
func (r *MediaPG) List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error) {
	where, args := buildWhere(f)
	order := buildOrder(s)

	dataSQL := `
		SELECT
			id, user_id, url, thumb_url, mime_type, size_bytes, checksum,
			created_at, taken_at,
			meta_width, meta_height, meta_file_format,
			meta_camera_make, meta_camera_model, meta_software, meta_orientation
		FROM ` + r.table + `
	` + where + `
	` + order + `
		LIMIT $%d OFFSET $%d
	`
	// append limit/offset
	argPosLimit := len(args) + 1
	argPosOffset := len(args) + 2
	dataSQL = fmt.Sprintf(dataSQL, argPosLimit, argPosOffset)
	args = append(args, p.Limit, p.Offset)

	// run data query
	rows, err := r.db.Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]*domain.Media, 0, p.Limit)
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// run count query (same WHERE)
	cntSQL := `SELECT COUNT(1) FROM ` + r.table + ` ` + where
	var total int
	if err := r.db.QueryRow(ctx, cntSQL, args[:len(args)-2]...).Scan(&total); err != nil { // exclude limit/offset
		return nil, 0, err
	}

	return list, total, nil
}

/* -------------------- helpers -------------------- */

func scanMedia(row pgx.Row) (*domain.Media, error) {
	var (
		id, userID, url, thumbURL, mimeType, checksum string
		createdAt                                     time.Time
		takenAt                                       *time.Time
		width, height, orientation                    int
		sizeBytes                                     int64
		fileFormat, make, model, sw                   string
	)

	err := row.Scan(
		&id, &userID, &url, &thumbURL, &mimeType, &sizeBytes, &checksum,
		&createdAt, &takenAt,
		&width, &height, &fileFormat,
		&make, &model, &sw, &orientation,
	)
	if err != nil {
		return nil, err
	}

	m := &domain.Media{
		ID:        domain.MediaID(id),
		UserID:    domain.UserID(userID),
		URL:       url,
		ThumbURL:  thumbURL,
		MimeType:  mimeType,
		SizeBytes: sizeBytes,
		Checksum:  checksum,
		CreatedAt: createdAt.UTC(),
		Metadata: domain.Metadata{
			DateTimeOriginal: takenAt,
			Orientation:      orientation,
			Width:            width,
			Height:           height,
			FileFormat:       fileFormat,
			CameraMake:       make,
			CameraModel:      model,
			Software:         sw,
		},
	}
	return m, nil
}

// buildWhere translates MediaFilter into a WHERE clause and args.
func buildWhere(f domain.MediaFilter) (string, []any) {
	parts := make([]string, 0, 4)
	args := make([]any, 0, 4)
	i := 1

	if f.UserID != nil {
		parts = append(parts, fmt.Sprintf("user_id = $%d", i))
		args = append(args, *f.UserID)
		i++
	}
	if f.After != nil {
		parts = append(parts, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, f.After.UTC())
		i++
	}
	if f.Before != nil {
		parts = append(parts, fmt.Sprintf("created_at < $%d", i))
		args = append(args, f.Before.UTC())
		i++
	}
	if len(f.MimeLike) > 0 {
		likes := make([]string, 0, len(f.MimeLike))
		for _, v := range f.MimeLike {
			likes = append(likes, fmt.Sprintf("mime_type ILIKE $%d", i))
			args = append(args, v)
			i++
		}
		parts = append(parts, "("+strings.Join(likes, " OR ")+")")
	}

	if len(parts) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(parts, " AND "), args
}

// buildOrder whitelists sort fields to avoid SQL injection.
func buildOrder(s domain.Sort) string {
	field := "created_at"
	switch s.Field {
	case "created_at", "taken_at", "size_bytes":
		field = s.Field
	case "width":
		field = "meta_width"
	case "height":
		field = "meta_height"
	}
	dir := "ASC"
	if s.Desc {
		dir = "DESC"
	}
	return "ORDER BY " + field + " " + dir
}
