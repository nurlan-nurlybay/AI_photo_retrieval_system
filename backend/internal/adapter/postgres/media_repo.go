package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		uniqUserChecksum: "media_user_checksum_uniq",
	}
}

var _ domain.MediaRepository = (*MediaPG)(nil)

func (r *MediaPG) Create(ctx context.Context, m *domain.Media) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Insert media and capture the generated ID
	const mediaQ = `
		INSERT INTO media (
			user_id, url, thumb_url, mime_type, size_bytes, checksum, created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id
	`
	err = tx.QueryRow(ctx, mediaQ,
		m.UserID, m.URL, m.ThumbURL, m.MimeType,
		m.SizeBytes, m.Checksum, m.CreatedAt,
	).Scan(&m.ID)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.ConstraintName == r.uniqUserChecksum {
			return &domain.DomainError{
				Code: domain.ErrMediaAlreadyExists.Code,
				Msg:  domain.ErrMediaAlreadyExists.Msg,
			}
		}
		return err
	}

	// Only insert metadata if it exists
	if m.Metadata.Width != 0 || m.Metadata.Height != 0 || m.Metadata.FileFormat != "" ||
		m.Metadata.CameraMake != "" || m.Metadata.CameraModel != "" || m.Metadata.Software != "" ||
		m.Metadata.DateTimeOriginal != nil {

		const metaQ = `
			INSERT INTO media_metadata (
				media_id, datetime_original, orientation, width, height, file_format,
				camera_make, camera_model, software
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`
		_, err = tx.Exec(ctx, metaQ,
			m.ID, m.Metadata.DateTimeOriginal, m.Metadata.Orientation, m.Metadata.Width,
			m.Metadata.Height, m.Metadata.FileFormat, m.Metadata.CameraMake,
			m.Metadata.CameraModel, m.Metadata.Software,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}


func (r *MediaPG) Delete(ctx context.Context, userID, id int64) error {
	const q = `
		DELETE FROM media
		WHERE id = $1 AND user_id = $2
	`

	cmd, err := r.db.Exec(ctx, q, id, userID)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return &domain.DomainError{Code: domain.ErrMediaNotFound.Code, Msg: domain.ErrMediaNotFound.Msg}
	}

	return nil
}

func (r *MediaPG) Get(ctx context.Context, userID, id int64) (*domain.Media, error) {
	const q = `
		SELECT 
			m.id, m.user_id, m.url, m.thumb_url, m.mime_type, m.size_bytes, 
			m.checksum, m.created_at,
			md.datetime_original, md.orientation, md.width, md.height, 
			md.file_format, md.camera_make, md.camera_model, md.software
		FROM media m
		LEFT JOIN media_metadata md ON m.id = md.media_id
		WHERE m.id = $1 AND m.user_id = $2
	`

	var media domain.Media
	var md domain.Metadata

	err := r.db.QueryRow(ctx, q, id, userID).Scan(
		&media.ID,
		&media.UserID,
		&media.URL,
		&media.ThumbURL,
		&media.MimeType,
		&media.SizeBytes,
		&media.Checksum,
		&media.CreatedAt,
		&md.DateTimeOriginal,
		&md.Orientation,
		&md.Width,
		&md.Height,
		&md.FileFormat,
		&md.CameraMake,
		&md.CameraModel,
		&md.Software,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.DomainError{Code: domain.ErrMediaNotFound.Code, Msg: domain.ErrMediaNotFound.Msg}
		}
		return nil, err
	}

	media.Metadata = md
	return &media, nil
}

// fetches by user + checksum for exact-byte dedup.
func (r *MediaPG) GetByChecksum(ctx context.Context, userID int64, checksum string) (*domain.Media, error) {
	const q = `
		SELECT 
			m.id, m.user_id, m.url, m.thumb_url, m.mime_type, m.size_bytes, 
			m.checksum, m.created_at,
			md.datetime_original, md.orientation, md.width, md.height, 
			md.file_format, md.camera_make, md.camera_model, md.software
		FROM media m
		LEFT JOIN media_metadata md ON m.id = md.media_id
		WHERE m.user_id = $1 AND m.checksum = $2
	`

	var media domain.Media
	var md domain.Metadata

	err := r.db.QueryRow(ctx, q, userID, checksum).Scan(
		&media.ID,
		&media.UserID,
		&media.URL,
		&media.ThumbURL,
		&media.MimeType,
		&media.SizeBytes,
		&media.Checksum,
		&media.CreatedAt,
		&md.DateTimeOriginal,
		&md.Orientation,
		&md.Width,
		&md.Height,
		&md.FileFormat,
		&md.CameraMake,
		&md.CameraModel,
		&md.Software,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.DomainError{Code: domain.ErrMediaNotFound.Code, Msg: domain.ErrMediaNotFound.Msg}
		}
		return nil, err
	}

	media.Metadata = md
	return &media, nil
}

// returns paginated results and the total count.
func (r *MediaPG) List(ctx context.Context, f domain.MediaFilter, p domain.Page, s domain.Sort) ([]*domain.Media, int, error) {
	shotExpr := "COALESCE(md.datetime_original, m.created_at)"

	// Whitelist sort fields to avoid SQL injection-by-config
	allowedSort := map[string]string{
		"":           "m.created_at", // default
		"created_at": "m.created_at",
		"id":         "m.id",
		"size_bytes": "m.size_bytes",
		"mime_type":  "m.mime_type",
		"shot_at":    shotExpr,
	}
	sortField, ok := allowedSort[s.Field]
	if !ok {
		sortField = allowedSort[""]
	}
	orderDir := "ASC"
	if s.Desc {
		orderDir = "DESC"
	}

	baseQ := strings.Builder{}
	baseQ.WriteString(`
		SELECT 
			m.id, m.user_id, m.url, m.thumb_url, m.mime_type, m.size_bytes, 
			m.checksum, m.created_at,
			md.datetime_original, md.orientation, md.width, md.height, 
			md.file_format, md.camera_make, md.camera_model, md.software
		FROM media m
		LEFT JOIN media_metadata md ON m.id = md.media_id
		WHERE 1=1
	`)

	countQ := strings.Builder{}
	countQ.WriteString(`
		SELECT COUNT(*)
		FROM media m
		LEFT JOIN media_metadata md ON m.id = md.media_id
		WHERE 1=1
	`)

	var args []any
	var countArgs []any
	argPos := 1

	addCond := func(cond string, val any) {
		baseQ.WriteString(" AND " + cond)
		countQ.WriteString(" AND " + cond)
		args = append(args, val)
		countArgs = append(countArgs, val)
		argPos++
	}

	// Filters
	if f.UserID != nil && *f.UserID != "" {
		addCond(fmt.Sprintf("m.user_id = $%d::bigint", argPos), *f.UserID)
	}
	if f.After != nil {
		addCond(fmt.Sprintf("%s >= $%d", shotExpr, argPos), *f.After)
	}
	if f.Before != nil {
		addCond(fmt.Sprintf("%s <= $%d", shotExpr, argPos), *f.Before)
	}
	if len(f.MimeLike) > 0 {
		// ILIKE ANY($n) expects text[]
		addCond(fmt.Sprintf("m.mime_type ILIKE ANY($%d)", argPos), f.MimeLike)
	}

	// Sorting
	baseQ.WriteString(fmt.Sprintf(" ORDER BY %s %s", sortField, orderDir))

	// Pagination
	if p.Limit > 0 {
		baseQ.WriteString(fmt.Sprintf(" LIMIT $%d", argPos))
		args = append(args, p.Limit)
		argPos++
	}
	if p.Offset > 0 {
		baseQ.WriteString(fmt.Sprintf(" OFFSET $%d", argPos))
		args = append(args, p.Offset)
		argPos++
	}

	// Run list query
	rows, err := r.db.Query(ctx, baseQ.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []*domain.Media
	for rows.Next() {
		var m domain.Media
		var md domain.Metadata
		err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.URL,
			&m.ThumbURL,
			&m.MimeType,
			&m.SizeBytes,
			&m.Checksum,
			&m.CreatedAt,
			&md.DateTimeOriginal,
			&md.Orientation,
			&md.Width,
			&md.Height,
			&md.FileFormat,
			&md.CameraMake,
			&md.CameraModel,
			&md.Software,
		)
		if err != nil {
			return nil, 0, err
		}
		m.Metadata = md
		out = append(out, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Total count (same filters, no limit/offset)
	var total int
	if err := r.db.QueryRow(ctx, countQ.String(), countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	return out, total, nil
}
